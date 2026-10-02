package storage

import (
	"context"
	"sync"
	"time"

	"example.com/spillway-gate-coordinator/internal/domain"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	gates       map[string]domain.Gate
	commands    map[string]domain.GateCommand
	operators   map[string]domain.Operator
	areas       map[string]domain.ControlArea
	shifts      map[string]domain.Shift
	assignments map[string]domain.ShiftAssignment
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		gates:       make(map[string]domain.Gate),
		commands:    make(map[string]domain.GateCommand),
		operators:   make(map[string]domain.Operator),
		areas:       make(map[string]domain.ControlArea),
		shifts:      make(map[string]domain.Shift),
		assignments: make(map[string]domain.ShiftAssignment),
	}
}

func (m *MemoryRepository) CreateGate(_ context.Context, gate domain.Gate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.gates[gate.ID]; exists {
		return domain.Conflict("duplicate_gate_id", "gate id already exists")
	}
	m.gates[gate.ID] = gate
	return nil
}

func (m *MemoryRepository) Gate(_ context.Context, id string) (domain.Gate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gate, exists := m.gates[id]
	if !exists {
		return domain.Gate{}, domain.NotFound("gate_not_found", "gate id was not found")
	}
	return gate, nil
}

func (m *MemoryRepository) CreateCommand(_ context.Context, command domain.GateCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	gate, exists := m.gates[command.GateID]
	if !exists {
		return domain.NotFound("gate_not_found", "gate id was not found")
	}
	if gate.Status == domain.GateFaulted {
		return domain.Conflict("gate_faulted", "faulted gates cannot receive new commands")
	}
	if command.StartingAperture != gate.Aperture {
		return domain.Conflict("gate_aperture_changed", "gate aperture changed before the command was stored")
	}
	if command.TargetAperture > gate.MaxAperture {
		return domain.Invalid("invalid_target_aperture", "target aperture is outside the gate limit")
	}
	if _, exists := m.commands[command.ID]; exists {
		return domain.Conflict("duplicate_command_id", "command id already exists")
	}

	m.commands[command.ID] = command
	return nil
}

func (m *MemoryRepository) Command(_ context.Context, id string) (domain.GateCommand, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	command, exists := m.commands[id]
	if !exists {
		return domain.GateCommand{}, domain.NotFound("command_not_found", "command id was not found")
	}
	return cloneCommand(command), nil
}

func (m *MemoryRepository) ReviewCommand(
	_ context.Context,
	commandID string,
	decision domain.InterlockDecision,
	token string,
	now time.Time,
) (domain.GateCommand, domain.Gate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	command, exists := m.commands[commandID]
	if !exists {
		return domain.GateCommand{}, domain.Gate{}, domain.NotFound("command_not_found", "command id was not found")
	}
	gate, exists := m.gates[command.GateID]
	if !exists {
		return domain.GateCommand{}, domain.Gate{}, domain.NotFound("gate_not_found", "gate id was not found")
	}

	nextCommand := cloneCommand(command)
	if err := nextCommand.Review(decision, token, now); err != nil {
		return domain.GateCommand{}, domain.Gate{}, err
	}

	nextGate := gate
	if decision.IsClear() {
		if err := nextGate.Reserve(nextCommand.ID, now); err != nil {
			return domain.GateCommand{}, domain.Gate{}, err
		}
	}

	m.commands[nextCommand.ID] = nextCommand
	m.gates[nextGate.ID] = nextGate
	return cloneCommand(nextCommand), nextGate, nil
}

func (m *MemoryRepository) ExecuteCommand(
	_ context.Context,
	commandID string,
	token string,
	operatorID string,
	simulateFault bool,
	now time.Time,
) (domain.GateCommand, domain.Gate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	command, exists := m.commands[commandID]
	if !exists {
		return domain.GateCommand{}, domain.Gate{}, domain.NotFound("command_not_found", "command id was not found")
	}
	gate, exists := m.gates[command.GateID]
	if !exists {
		return domain.GateCommand{}, domain.Gate{}, domain.NotFound("gate_not_found", "gate id was not found")
	}

	// Resolve the on-duty fact inside the same critical section as the state
	// transition, so a handover cannot interleave between coverage check and
	// movement start.
	var operator *domain.Operator
	if recorded, ok := m.operators[operatorID]; ok {
		operatorRecord := recorded
		operator = &operatorRecord
	}
	query := domain.DutyCoverageQuery{
		OperatorID: operatorID,
		Operator:   operator,
		GateID:     gate.ID,
		GateAreaID: gate.AreaID,
		Shifts:     m.shifts,
		Areas:      m.areas,
		At:         now,
	}
	for _, assignment := range m.assignments {
		query.Assignments = append(query.Assignments, assignment)
	}
	duty, err := domain.ResolveExecutionDuty(query)
	if err != nil {
		return domain.GateCommand{}, domain.Gate{}, err
	}

	nextCommand := cloneCommand(command)
	if err := nextCommand.BeginExecution(token, duty, now); err != nil {
		return domain.GateCommand{}, domain.Gate{}, err
	}

	nextGate := gate
	if err := nextGate.StartMovement(nextCommand.ID, now); err != nil {
		return domain.GateCommand{}, domain.Gate{}, err
	}

	if simulateFault {
		if err := nextGate.Settle(nextCommand.TargetAperture, true, now); err != nil {
			return domain.GateCommand{}, domain.Gate{}, err
		}
		if err := nextCommand.Abort("simulated_gate_fault", now); err != nil {
			return domain.GateCommand{}, domain.Gate{}, err
		}
	} else {
		if err := nextGate.Settle(nextCommand.TargetAperture, false, now); err != nil {
			return domain.GateCommand{}, domain.Gate{}, err
		}
		if err := nextCommand.Complete(now); err != nil {
			return domain.GateCommand{}, domain.Gate{}, err
		}
	}

	m.commands[nextCommand.ID] = nextCommand
	m.gates[nextGate.ID] = nextGate
	return cloneCommand(nextCommand), nextGate, nil
}

func (m *MemoryRepository) RegisterOperator(_ context.Context, operator domain.Operator) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.operators[operator.ID]; exists {
		return domain.Conflict("duplicate_operator_id", "operator id already exists")
	}
	m.operators[operator.ID] = operator
	return nil
}

func (m *MemoryRepository) Operator(_ context.Context, id string) (domain.Operator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	operator, exists := m.operators[id]
	if !exists {
		return domain.Operator{}, domain.NotFound("operator_not_found", "operator id was not found")
	}
	return operator, nil
}

func (m *MemoryRepository) RevokeOperator(_ context.Context, id string, now time.Time) (domain.Operator, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	operator, exists := m.operators[id]
	if !exists {
		return domain.Operator{}, domain.NotFound("operator_not_found", "operator id was not found")
	}
	if err := operator.Revoke(now); err != nil {
		return domain.Operator{}, err
	}
	m.operators[id] = operator
	return operator, nil
}

func (m *MemoryRepository) CreateArea(_ context.Context, area domain.ControlArea) (domain.ControlArea, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.areas[area.ID]; exists {
		return domain.ControlArea{}, domain.Conflict("duplicate_area_id", "area id already exists")
	}

	// Bind every listed gate atomically; any unknown or already-bound gate
	// rejects the whole area and leaves no gate changed.
	boundGates := make(map[string]domain.Gate, len(area.GateIDs))
	for _, gateID := range area.GateIDs {
		gate, exists := m.gates[gateID]
		if !exists {
			return domain.ControlArea{}, domain.NotFound("gate_not_found", "area references a gate that does not exist")
		}
		if gate.AreaID != "" {
			return domain.ControlArea{}, domain.Conflict("gate_already_assigned_area", "gate is already bound to another control area")
		}
		if err := gate.BindArea(area.ID, area.CreatedAt); err != nil {
			return domain.ControlArea{}, err
		}
		boundGates[gateID] = gate
	}

	m.areas[area.ID] = area
	for gateID, gate := range boundGates {
		m.gates[gateID] = gate
	}
	return area, nil
}

func (m *MemoryRepository) Area(_ context.Context, id string) (domain.ControlArea, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	area, exists := m.areas[id]
	if !exists {
		return domain.ControlArea{}, domain.NotFound("area_not_found", "area id was not found")
	}
	return cloneArea(area), nil
}

func (m *MemoryRepository) CreateShift(_ context.Context, shift domain.Shift) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.shifts[shift.ID]; exists {
		return domain.Conflict("duplicate_shift_id", "shift id already exists")
	}
	m.shifts[shift.ID] = shift
	return nil
}

func (m *MemoryRepository) Shift(_ context.Context, id string) (domain.Shift, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shift, exists := m.shifts[id]
	if !exists {
		return domain.Shift{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	return shift, nil
}

func (m *MemoryRepository) EndShift(_ context.Context, id string, now time.Time) (domain.Shift, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shift, exists := m.shifts[id]
	if !exists {
		return domain.Shift{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	if err := shift.End(now); err != nil {
		return domain.Shift{}, err
	}
	m.shifts[id] = shift
	return shift, nil
}

func (m *MemoryRepository) CreateAssignment(
	_ context.Context,
	assignment domain.ShiftAssignment,
) (domain.ShiftAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.assignments[assignment.ID]; exists {
		return domain.ShiftAssignment{}, domain.Conflict("duplicate_assignment_id", "assignment id already exists")
	}
	if _, exists := m.shifts[assignment.ShiftID]; !exists {
		return domain.ShiftAssignment{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	if _, exists := m.areas[assignment.AreaID]; !exists {
		return domain.ShiftAssignment{}, domain.NotFound("area_not_found", "area id was not found")
	}
	operator, exists := m.operators[assignment.OperatorID]
	if !exists {
		return domain.ShiftAssignment{}, domain.NotFound("operator_not_found", "operator id was not found")
	}
	if !operator.Active {
		return domain.ShiftAssignment{}, domain.Conflict("operator_not_qualified", "operator qualification is not active")
	}

	for _, existing := range m.assignments {
		if existing.ShiftID != assignment.ShiftID || existing.AreaID != assignment.AreaID {
			continue
		}
		if existing.OperatorID == assignment.OperatorID {
			return domain.ShiftAssignment{}, domain.Conflict(
				"duplicate_assignment",
				"operator is already assigned to this area for the shift",
			)
		}
		return domain.ShiftAssignment{}, domain.Conflict(
			"duplicate_area_assignment",
			"the area already has a different operator assigned for this shift",
		)
	}

	m.assignments[assignment.ID] = assignment
	return assignment, nil
}

func (m *MemoryRepository) Assignment(_ context.Context, id string) (domain.ShiftAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assignment, exists := m.assignments[id]
	if !exists {
		return domain.ShiftAssignment{}, domain.NotFound("assignment_not_found", "assignment id was not found")
	}
	return assignment, nil
}

func cloneCommand(command domain.GateCommand) domain.GateCommand {
	if command.Interlock != nil {
		decision := *command.Interlock
		command.Interlock = &decision
	}
	if command.ExecutionDuty != nil {
		duty := *command.ExecutionDuty
		command.ExecutionDuty = &duty
	}
	return command
}

func cloneArea(area domain.ControlArea) domain.ControlArea {
	area.GateIDs = append([]string(nil), area.GateIDs...)
	return area
}
