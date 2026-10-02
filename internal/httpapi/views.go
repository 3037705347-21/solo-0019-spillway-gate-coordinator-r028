package httpapi

import (
	"time"

	"example.com/spillway-gate-coordinator/internal/contracts"
	"example.com/spillway-gate-coordinator/internal/domain"
)

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func gateView(gate domain.Gate) contracts.GateView {
	return contracts.GateView{
		ID:              gate.ID,
		Name:            gate.Name,
		Status:          string(gate.Status),
		Aperture:        gate.Aperture,
		MaxAperture:     gate.MaxAperture,
		ActiveCommandID: gate.ActiveCommandID,
		UpdatedAt:       formatTime(gate.UpdatedAt),
		Version:         gate.Version,
	}
}

func commandView(command domain.GateCommand, includeToken bool) contracts.CommandView {
	view := contracts.CommandView{
		ID:               command.ID,
		GateID:           command.GateID,
		TargetAperture:   command.TargetAperture,
		StartingAperture: command.StartingAperture,
		Reason:           command.Reason,
		RequestedBy:      command.RequestedBy,
		Status:           string(command.Status),
		ExecutedBy:       command.ExecutedBy,
		TerminalReason:   command.TerminalReason,
		CreatedAt:        formatTime(command.CreatedAt),
		UpdatedAt:        formatTime(command.UpdatedAt),
		Version:          command.Version,
	}

	if includeToken && command.Status == domain.CommandApproved {
		view.ExecutionToken = command.ExecutionToken
	}
	if command.Interlock != nil {
		view.Interlock = &contracts.InterlockDecisionView{
			Verdict:    string(command.Interlock.Verdict),
			Verifier:   command.Interlock.Verifier,
			Note:       command.Interlock.Note,
			ReviewedAt: formatTime(command.Interlock.ReviewedAt),
		}
	}
	if command.Duty != nil {
		view.Duty = &contracts.DutyAttestationView{
			ShiftID:    command.Duty.ShiftID,
			OperatorID: command.Duty.OperatorID,
			GateID:     command.Duty.GateID,
			StartsAt:   formatTime(command.Duty.StartsAt),
			EndsAt:     formatTime(command.Duty.EndsAt),
			VerifiedAt: formatTime(command.Duty.VerifiedAt),
		}
	}
	return view
}

func operatorView(operator domain.Operator) contracts.OperatorView {
	view := contracts.OperatorView{
		ID:           operator.ID,
		Name:         operator.Name,
		Status:       string(operator.Status),
		RegisteredAt: formatTime(operator.RegisteredAt),
		UpdatedAt:    formatTime(operator.UpdatedAt),
		Version:      operator.Version,
	}
	if !operator.ValidUntil.IsZero() {
		view.ValidUntil = formatTime(operator.ValidUntil)
	}
	return view
}

func shiftView(shift domain.DutyShift, now time.Time) contracts.ShiftView {
	assignments := make([]contracts.ShiftAssignmentView, 0, len(shift.Assignments))
	for _, assignment := range shift.Assignments {
		assignments = append(assignments, contracts.ShiftAssignmentView{
			OperatorID: assignment.OperatorID,
			AssignedAt: formatTime(assignment.AssignedAt),
		})
	}

	view := contracts.ShiftView{
		ID:          shift.ID,
		Label:       shift.Label,
		Status:      string(shift.StatusAt(now)),
		StartsAt:    formatTime(shift.StartsAt),
		EndsAt:      formatTime(shift.EndsAt),
		GateIDs:     shift.GateIDs,
		Assignments: assignments,
		CreatedAt:   formatTime(shift.CreatedAt),
		UpdatedAt:   formatTime(shift.UpdatedAt),
		Version:     shift.Version,
	}
	if !shift.EndedAt.IsZero() {
		view.EndedAt = formatTime(shift.EndedAt)
	}
	return view
}

func coverageView(coverage domain.GateCoverage, now time.Time) contracts.GateCoverageView {
	status := "unattended"
	if coverage.Contended {
		status = "contended"
	} else if coverage.Covered {
		status = "covered"
	}
	openShiftIDs := coverage.OpenShiftIDs
	if openShiftIDs == nil {
		openShiftIDs = []string{}
	}
	memberShiftIDs := coverage.MemberShiftIDs
	if memberShiftIDs == nil {
		memberShiftIDs = []string{}
	}
	return contracts.GateCoverageView{
		GateID:         coverage.GateID,
		Status:         status,
		Covered:        coverage.Covered,
		Contended:      coverage.Contended,
		ActiveShiftID:  coverage.ActiveShiftID,
		OpenShiftIDs:   openShiftIDs,
		MemberShiftIDs: memberShiftIDs,
		CheckedAt:      formatTime(now),
	}
}
