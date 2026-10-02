package main

import (
	"fmt"

	"net/http"
	"time"
)

// runDutyCoverage exercises the shift assignment and post coverage capability.
// It proves the explicit outcomes required for unattended windows, ended
// shifts, handover contention, duplicate assignments, revoked qualifications
// and the preservation of the on-duty fact on a historical command.
func runDutyCoverage(client *apiClient) error {
	now := time.Now().UTC()

	if err := setupCoverageScenario(client, now); err != nil {
		return err
	}

	if err := verifyDuplicateAssignments(client); err != nil {
		return err
	}
	if err := verifyUnattendedExecution(client); err != nil {
		return err
	}
	if err := verifyNotOnDutyOperator(client, now); err != nil {
		return err
	}
	if err := verifyEndedShiftExecution(client, now); err != nil {
		return err
	}
	if err := verifyCancelScheduledShift(client); err != nil {
		return err
	}
	if err := verifyHandoverContention(client, now); err != nil {
		return err
	}
	if err := verifyRevokedQualification(client); err != nil {
		return err
	}
	if err := verifyFaultedDutySnapshot(client); err != nil {
		return err
	}
	if err := verifyHistoricalDutySnapshot(client, now); err != nil {
		return err
	}

	fmt.Println("duty-coverage: unattended, ended, contention, duplicate and revoked attempts explicitly resolved; duty snapshot preserved")
	return nil
}

func setupCoverageScenario(client *apiClient, now time.Time) error {
	for _, gate := range []struct {
		id   string
		name string
	}{
		{"gate-a1", "A1 闸"},
		{"gate-a2", "A2 闸"},
		{"gate-b1", "B1 闸"},
		{"gate-c1", "C1 闸"},
		{"gate-d1", "D1 闸"},
		{"gate-e1", "E1 闸"},
		{"gate-f1", "F1 闸"},
		{"gate-g1", "G1 闸"},
	} {
		if _, err := fixtureGate(client, gate.id, gate.name, 4, 0.5); err != nil {
			return err
		}
	}

	for _, operator := range []struct {
		id, name string
	}{
		{"op-chen", "陈晨"},
		{"op-lin", "林林"},
		{"op-ma", "马玛"},
		{"op-qin", "秦勤"},
		{"op-other", "其他操作员"},
		{"op-revoked", "待注销"},
	} {
		if err := fixtureOperator(client, operator.id, operator.name); err != nil {
			return err
		}
	}

	areas := []struct {
		id, name string
		gates    []string
	}{
		{"area-a", "A 区", []string{"gate-a1", "gate-a2"}},
		{"area-b", "B 区", []string{"gate-b1"}},
		{"area-c", "C 区", []string{"gate-c1"}},
		{"area-d", "D 区", []string{"gate-d1"}},
		{"area-e", "E 区", []string{"gate-e1"}},
		{"area-f", "F 区", []string{"gate-f1"}},
		{"area-g", "G 区", []string{"gate-g1"}},
	}
	for _, area := range areas {
		if err := fixtureArea(client, area.id, area.name, area.gates); err != nil {
			return err
		}
	}

	// gate-e1 intentionally gets no assignment: it stays inside an area but
	// unattended.
	activeShifts := []struct {
		id, name string
	}{
		{"shift-a", "A 区当班"},
		{"shift-b", "B 区当班"},
		{"shift-c-old", "C 区旧班"},
		{"shift-d1", "D 区交班一"},
		{"shift-d2", "D 区交班二"},
	}
	for _, shift := range activeShifts {
		if err := fixtureShift(client, shift.id, shift.name, nil, nil); err != nil {
			return err
		}
	}

	// A future (scheduled) shift exists for area A; it must not count as
	// coverage yet.
	futureStart := now.Add(time.Hour)
	futureEnd := now.Add(2 * time.Hour)
	if err := fixtureShift(client, "shift-a-future", "A 区预约班", &futureStart, &futureEnd); err != nil {
		return err
	}
	// Area G only has a scheduled future shift, which gets canceled before it
	// starts in the verification step.
	if err := fixtureShift(client, "shift-g-future", "G 区预约班", &futureStart, &futureEnd); err != nil {
		return err
	}

	assignments := []struct {
		shift, area, operator string
	}{
		{"shift-a", "area-a", "op-chen"},
		{"shift-b", "area-b", "op-ma"},
		{"shift-c-old", "area-c", "op-qin"},
		{"shift-d1", "area-d", "op-lin"},
		{"shift-d2", "area-d", "op-ma"},
		{"shift-a-future", "area-a", "op-lin"},
		{"shift-a", "area-f", "op-chen"},
		{"shift-g-future", "area-g", "op-qin"},
	}
	for _, assignment := range assignments {
		if _, err := fixtureAssignment(client, assignment.shift, assignment.area, assignment.operator); err != nil {
			return err
		}
	}
	return nil
}

func verifiedApprovedCommand(client *apiClient, gateID string) (string, string, error) {
	command, err := fixtureCommand(client, gateID, 1.5, "值班覆盖验证", "dispatcher-zhao")
	if err != nil {
		return "", "", err
	}
	commandID, err := stringField(command, "id")
	if err != nil {
		return "", "", err
	}
	approved, err := fixtureClearReview(client, commandID, "safety-he")
	if err != nil {
		return "", "", err
	}
	token, err := stringField(approved, "execution_token")
	if err != nil {
		return "", "", err
	}
	return commandID, token, nil
}

func attemptExecute(client *apiClient, commandID, token, operatorID string, simulate bool) apiResult {
	result, _ := client.post("/commands/"+commandID+"/execute", map[string]any{
		"operator_id":    operatorID,
		"token":          token,
		"simulate_fault": simulate,
	})
	return result
}

func verifyDuplicateAssignments(client *apiClient) error {
	// Same area + shift with a different operator is a post-contention
	// conflict.
	dupArea, err := client.post("/assignments", map[string]any{
		"shift_id":    "shift-a",
		"area_id":     "area-a",
		"operator_id": "op-other",
	})
	if err != nil {
		return err
	}
	if err := requireStatus(dupArea, http.StatusConflict, "reject duplicate area assignment"); err != nil {
		return err
	}
	if err := expectErrorCode(dupArea, "duplicate_area_assignment", "duplicate area assignment code"); err != nil {
		return err
	}

	// Same operator + area + shift again is an explicit repeated allocation.
	dupExact, err := client.post("/assignments", map[string]any{
		"shift_id":    "shift-a",
		"area_id":     "area-a",
		"operator_id": "op-chen",
	})
	if err != nil {
		return err
	}
	if err := requireStatus(dupExact, http.StatusConflict, "reject repeated operator assignment"); err != nil {
		return err
	}
	if err := expectErrorCode(dupExact, "duplicate_assignment", "repeated assignment code"); err != nil {
		return err
	}

	// Assigning a revoked qualification is refused.
	if _, err := client.post("/operators/op-revoked/revoke", nil); err != nil {
		return err
	}
	revokedAssignment, err := client.post("/assignments", map[string]any{
		"shift_id":    "shift-a",
		"area_id":     "area-b",
		"operator_id": "op-revoked",
	})
	if err != nil {
		return err
	}
	if err := requireStatus(revokedAssignment, http.StatusConflict, "reject revoked operator assignment"); err != nil {
		return err
	}
	if err := expectErrorCode(revokedAssignment, "operator_not_qualified", "revoked assignment code"); err != nil {
		return err
	}
	return nil
}

func verifyUnattendedExecution(client *apiClient) error {
	// gate-e1 belongs to area-e which has no assignment at all.
	commandID, token, err := verifiedApprovedCommand(client, "gate-e1")
	if err != nil {
		return err
	}
	result := attemptExecute(client, commandID, token, "op-chen", false)
	if err := requireStatus(result, http.StatusConflict, "reject unattended execution"); err != nil {
		return err
	}
	if err := expectErrorCode(result, "duty_unattended", "unattended execution code"); err != nil {
		return err
	}

	// The failed attempt must leave command and gate untouched: the command is
	// still approved and its token still works later once coverage exists.
	fetched, err := client.get("/commands/" + commandID)
	if err != nil {
		return err
	}
	if err := requireStatus(fetched, http.StatusOK, "read unattended command"); err != nil {
		return err
	}
	command, err := objectField(fetched.Body, "command")
	if err != nil {
		return err
	}
	status, err := stringField(command, "status")
	if err != nil {
		return err
	}
	if status != "approved" {
		return fmt.Errorf("unattended attempt changed command status to %q", status)
	}

	// Assign coverage for area-e, then the same approved command executes.
	if _, err := fixtureAssignment(client, "shift-b", "area-e", "op-chen"); err != nil {
		return err
	}
	result = attemptExecute(client, commandID, token, "op-chen", false)
	if err := requireStatus(result, http.StatusOK, "execute once coverage assigned"); err != nil {
		return err
	}
	return nil
}

func verifyNotOnDutyOperator(client *apiClient, now time.Time) error {
	// The only active coverage for area A is op-chen; op-lin only holds the
	// scheduled future shift, so op-lin is not the current duty operator.
	commandID, token, err := verifiedApprovedCommand(client, "gate-a1")
	if err != nil {
		return err
	}
	result := attemptExecute(client, commandID, token, "op-lin", false)
	if err := requireStatus(result, http.StatusConflict, "reject non-covering operator"); err != nil {
		return err
	}
	if err := expectErrorCode(result, "operator_not_covering_gate", "non-covering operator code"); err != nil {
		return err
	}

	// Unknown operator identity is a distinct not-found result.
	missing := attemptExecute(client, commandID, token, "op-nobody", false)
	if err := requireStatus(missing, http.StatusNotFound, "reject unregistered operator"); err != nil {
		return err
	}
	if err := expectErrorCode(missing, "operator_not_found", "unregistered operator code"); err != nil {
		return err
	}

	// The assigned operator succeeds.
	result = attemptExecute(client, commandID, token, "op-chen", false)
	if err := requireStatus(result, http.StatusOK, "covering operator executes"); err != nil {
		return err
	}
	return nil
}

func verifyEndedShiftExecution(client *apiClient, now time.Time) error {
	// End area C's only shift, then its gate becomes unattended because the
	// covering shift ended.
	endResult, err := client.post("/shifts/shift-c-old/end", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(endResult, http.StatusOK, "end covering shift"); err != nil {
		return err
	}
	shift, err := objectField(endResult.Body, "shift")
	if err != nil {
		return err
	}
	if phase, err := stringField(shift, "phase"); err != nil || phase != "ended" {
		return fmt.Errorf("ended shift phase mismatch: %q %v", phase, err)
	}

	// Ending it again is an explicit conflict.
	again, err := client.post("/shifts/shift-c-old/end", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(again, http.StatusConflict, "reject ending ended shift"); err != nil {
		return err
	}
	if err := expectErrorCode(again, "shift_already_ended", "double end code"); err != nil {
		return err
	}

	commandID, token, err := verifiedApprovedCommand(client, "gate-c1")
	if err != nil {
		return err
	}
	result := attemptExecute(client, commandID, token, "op-qin", false)
	if err := requireStatus(result, http.StatusConflict, "reject ended-shift execution"); err != nil {
		return err
	}
	if err := expectErrorCode(result, "duty_shift_ended", "ended shift execution code"); err != nil {
		return err
	}

	// The old shift's operator can no longer operate even though the old token
	// was valid: scheduling a new shift without an assignment stays unattended.
	futureStart := now.Add(time.Hour)
	futureEnd := now.Add(2 * time.Hour)
	if err := fixtureShift(client, "shift-c-next", "C 区新班", &futureStart, &futureEnd); err != nil {
		return err
	}
	result = attemptExecute(client, commandID, token, "op-qin", false)
	if err := requireStatus(result, http.StatusConflict, "reject old-shift operator after handover"); err != nil {
		return err
	}
	return nil
}

func verifyCancelScheduledShift(client *apiClient) error {
	// Before cancellation a scheduled shift cannot cover movement.
	commandID, token, err := verifiedApprovedCommand(client, "gate-g1")
	if err != nil {
		return err
	}
	before := attemptExecute(client, commandID, token, "op-qin", false)
	if err := requireStatus(before, http.StatusConflict, "reject execution before shift start"); err != nil {
		return err
	}
	if err := expectErrorCode(before, "duty_shift_not_started", "scheduled shift code"); err != nil {
		return err
	}

	// Cancel the not-yet-started shift; it must report ended and cannot be
	// ended twice.
	cancelled, err := client.post("/shifts/shift-g-future/end", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(cancelled, http.StatusOK, "cancel scheduled shift"); err != nil {
		return err
	}
	shift, err := objectField(cancelled.Body, "shift")
	if err != nil {
		return err
	}
	if phase, err := stringField(shift, "phase"); err != nil || phase != "ended" {
		return fmt.Errorf("canceled scheduled shift phase mismatch: %q %v", phase, err)
	}
	again, err := client.post("/shifts/shift-g-future/end", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(again, http.StatusConflict, "reject ending canceled shift"); err != nil {
		return err
	}
	if err := expectErrorCode(again, "shift_already_ended", "double cancel code"); err != nil {
		return err
	}

	// After cancellation the same approved command is treated as ended-shift,
	// never as still scheduled.
	after := attemptExecute(client, commandID, token, "op-qin", false)
	if err := requireStatus(after, http.StatusConflict, "reject execution after shift cancellation"); err != nil {
		return err
	}
	if err := expectErrorCode(after, "duty_shift_ended", "canceled shift code"); err != nil {
		return err
	}
	return nil
}

func verifyHandoverContention(client *apiClient, now time.Time) error {
	// Area D is covered by two overlapping active shifts with different
	// operators: neither may start a movement until the handover resolves.
	commandID, token, err := verifiedApprovedCommand(client, "gate-d1")
	if err != nil {
		return err
	}

	for _, operatorID := range []string{"op-lin", "op-ma"} {
		result := attemptExecute(client, commandID, token, operatorID, false)
		if err := requireStatus(result, http.StatusConflict, "reject contended handover execution"); err != nil {
			return err
		}
		if err := expectErrorCode(result, "duty_coverage_conflict", "handover contention code"); err != nil {
			return err
		}
	}

	// Ending one of the overlapping shifts resolves the contention; the
	// remaining duty operator wins.
	if _, err := client.post("/shifts/shift-d1/end", nil); err != nil {
		return err
	}
	losing := attemptExecute(client, commandID, token, "op-lin", false)
	if err := requireStatus(losing, http.StatusConflict, "old shift cannot continue after handover"); err != nil {
		return err
	}
	if err := expectErrorCode(losing, "operator_not_covering_gate", "old shift operator replaced code"); err != nil {
		return err
	}
	winning := attemptExecute(client, commandID, token, "op-ma", false)
	if err := requireStatus(winning, http.StatusOK, "remaining shift operator executes"); err != nil {
		return err
	}
	return nil
}

func verifyRevokedQualification(client *apiClient) error {
	// op-ma is still the active duty operator for area B after the D shift
	// ended; revoking the qualification must block movement immediately.
	revoke, err := client.post("/operators/op-ma/revoke", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(revoke, http.StatusOK, "revoke active operator"); err != nil {
		return err
	}
	operator, err := objectField(revoke.Body, "operator")
	if err != nil {
		return err
	}
	if qualified, ok := operator["qualified"].(bool); !ok || qualified {
		return fmt.Errorf("revoked operator still reports qualified")
	}

	commandID, token, err := verifiedApprovedCommand(client, "gate-b1")
	if err != nil {
		return err
	}
	result := attemptExecute(client, commandID, token, "op-ma", false)
	if err := requireStatus(result, http.StatusConflict, "reject revoked operator execution"); err != nil {
		return err
	}
	if err := expectErrorCode(result, "operator_not_qualified", "revoked operator code"); err != nil {
		return err
	}
	return nil
}

func verifyFaultedDutySnapshot(client *apiClient) error {
	// A fault during movement aborts the command and latches the gate, but the
	// on-duty fact under which the movement began must still be recorded.
	commandID, token, err := verifiedApprovedCommand(client, "gate-f1")
	if err != nil {
		return err
	}
	result := attemptExecute(client, commandID, token, "op-chen", true)
	if err := requireStatus(result, http.StatusOK, "execute faulting command under coverage"); err != nil {
		return err
	}
	executedCommand, err := objectField(result.Body, "command")
	if err != nil {
		return err
	}
	status, err := stringField(executedCommand, "status")
	if err != nil {
		return err
	}
	if status != "aborted" {
		return fmt.Errorf("fault execution status: expected aborted, got %q", status)
	}
	terminalReason, err := stringField(executedCommand, "terminal_reason")
	if err != nil {
		return err
	}
	if terminalReason != "simulated_gate_fault" {
		return fmt.Errorf("terminal reason mismatch: %q", terminalReason)
	}
	duty, err := objectField(executedCommand, "execution_duty")
	if err != nil {
		return err
	}
	if shiftID, err := stringField(duty, "shift_id"); err != nil || shiftID != "shift-a" {
		return fmt.Errorf("fault duty shift mismatch: %q %v", shiftID, err)
	}
	if operatorID, err := stringField(duty, "operator_id"); err != nil || operatorID != "op-chen" {
		return fmt.Errorf("fault duty operator mismatch: %q %v", operatorID, err)
	}

	executedGate, err := objectField(result.Body, "gate")
	if err != nil {
		return err
	}
	if gateStatus, err := stringField(executedGate, "status"); err != nil || gateStatus != "faulted" {
		return fmt.Errorf("fault gate status mismatch: %q %v", gateStatus, err)
	}
	return nil
}

func verifyHistoricalDutySnapshot(client *apiClient, now time.Time) error {
	// Execute on gate-a2 under the current shift, then end the shift and
	// revoke the operator: the completed command must still carry the exact
	// on-duty fact recorded at execution time.
	command, err := fixtureCommand(client, "gate-a2", 2.25, "历史值班事实", "dispatcher-zhao")
	if err != nil {
		return err
	}
	commandID, err := stringField(command, "id")
	if err != nil {
		return err
	}
	approved, err := fixtureClearReview(client, commandID, "safety-he")
	if err != nil {
		return err
	}
	token, err := stringField(approved, "execution_token")
	if err != nil {
		return err
	}

	executed := attemptExecute(client, commandID, token, "op-chen", false)
	if err := requireStatus(executed, http.StatusOK, "execute for historical snapshot"); err != nil {
		return err
	}

	// End the shift under which the movement happened.
	if _, err := client.post("/shifts/shift-a/end", nil); err != nil {
		return err
	}
	if _, err := client.post("/operators/op-chen/revoke", nil); err != nil {
		return err
	}

	fetched, err := client.get("/commands/" + commandID)
	if err != nil {
		return err
	}
	if err := requireStatus(fetched, http.StatusOK, "read historical command"); err != nil {
		return err
	}
	stored, err := objectField(fetched.Body, "command")
	if err != nil {
		return err
	}
	status, err := stringField(stored, "status")
	if err != nil {
		return err
	}
	if status != "completed" {
		return fmt.Errorf("historical command status changed to %q", status)
	}
	duty, err := objectField(stored, "execution_duty")
	if err != nil {
		return err
	}
	for field, expected := range map[string]string{
		"shift_id":      "shift-a",
		"shift_name":    "A 区当班",
		"area_id":       "area-a",
		"operator_id":   "op-chen",
		"operator_name": "陈晨",
	} {
		value, err := stringField(duty, field)
		if err != nil {
			return fmt.Errorf("historical duty missing %s: %w", field, err)
		}
		if value != expected {
			return fmt.Errorf("historical duty %s: expected %q, got %q", field, expected, value)
		}
	}
	return nil
}
