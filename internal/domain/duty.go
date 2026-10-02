package domain

import "time"

// DutyAttestation is the on-duty fact frozen onto a command at the instant its
// movement starts. It survives later shift end, reassignment and credential
// revocation, so historical commands keep showing who actually covered the post
// when they were executed.
type DutyAttestation struct {
	ShiftID    string
	OperatorID string
	GateID     string
	StartsAt   time.Time
	EndsAt     time.Time
	VerifiedAt time.Time
}

// GateCoverage is the resolved duty picture for one gate at a point in time.
type GateCoverage struct {
	GateID         string
	OpenShiftIDs   []string
	MemberShiftIDs []string
	ActiveShiftID  string
	Covered        bool
	Contended      bool
}

// ResolveCoverage computes the duty status of a gate from the full shift set.
//
// Because overlapping shifts covering the same gate are rejected at creation,
// at most one open (non-ended) shift can cover a gate. The contended branch is
// still evaluated here defensively so a handover race always yields an explicit
// result instead of an arbitrary winner.
func ResolveCoverage(gateID string, shifts []DutyShift, now time.Time) GateCoverage {
	coverage := GateCoverage{GateID: gateID}
	for _, shift := range shifts {
		if !shift.listsGate(gateID) {
			continue
		}
		if shift.StatusAt(now) == ShiftEnded {
			coverage.MemberShiftIDs = append(coverage.MemberShiftIDs, shift.ID)
			continue
		}
		if shift.Covers(gateID, now) {
			coverage.OpenShiftIDs = append(coverage.OpenShiftIDs, shift.ID)
		}
	}

	switch len(coverage.OpenShiftIDs) {
	case 0:
		// no open shift covers the gate: unattended
	case 1:
		coverage.ActiveShiftID = coverage.OpenShiftIDs[0]
		coverage.Covered = true
	default:
		coverage.Contended = true
	}
	return coverage
}

// ResolveDuty verifies that the operator may move the gate right now and
// returns the frozen attestation. Every failure gets a distinct stable code so
// unattended posts, handover contention, ended shifts and off-duty operators
// never collapse into an ambiguous rejection.
func ResolveDuty(
	gateID string,
	operatorID string,
	operators map[string]Operator,
	shifts []DutyShift,
	now time.Time,
) (DutyAttestation, error) {
	coverage := ResolveCoverage(gateID, shifts, now)

	if coverage.Contended {
		return DutyAttestation{}, Conflict("shift_handover_contention", "multiple active shifts cover the gate; handover must be completed first")
	}

	// Membership is judged from the shift roster first: the operator may be on
	// an ended crew even when a newer shift now owns the area, and that must be
	// reported as an ended shift rather than a generic off-duty rejection.
	var openMember *DutyShift
	var endedMemberID string
	for i := range shifts {
		shift := &shifts[i]
		if !shift.listsGate(gateID) || !shift.HasOperator(operatorID) {
			continue
		}
		if shift.StatusAt(now) == ShiftEnded {
			endedMemberID = shift.ID
			continue
		}
		if shift.Covers(gateID, now) && openMember == nil {
			openMember = shift
		}
	}

	if openMember != nil {
		operator, exists := operators[operatorID]
		if !exists {
			return DutyAttestation{}, NotFound("operator_not_found", "operator credential was not found")
		}
		if !operator.IsQualifiedAt(now) {
			return DutyAttestation{}, Conflict("operator_credential_invalid", "operator credential is revoked or expired")
		}
		return DutyAttestation{
			ShiftID:    openMember.ID,
			OperatorID: operatorID,
			GateID:     gateID,
			StartsAt:   openMember.StartsAt,
			EndsAt:     openMember.EffectiveEnd(),
			VerifiedAt: now.UTC(),
		}, nil
	}

	if endedMemberID != "" {
		return DutyAttestation{}, Conflict("shift_already_ended", "the operator's shift covering the gate has ended")
	}
	if !coverage.Covered {
		return DutyAttestation{}, Conflict("gate_unattended", "no active shift covers the gate")
	}
	return DutyAttestation{}, Conflict("operator_not_on_duty", "operator is not assigned to the active shift covering the gate")
}

func (s DutyShift) listsGate(gateID string) bool {
	for _, covered := range s.GateIDs {
		if covered == gateID {
			return true
		}
	}
	return false
}
