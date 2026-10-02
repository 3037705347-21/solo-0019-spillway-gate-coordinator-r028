package domain

import (
	"sort"
	"strings"
	"time"
)

type ShiftStatus string

const (
	ShiftScheduled ShiftStatus = "scheduled"
	ShiftActive    ShiftStatus = "active"
	ShiftEnded     ShiftStatus = "ended"
)

// ShiftAssignment places one qualified operator on duty for one shift.
type ShiftAssignment struct {
	OperatorID string
	AssignedAt time.Time
}

// DutyShift is one duty period maintained by the duty supervisor. It names the
// operators on duty and the gate area (gate ids) the shift is responsible for.
type DutyShift struct {
	ID          string
	Label       string
	StartsAt    time.Time
	EndsAt      time.Time
	GateIDs     []string
	Assignments []ShiftAssignment
	EndedAt     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int64
}

func NewDutyShift(
	id string,
	label string,
	startsAt time.Time,
	endsAt time.Time,
	gateIDs []string,
	now time.Time,
) (DutyShift, error) {
	id, err := validateIdentifier(id, "invalid_shift_id")
	if err != nil {
		return DutyShift{}, err
	}

	startsAt = startsAt.UTC()
	endsAt = endsAt.UTC()
	if startsAt.IsZero() || endsAt.IsZero() {
		return DutyShift{}, Invalid("invalid_shift_window", "starts_at and ends_at are required")
	}
	if !endsAt.After(startsAt) {
		return DutyShift{}, Invalid("invalid_shift_window", "ends_at must be after starts_at")
	}
	if !endsAt.After(now.UTC()) {
		return DutyShift{}, Invalid("shift_already_ended", "the shift window must end in the future")
	}

	gates, err := normalizeGateIDs(gateIDs)
	if err != nil {
		return DutyShift{}, err
	}

	return DutyShift{
		ID:        id,
		Label:     strings.TrimSpace(label),
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		GateIDs:   gates,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
		Version:   1,
	}, nil
}

func normalizeGateIDs(gateIDs []string) ([]string, error) {
	if len(gateIDs) == 0 {
		return nil, Invalid("invalid_coverage_area", "at least one covered gate is required")
	}
	seen := make(map[string]struct{}, len(gateIDs))
	normalized := make([]string, 0, len(gateIDs))
	for _, raw := range gateIDs {
		gateID, err := validateIdentifier(raw, "invalid_gate_reference")
		if err != nil {
			return nil, err
		}
		if _, dup := seen[gateID]; dup {
			return nil, Invalid("duplicate_gate_coverage", "a shift cannot list the same gate more than once")
		}
		seen[gateID] = struct{}{}
		normalized = append(normalized, gateID)
	}
	sort.Strings(normalized)
	return normalized, nil
}

// StatusAt derives the lifecycle status at a given moment. An early end always
// wins over the nominal window.
func (s DutyShift) StatusAt(at time.Time) ShiftStatus {
	at = at.UTC()
	if !s.EndedAt.IsZero() && !at.Before(s.EndedAt) {
		return ShiftEnded
	}
	if at.Before(s.StartsAt) {
		return ShiftScheduled
	}
	if !at.Before(s.EndsAt) {
		return ShiftEnded
	}
	return ShiftActive
}

// EffectiveEnd is the real end of the coverage window: the early-end timestamp
// when the shift was closed ahead of time, otherwise the nominal end.
func (s DutyShift) EffectiveEnd() time.Time {
	if !s.EndedAt.IsZero() && s.EndedAt.Before(s.EndsAt) {
		return s.EndedAt
	}
	return s.EndsAt
}

// Covers reports whether the shift is responsible for the gate at the given
// moment, using half-open [starts_at, effective_end) boundaries.
func (s DutyShift) Covers(gateID string, at time.Time) bool {
	at = at.UTC()
	if at.Before(s.StartsAt) || !at.Before(s.EffectiveEnd()) {
		return false
	}
	for _, covered := range s.GateIDs {
		if covered == gateID {
			return true
		}
	}
	return false
}

// AssignOperator rosters one operator on the shift. Rostering a future shift is
// allowed so the incoming crew can be prepared before handover.
func (s *DutyShift) AssignOperator(operatorID string, now time.Time) error {
	if s.StatusAt(now) == ShiftEnded {
		return Conflict("shift_already_ended", "operators cannot be assigned to an ended shift")
	}
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return Invalid("invalid_operator", "operator id is required")
	}
	for _, assignment := range s.Assignments {
		if assignment.OperatorID == operatorID {
			return Conflict("duplicate_shift_assignment", "operator is already assigned to the shift")
		}
	}

	s.Assignments = append(s.Assignments, ShiftAssignment{
		OperatorID: operatorID,
		AssignedAt: now.UTC(),
	})
	s.UpdatedAt = now.UTC()
	s.Version++
	return nil
}

// End closes the shift immediately. It is idempotent only in the sense that an
// already ended shift yields a stable conflict rather than a silent rewrite.
func (s *DutyShift) End(now time.Time) error {
	now = now.UTC()
	if s.StatusAt(now) == ShiftEnded {
		return Conflict("shift_already_ended", "the shift has already ended")
	}
	s.EndedAt = now
	s.UpdatedAt = now
	s.Version++
	return nil
}

// HasOperator reports whether an operator is rostered on the shift.
func (s DutyShift) HasOperator(operatorID string) bool {
	for _, assignment := range s.Assignments {
		if assignment.OperatorID == operatorID {
			return true
		}
	}
	return false
}

// Overlaps reports whether two shifts claim any common gate over overlapping
// effective windows. Half-open intervals touching only at an endpoint are not
// an overlap, so a nominal handover is legal while a contested handover is not.
func (s DutyShift) Overlaps(other DutyShift) bool {
	if !gateSetsIntersect(s.GateIDs, other.GateIDs) {
		return false
	}
	return s.StartsAt.Before(other.EffectiveEnd()) && other.StartsAt.Before(s.EffectiveEnd())
}

func gateSetsIntersect(left, right []string) bool {
	for _, gateID := range left {
		for _, other := range right {
			if gateID == other {
				return true
			}
		}
	}
	return false
}
