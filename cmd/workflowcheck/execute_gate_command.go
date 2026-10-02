package main

import (
	"fmt"
	"net/http"
)

func runExecuteGateCommand(client *apiClient) error {
	if _, err := fixtureGate(client, "gate-west", "西侧泄洪闸", 4, 0.25); err != nil {
		return err
	}
	if _, err := fixtureOperator(client, "operator-chen", "陈操作"); err != nil {
		return err
	}
	if _, err := fixtureActiveShift(client, "shift-west", []string{"gate-west"}, "operator-chen"); err != nil {
		return err
	}

	coverage, err := client.get("/gates/gate-west/coverage")
	if err != nil {
		return err
	}
	if err := requireStatus(coverage, http.StatusOK, "read gate coverage"); err != nil {
		return err
	}
	coverageBody, err := objectField(coverage.Body, "coverage")
	if err != nil {
		return err
	}
	coverageStatus, err := stringField(coverageBody, "status")
	if err != nil {
		return err
	}
	if coverageStatus != "covered" {
		return fmt.Errorf("coverage before execution: expected covered, got %q", coverageStatus)
	}

	command, err := fixtureCommand(client, "gate-west", 2.0, "平衡下游水位", "dispatcher-zhao")
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

	executed, err := client.post("/commands/"+commandID+"/execute", map[string]any{
		"operator":       "operator-chen",
		"token":          token,
		"simulate_fault": false,
	})
	if err != nil {
		return err
	}
	if err := requireStatus(executed, http.StatusOK, "execute approved command"); err != nil {
		return err
	}
	executedCommand, err := objectField(executed.Body, "command")
	if err != nil {
		return err
	}
	completedStatus, err := stringField(executedCommand, "status")
	if err != nil {
		return err
	}
	if completedStatus != "completed" {
		return fmt.Errorf("execution status: expected completed, got %q", completedStatus)
	}

	duty, err := objectField(executedCommand, "duty")
	if err != nil {
		return err
	}
	if shiftID, _ := stringField(duty, "shift_id"); shiftID != "shift-west" {
		return fmt.Errorf("duty snapshot shift: expected shift-west, got %q", shiftID)
	}
	if operatorID, _ := stringField(duty, "operator_id"); operatorID != "operator-chen" {
		return fmt.Errorf("duty snapshot operator: expected operator-chen, got %q", operatorID)
	}
	if gateID, _ := stringField(duty, "gate_id"); gateID != "gate-west" {
		return fmt.Errorf("duty snapshot gate: expected gate-west, got %q", gateID)
	}
	if _, err := stringField(duty, "verified_at"); err != nil {
		return fmt.Errorf("duty snapshot is missing verified_at: %w", err)
	}

	executedGate, err := objectField(executed.Body, "gate")
	if err != nil {
		return err
	}
	gateStatus, err := stringField(executedGate, "status")
	if err != nil {
		return err
	}
	aperture, err := floatField(executedGate, "aperture")
	if err != nil {
		return err
	}
	if gateStatus != "idle" || aperture != 2.0 {
		return fmt.Errorf("settled gate mismatch: status=%q aperture=%v", gateStatus, aperture)
	}

	// Revoking the credential after the fact must not rewrite the on-duty fact
	// recorded by the historical command.
	if _, err := client.post("/operators/operator-chen/revoke", nil); err != nil {
		return err
	}
	stored, err := client.get("/commands/" + commandID)
	if err != nil {
		return err
	}
	storedCommand, err := objectField(stored.Body, "command")
	if err != nil {
		return err
	}
	storedDuty, err := objectField(storedCommand, "duty")
	if err != nil {
		return err
	}
	if shiftID, _ := stringField(storedDuty, "shift_id"); shiftID != "shift-west" {
		return fmt.Errorf("historical duty snapshot changed after revocation: shift=%q", shiftID)
	}

	reused, err := client.post("/commands/"+commandID+"/execute", map[string]any{
		"operator":       "operator-chen",
		"token":          token,
		"simulate_fault": false,
	})
	if err != nil {
		return err
	}
	if err := requireStatus(reused, http.StatusConflict, "reject reused execution token"); err != nil {
		return err
	}

	fmt.Println("execute-gate-command: qualified on-duty operator completed; duty fact retained; token reuse rejected")
	return nil
}
