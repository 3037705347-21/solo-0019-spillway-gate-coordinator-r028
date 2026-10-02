package domain

import (
	"strings"
	"time"
)

// Operator is the registered control-room operator qualification record.
// Being registered does not mean the operator is currently on duty; that fact
// comes from a ShiftAssignment covering the gate's control area inside an
// active shift.
type Operator struct {
	ID        string
	Name      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int64
}

func NewOperator(id, name string, now time.Time) (Operator, error) {
	id, err := normalizeIdentifier(id, "invalid_operator_id")
	if err != nil {
		return Operator{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Operator{}, Invalid("invalid_operator_name", "operator name is required")
	}

	return Operator{
		ID:        id,
		Name:      name,
		Active:    true,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
		Version:   1,
	}, nil
}

func (o *Operator) Revoke(now time.Time) error {
	if !o.Active {
		return Conflict("operator_not_qualified", "operator qualification is not active")
	}
	o.Active = false
	o.UpdatedAt = now.UTC()
	o.Version++
	return nil
}

// ControlArea defines the responsibility scope a duty assignment covers.
// Gates are bound to at most one area through Gate.AreaID and the area's
// GateIDs membership list, which is frozen when the area is registered.
type ControlArea struct {
	ID        string
	Name      string
	GateIDs   []string
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int64
}

func NewControlArea(id, name string, gateIDs []string, now time.Time) (ControlArea, error) {
	id, err := normalizeIdentifier(id, "invalid_area_id")
	if err != nil {
		return ControlArea{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ControlArea{}, Invalid("invalid_area_name", "area name is required")
	}
	if len(gateIDs) == 0 {
		return ControlArea{}, Invalid("invalid_area_scope", "area must cover at least one gate")
	}

	seen := make(map[string]bool, len(gateIDs))
	normalized := make([]string, 0, len(gateIDs))
	for _, raw := range gateIDs {
		gateID := strings.TrimSpace(raw)
		if gateID == "" {
			return ControlArea{}, Invalid("invalid_area_scope", "area gate ids must not be empty")
		}
		if seen[gateID] {
			return ControlArea{}, Invalid("invalid_area_scope", "area gate ids must not repeat")
		}
		seen[gateID] = true
		normalized = append(normalized, gateID)
	}

	return ControlArea{
		ID:        id,
		Name:      name,
		GateIDs:   normalized,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
		Version:   1,
	}, nil
}

func (a ControlArea) CoversGate(gateID string) bool {
	for _, candidate := range a.GateIDs {
		if candidate == gateID {
			return true
		}
	}
	return false
}

type ShiftPhase string

const (
	ShiftScheduled ShiftPhase = "scheduled"
	ShiftActive    ShiftPhase = "active"
	ShiftEnded     ShiftPhase = "ended"
)

// Shift is a duty window. Shifts may overlap in clock time (a handover
// window), but an area must never be covered by two active shifts at the
// moment a movement is started.
type Shift struct {
	ID         string
	Name       string
	StartsAt   time.Time
	EndsAt     time.Time
	EndedEarly bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int64
}

func NewShift(id, name string, startsAt, endsAt time.Time, now time.Time) (Shift, error) {
	id, err := normalizeIdentifier(id, "invalid_shift_id")
	if err != nil {
		return Shift{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Shift{}, Invalid("invalid_shift_name", "shift name is required")
	}
	if !endsAt.After(startsAt) {
		return Shift{}, Invalid("invalid_shift_window", "shift end must be after shift start")
	}

	return Shift{
		ID:        id,
		Name:      name,
		StartsAt:  startsAt.UTC(),
		EndsAt:    endsAt.UTC(),
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
		Version:   1,
	}, nil
}

func (s Shift) PhaseAt(at time.Time) ShiftPhase {
	// An explicit early end is terminal even when it was placed before the
	// planned start (canceling a scheduled shift).
	if s.EndedEarly {
		return ShiftEnded
	}
	at = at.UTC()
	if at.Before(s.StartsAt) {
		return ShiftScheduled
	}
	if at.Before(s.EndsAt) {
		return ShiftActive
	}
	return ShiftEnded
}

// End closes a shift ahead of schedule. Both scheduled and active shifts can
// be explicitly closed; an already ended shift yields a stable conflict.
func (s *Shift) End(at time.Time) error {
	phase := s.PhaseAt(at)
	if phase == ShiftEnded {
		return Conflict("shift_already_ended", "shift is already ended")
	}

	now := at.UTC()
	if phase == ShiftScheduled {
		// Cancel a future shift: keep the window coherent by closing it at its
		// planned start.
		s.EndsAt = s.StartsAt
	} else {
		s.EndsAt = now
	}
	s.EndedEarly = true
	s.UpdatedAt = now
	s.Version++
	return nil
}

// ShiftAssignment binds one qualified operator to one control area for one
// shift. The uniqueness rules (one assignment per area+shift, one
// operator+area+shift) are enforced atomically by the repository.
type ShiftAssignment struct {
	ID         string
	ShiftID    string
	AreaID     string
	OperatorID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int64
}

func NewShiftAssignment(id, shiftID, areaID, operatorID string, now time.Time) (ShiftAssignment, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ShiftAssignment{}, Invalid("invalid_assignment_id", "assignment id is required")
	}
	shiftID = strings.TrimSpace(shiftID)
	if shiftID == "" {
		return ShiftAssignment{}, Invalid("invalid_assignment_shift", "shift id is required")
	}
	areaID = strings.TrimSpace(areaID)
	if areaID == "" {
		return ShiftAssignment{}, Invalid("invalid_assignment_area", "area id is required")
	}
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return ShiftAssignment{}, Invalid("invalid_assignment_operator", "operator id is required")
	}

	return ShiftAssignment{
		ID:         id,
		ShiftID:    shiftID,
		AreaID:     areaID,
		OperatorID: operatorID,
		CreatedAt:  now.UTC(),
		UpdatedAt:  now.UTC(),
		Version:    1,
	}, nil
}

// ExecutionDuty is the immutable on-duty fact captured on a command the
// moment its movement starts. It survives shift ends and qualification
// revocations so historical commands always show who was actually on duty.
type ExecutionDuty struct {
	ShiftID      string
	ShiftName    string
	AssignmentID string
	AreaID       string
	OperatorID   string
	OperatorName string
}

// DutyCoverageQuery gathers the full duty picture for a single movement
// attempt. ResolveExecutionDuty decides with a deterministic precedence:
//
//  1. the operator must be a registered, qualified operator;
//  2. the gate must belong to a control area;
//  3. the area must be covered by exactly one currently active shift
//     (unattended windows, ended shifts and overlapping handovers are all
//     explicit, distinct failures);
//  4. the requesting operator must be that shift's assigned operator for
//     the area.
type DutyCoverageQuery struct {
	OperatorID  string
	Operator    *Operator
	GateID      string
	GateAreaID  string
	Assignments []ShiftAssignment
	Shifts      map[string]Shift
	Areas       map[string]ControlArea
	At          time.Time
}

func ResolveExecutionDuty(query DutyCoverageQuery) (ExecutionDuty, error) {
	operatorID := strings.TrimSpace(query.OperatorID)
	if operatorID == "" {
		return ExecutionDuty{}, Invalid("invalid_operator", "operator is required")
	}
	if query.Operator == nil {
		return ExecutionDuty{}, NotFound("operator_not_found", "operator was not registered")
	}
	if query.Operator.ID != operatorID || !query.Operator.Active {
		return ExecutionDuty{}, Conflict("operator_not_qualified", "operator qualification is not active")
	}

	areaID := strings.TrimSpace(query.GateAreaID)
	if areaID == "" {
		return ExecutionDuty{}, Conflict("duty_unattended", "gate is not bound to a control area")
	}
	area, areaExists := query.Areas[areaID]
	if !areaExists || !area.CoversGate(query.GateID) {
		return ExecutionDuty{}, Conflict("duty_unattended", "gate is not covered by a control area")
	}

	var scheduled, ended, active []ShiftAssignment
	for _, assignment := range query.Assignments {
		if assignment.AreaID != areaID {
			continue
		}
		shift, exists := query.Shifts[assignment.ShiftID]
		if !exists {
			continue
		}
		switch shift.PhaseAt(query.At) {
		case ShiftActive:
			active = append(active, assignment)
		case ShiftScheduled:
			scheduled = append(scheduled, assignment)
		case ShiftEnded:
			ended = append(ended, assignment)
		}
	}

	if len(active) == 0 {
		switch {
		case len(ended) > 0:
			return ExecutionDuty{}, Conflict("duty_shift_ended", "the covering shift has ended")
		case len(scheduled) > 0:
			return ExecutionDuty{}, Conflict("duty_shift_not_started", "the covering shift has not started")
		default:
			return ExecutionDuty{}, Conflict("duty_unattended", "no on-duty assignment covers the gate right now")
		}
	}
	if len(active) > 1 {
		return ExecutionDuty{}, Conflict("duty_coverage_conflict", "multiple active shifts cover the gate; handover is ambiguous")
	}

	covering := active[0]
	coveringShift := query.Shifts[covering.ShiftID]
	if covering.OperatorID != operatorID {
		return ExecutionDuty{}, Conflict("operator_not_covering_gate", "operator is not assigned to the gate during the current shift")
	}

	return ExecutionDuty{
		ShiftID:      covering.ShiftID,
		ShiftName:    coveringShift.Name,
		AssignmentID: covering.ID,
		AreaID:       areaID,
		OperatorID:   operatorID,
		OperatorName: query.Operator.Name,
	}, nil
}
