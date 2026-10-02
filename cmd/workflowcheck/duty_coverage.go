package main

import (
	"fmt"
	"net/http"
	"time"
)

// runDutyCoverage exercises the duty assignment and post-coverage rules added
// on top of the one-time execution token: the responsible party maintains
// shifts, crews and covered areas, and only an operator whose active shift
// actually covers the gate with a valid credential can start movement.
func runDutyCoverage(client *apiClient) error {
	// --- operator registry ------------------------------------------------
	if _, err := fixtureOperator(client, "operator-qin", "秦值班"); err != nil {
		return err
	}
	if _, err := fixtureOperator(client, "operator-luo", "罗值班"); err != nil {
		return err
	}

	duplicate, err := client.post("/operators", map[string]any{"id": "operator-qin", "name": "重复登记"})
	if err != nil {
		return err
	}
	if err := expectConflictCode(duplicate, "duplicate_operator_id", "duplicate operator registration"); err != nil {
		return err
	}

	// --- unattended gate: no shift covers it ------------------------------
	if _, err := fixtureGate(client, "gate-a", "甲号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	cmdA, tokenA, err := approvedCommand(client, "gate-a", 2.0)
	if err != nil {
		return err
	}
	unattended, err := client.post("/commands/"+cmdA+"/execute", executePayload("operator-qin", tokenA))
	if err != nil {
		return err
	}
	if err := expectConflictCode(unattended, "gate_unattended", "execute against unattended gate"); err != nil {
		return err
	}

	coverage, err := client.get("/gates/gate-a/coverage")
	if err != nil {
		return err
	}
	if err := requireStatus(coverage, http.StatusOK, "read unattended coverage"); err != nil {
		return err
	}
	coverageBody, err := objectField(coverage.Body, "coverage")
	if err != nil {
		return err
	}
	if status, _ := stringField(coverageBody, "status"); status != "unattended" {
		return fmt.Errorf("coverage status: expected unattended, got %q", status)
	}

	// --- active shift with the assigned operator succeeds ----------------
	if _, err := fixtureActiveShift(client, "shift-a", []string{"gate-a"}, "operator-qin"); err != nil {
		return err
	}
	done, err := client.post("/commands/"+cmdA+"/execute", executePayload("operator-qin", tokenA))
	if err != nil {
		return err
	}
	if err := requireStatus(done, http.StatusOK, "execute with covering on-duty operator"); err != nil {
		return err
	}

	// --- off-duty operator: a different shift crew owns another gate ------
	if _, err := fixtureGate(client, "gate-b", "乙号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	if _, err := fixtureActiveShift(client, "shift-b", []string{"gate-b"}, "operator-luo"); err != nil {
		return err
	}
	cmdB, tokenB, err := approvedCommand(client, "gate-b", 2.0)
	if err != nil {
		return err
	}
	offDuty, err := client.post("/commands/"+cmdB+"/execute", executePayload("operator-qin", tokenB))
	if err != nil {
		return err
	}
	if err := expectConflictCode(offDuty, "operator_not_on_duty", "execute by operator outside the active crew"); err != nil {
		return err
	}

	// --- duplicate assignment is rejected --------------------------------
	dupAssign, err := client.post("/shifts/shift-b/assignments", map[string]any{"operator_id": "operator-luo"})
	if err != nil {
		return err
	}
	if err := expectConflictCode(dupAssign, "duplicate_shift_assignment", "assign same operator twice"); err != nil {
		return err
	}

	// --- ended shift: early end closes the post and every action is clear -
	if _, err := fixtureGate(client, "gate-c", "丙号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	if _, err := fixtureOperator(client, "operator-gao", "高值班"); err != nil {
		return err
	}
	if _, err := fixtureActiveShift(client, "shift-c", []string{"gate-c"}, "operator-gao"); err != nil {
		return err
	}
	endResult, err := client.post("/shifts/shift-c/end", nil)
	if err != nil {
		return err
	}
	if err := requireStatus(endResult, http.StatusOK, "end active shift early"); err != nil {
		return err
	}
	endAgain, err := client.post("/shifts/shift-c/end", nil)
	if err != nil {
		return err
	}
	if err := expectConflictCode(endAgain, "shift_already_ended", "end an ended shift"); err != nil {
		return err
	}
	assignEnded, err := client.post("/shifts/shift-c/assignments", map[string]any{"operator_id": "operator-luo"})
	if err != nil {
		return err
	}
	if err := expectConflictCode(assignEnded, "shift_already_ended", "assign into an ended shift"); err != nil {
		return err
	}
	cmdC, tokenC, err := approvedCommand(client, "gate-c", 2.0)
	if err != nil {
		return err
	}
	ended, err := client.post("/commands/"+cmdC+"/execute", executePayload("operator-gao", tokenC))
	if err != nil {
		return err
	}
	if err := expectConflictCode(ended, "shift_already_ended", "execute under an ended shift"); err != nil {
		return err
	}

	// --- handover contention: overlapping shifts on the same gate ---------
	if _, err := fixtureGate(client, "gate-d", "丁号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	if _, err := fixtureActiveShift(client, "shift-d", []string{"gate-d"}); err != nil {
		return err
	}
	overlap, err := client.post("/shifts", map[string]any{
		"id":        "shift-d2",
		"starts_at": nowOffset(30 * time.Minute),
		"ends_at":   nowOffset(9 * time.Hour),
		"gate_ids":  []string{"gate-d"},
	})
	if err != nil {
		return err
	}
	if err := expectConflictCode(overlap, "shift_handover_contention", "overlapping handover shift"); err != nil {
		return err
	}
	// A back-to-back shift touching only at the boundary is a clean handover.
	touch, err := client.post("/shifts", map[string]any{
		"id":        "shift-d3",
		"starts_at": nowOffset(8 * time.Hour),
		"ends_at":   nowOffset(9 * time.Hour),
		"gate_ids":  []string{"gate-d"},
	})
	if err != nil {
		return err
	}
	if err := requireStatus(touch, http.StatusCreated, "back-to-back handover shift"); err != nil {
		return err
	}

	// --- revoked credential cannot be rostered or execute -----------------
	if _, err := fixtureGate(client, "gate-e", "戊号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	if _, err := fixtureOperator(client, "operator-he", "何值班"); err != nil {
		return err
	}
	if _, err := client.post("/operators/operator-he/revoke", nil); err != nil {
		return err
	}
	if _, err := fixtureActiveShift(client, "shift-e", []string{"gate-e"}); err != nil {
		return err
	}
	assignRevoked, err := client.post("/shifts/shift-e/assignments", map[string]any{"operator_id": "operator-he"})
	if err != nil {
		return err
	}
	if err := expectConflictCode(assignRevoked, "operator_credential_invalid", "assign a revoked operator"); err != nil {
		return err
	}

	// Revocation after rostering blocks execution too. Reuse shift-e: the
	// rejected assignment above did not change its roster.
	if _, err := fixtureOperator(client, "operator-ma", "马值班"); err != nil {
		return err
	}
	if _, err := client.post("/shifts/shift-e/assignments", map[string]any{"operator_id": "operator-ma"}); err != nil {
		return err
	}
	if _, err := client.post("/operators/operator-ma/revoke", nil); err != nil {
		return err
	}
	cmdE, tokenE, err := approvedCommand(client, "gate-e", 2.0)
	if err != nil {
		return err
	}
	invalidCredential, err := client.post("/commands/"+cmdE+"/execute", executePayload("operator-ma", tokenE))
	if err != nil {
		return err
	}
	if err := expectConflictCode(invalidCredential, "operator_credential_invalid", "execute with revoked credential"); err != nil {
		return err
	}

	// --- unknown operator is a stable not-found result --------------------
	if _, err := fixtureGate(client, "gate-f", "己号泄洪闸", 4, 0.5); err != nil {
		return err
	}
	cmdF, tokenF, err := approvedCommand(client, "gate-f", 2.0)
	if err != nil {
		return err
	}
	unknown, err := client.post("/commands/"+cmdF+"/execute", executePayload("operator-ghost", tokenF))
	if err != nil {
		return err
	}
	if err := requireStatus(unknown, http.StatusNotFound, "execute with unknown operator"); err != nil {
		return err
	}
	if code := errorCode(unknown); code != "operator_not_found" {
		return fmt.Errorf("unknown operator code: expected operator_not_found, got %q", code)
	}

	fmt.Println("duty-coverage: unattended, off-duty, ended, contention, duplicate and revoked operators rejected; active crew allowed")
	return nil
}

// approvedCommand creates a command and clears its interlock, returning the
// command id and its one-time execution token.
func approvedCommand(client *apiClient, gateID string, target float64) (string, string, error) {
	command, err := fixtureCommand(client, gateID, target, "值班覆盖验证", "dispatcher-duty")
	if err != nil {
		return "", "", err
	}
	commandID, err := stringField(command, "id")
	if err != nil {
		return "", "", err
	}
	approved, err := fixtureClearReview(client, commandID, "safety-duty")
	if err != nil {
		return "", "", err
	}
	token, err := stringField(approved, "execution_token")
	if err != nil {
		return "", "", err
	}
	return commandID, token, nil
}

func executePayload(operator, token string) map[string]any {
	return map[string]any{
		"operator":       operator,
		"token":          token,
		"simulate_fault": false,
	}
}
