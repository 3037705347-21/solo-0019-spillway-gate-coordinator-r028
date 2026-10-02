package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	"example.com/spillway-gate-coordinator/internal/domain"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	gates     map[string]domain.Gate
	commands  map[string]domain.GateCommand
	operators map[string]domain.Operator
	shifts    map[string]domain.DutyShift
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		gates:     make(map[string]domain.Gate),
		commands:  make(map[string]domain.GateCommand),
		operators: make(map[string]domain.Operator),
		shifts:    make(map[string]domain.DutyShift),
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
	operator string,
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

	// Resolve the live on-duty fact under the same lock that transitions the
	// command: unattended gates, handover contention, ended shifts and invalid
	// credentials must all be rejected before any state change.
	duty, err := m.resolveDuty(gate.ID, operator, now)
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

func (m *MemoryRepository) resolveDuty(gateID, operator string, now time.Time) (domain.DutyAttestation, error) {
	if operator == "" {
		return domain.DutyAttestation{}, domain.Invalid("invalid_operator", "operator is required")
	}

	credential, exists := m.operators[operator]
	if !exists {
		return domain.DutyAttestation{}, domain.NotFound("operator_not_found", "operator credential was not found")
	}
	if !credential.IsQualifiedAt(now) {
		return domain.DutyAttestation{}, domain.Conflict("operator_credential_invalid", "operator credential is revoked or expired")
	}

	return domain.ResolveDuty(gateID, operator, m.operators, m.shiftList(), now)
}

func (m *MemoryRepository) CreateOperator(_ context.Context, operator domain.Operator) error {
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
		return domain.Operator{}, domain.NotFound("operator_not_found", "operator credential was not found")
	}
	return operator, nil
}

func (m *MemoryRepository) RevokeOperator(_ context.Context, operatorID string, now time.Time) (domain.Operator, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	operator, exists := m.operators[operatorID]
	if !exists {
		return domain.Operator{}, domain.NotFound("operator_not_found", "operator credential was not found")
	}
	if err := operator.Revoke(now); err != nil {
		return domain.Operator{}, err
	}
	m.operators[operatorID] = operator
	return operator, nil
}

func (m *MemoryRepository) CreateShift(_ context.Context, shift domain.DutyShift) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.shifts[shift.ID]; exists {
		return domain.Conflict("duplicate_shift_id", "shift id already exists")
	}
	for _, gateID := range shift.GateIDs {
		if _, exists := m.gates[gateID]; !exists {
			return domain.NotFound("gate_not_found", "shift covers a gate id that was not found")
		}
	}
	for _, existing := range m.shifts {
		if existing.Overlaps(shift) {
			return domain.Conflict(
				"shift_handover_contention",
				"the new shift overlaps another shift on a covered gate; close the old shift first",
			)
		}
	}

	m.shifts[shift.ID] = shift
	return nil
}

func (m *MemoryRepository) Shift(_ context.Context, id string, now time.Time) (domain.DutyShift, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shift, exists := m.shifts[id]
	if !exists {
		return domain.DutyShift{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	return cloneShift(shift), nil
}

func (m *MemoryRepository) AssignShiftOperator(_ context.Context, shiftID, operatorID string, now time.Time) (domain.DutyShift, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shift, exists := m.shifts[shiftID]
	if !exists {
		return domain.DutyShift{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	operator, exists := m.operators[operatorID]
	if !exists {
		return domain.DutyShift{}, domain.NotFound("operator_not_found", "operator credential was not found")
	}
	if !operator.IsQualifiedAt(now) {
		return domain.DutyShift{}, domain.Conflict("operator_credential_invalid", "operator credential is revoked or expired")
	}

	if err := shift.AssignOperator(operatorID, now); err != nil {
		return domain.DutyShift{}, err
	}
	m.shifts[shiftID] = shift
	return cloneShift(shift), nil
}

func (m *MemoryRepository) EndShift(_ context.Context, shiftID string, now time.Time) (domain.DutyShift, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shift, exists := m.shifts[shiftID]
	if !exists {
		return domain.DutyShift{}, domain.NotFound("shift_not_found", "shift id was not found")
	}
	if err := shift.End(now); err != nil {
		return domain.DutyShift{}, err
	}
	m.shifts[shiftID] = shift
	return cloneShift(shift), nil
}

func (m *MemoryRepository) GateCoverage(_ context.Context, gateID string, now time.Time) (domain.GateCoverage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.gates[gateID]; !exists {
		return domain.GateCoverage{}, domain.NotFound("gate_not_found", "gate id was not found")
	}
	return domain.ResolveCoverage(gateID, m.shiftList(), now), nil
}

func (m *MemoryRepository) shiftList() []domain.DutyShift {
	shifts := make([]domain.DutyShift, 0, len(m.shifts))
	for _, shift := range m.shifts {
		shifts = append(shifts, cloneShift(shift))
	}
	sort.Slice(shifts, func(i, j int) bool { return shifts[i].ID < shifts[j].ID })
	return shifts
}

func cloneCommand(command domain.GateCommand) domain.GateCommand {
	if command.Interlock != nil {
		decision := *command.Interlock
		command.Interlock = &decision
	}
	if command.Duty != nil {
		duty := *command.Duty
		command.Duty = &duty
	}
	return command
}

func cloneShift(shift domain.DutyShift) domain.DutyShift {
	if shift.GateIDs != nil {
		shift.GateIDs = append([]string(nil), shift.GateIDs...)
	}
	if shift.Assignments != nil {
		shift.Assignments = append([]domain.ShiftAssignment(nil), shift.Assignments...)
	}
	return shift
}
