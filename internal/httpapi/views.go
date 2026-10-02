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
		AreaID:          gate.AreaID,
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
	if command.ExecutionDuty != nil {
		view.ExecutionDuty = &contracts.ExecutionDutyView{
			ShiftID:      command.ExecutionDuty.ShiftID,
			ShiftName:    command.ExecutionDuty.ShiftName,
			AssignmentID: command.ExecutionDuty.AssignmentID,
			AreaID:       command.ExecutionDuty.AreaID,
			OperatorID:   command.ExecutionDuty.OperatorID,
			OperatorName: command.ExecutionDuty.OperatorName,
		}
	}
	return view
}

func operatorView(operator domain.Operator) contracts.OperatorView {
	return contracts.OperatorView{
		ID:        operator.ID,
		Name:      operator.Name,
		Qualified: operator.Active,
		CreatedAt: formatTime(operator.CreatedAt),
		UpdatedAt: formatTime(operator.UpdatedAt),
		Version:   operator.Version,
	}
}

func areaView(area domain.ControlArea) contracts.AreaView {
	return contracts.AreaView{
		ID:        area.ID,
		Name:      area.Name,
		GateIDs:   append([]string(nil), area.GateIDs...),
		CreatedAt: formatTime(area.CreatedAt),
		UpdatedAt: formatTime(area.UpdatedAt),
		Version:   area.Version,
	}
}

// shiftView computes the phase at view time so ended/scheduled shifts are
// visible without requiring the caller to compare timestamps.
func shiftView(shift domain.Shift, now time.Time) contracts.ShiftView {
	return contracts.ShiftView{
		ID:         shift.ID,
		Name:       shift.Name,
		Phase:      string(shift.PhaseAt(now)),
		StartsAt:   formatTime(shift.StartsAt),
		EndsAt:     formatTime(shift.EndsAt),
		EndedEarly: shift.EndedEarly,
		CreatedAt:  formatTime(shift.CreatedAt),
		UpdatedAt:  formatTime(shift.UpdatedAt),
		Version:    shift.Version,
	}
}

func assignmentView(assignment domain.ShiftAssignment) contracts.AssignmentView {
	return contracts.AssignmentView{
		ID:         assignment.ID,
		ShiftID:    assignment.ShiftID,
		AreaID:     assignment.AreaID,
		OperatorID: assignment.OperatorID,
		CreatedAt:  formatTime(assignment.CreatedAt),
		UpdatedAt:  formatTime(assignment.UpdatedAt),
		Version:    assignment.Version,
	}
}
