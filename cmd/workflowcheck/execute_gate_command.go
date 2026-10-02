package main

import (
	"fmt"
	"net/http"
)

func runExecuteGateCommand(client *apiClient) error {
	if _, err := fixtureGate(client, "gate-west", "西侧泄洪闸", 4, 0.25); err != nil {
		return err
	}
	if err := fixtureOperator(client, "operator-chen", "陈操作"); err != nil {
		return err
	}
	if err := fixtureArea(client, "area-west", "西侧控制区", []string{"gate-west"}); err != nil {
		return err
	}
	if err := fixtureShift(client, "shift-west-day", "白班", nil, nil); err != nil {
		return err
	}
	if _, err := fixtureAssignment(client, "shift-west-day", "area-west", "operator-chen"); err != nil {
		return err
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
		"operator_id":    "operator-chen",
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
	duty, err := objectField(executedCommand, "execution_duty")
	if err != nil {
		return err
	}
	if onShift, err := stringField(duty, "shift_id"); err != nil || onShift != "shift-west-day" {
		return fmt.Errorf("execution duty shift mismatch: %v %v", onShift, err)
	}
	if onArea, err := stringField(duty, "area_id"); err != nil || onArea != "area-west" {
		return fmt.Errorf("execution duty area mismatch: %v %v", onArea, err)
	}
	if byOperator, err := stringField(executedCommand, "executed_by"); err != nil || byOperator != "operator-chen" {
		return fmt.Errorf("executed_by mismatch: %v %v", byOperator, err)
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

	reused, err := client.post("/commands/"+commandID+"/execute", map[string]any{
		"operator_id":    "operator-chen",
		"token":          token,
		"simulate_fault": false,
	})
	if err != nil {
		return err
	}
	if err := requireStatus(reused, http.StatusConflict, "reject reused execution token"); err != nil {
		return err
	}

	fmt.Println("execute-gate-command: command completed with on-duty fact and token reuse rejected")
	return nil
}
