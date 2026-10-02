package main

import (
	"fmt"
	"net/http"
	"time"
)

func fixtureGate(
	client *apiClient,
	id string,
	name string,
	maxAperture float64,
	initialAperture float64,
) (map[string]any, error) {
	result, err := client.post("/gates", map[string]any{
		"id":               id,
		"name":             name,
		"max_aperture":     maxAperture,
		"initial_aperture": initialAperture,
	})
	if err != nil {
		return nil, err
	}
	if err := requireStatus(result, http.StatusCreated, "create fixture gate"); err != nil {
		return nil, err
	}
	gate, err := objectField(result.Body, "gate")
	if err != nil {
		return nil, fmt.Errorf("create fixture gate response: %w", err)
	}
	return gate, nil
}

func fixtureCommand(
	client *apiClient,
	gateID string,
	targetAperture float64,
	reason string,
	requestedBy string,
) (map[string]any, error) {
	result, err := client.post("/commands", map[string]any{
		"gate_id":         gateID,
		"target_aperture": targetAperture,
		"reason":          reason,
		"requested_by":    requestedBy,
	})
	if err != nil {
		return nil, err
	}
	if err := requireStatus(result, http.StatusCreated, "create fixture command"); err != nil {
		return nil, err
	}
	command, err := objectField(result.Body, "command")
	if err != nil {
		return nil, fmt.Errorf("create fixture command response: %w", err)
	}
	return command, nil
}

func fixtureClearReview(
	client *apiClient,
	commandID string,
	verifier string,
) (map[string]any, error) {
	result, err := client.post("/commands/"+commandID+"/interlock", map[string]any{
		"verdict":  "clear",
		"verifier": verifier,
		"note":     "联锁条件满足",
	})
	if err != nil {
		return nil, err
	}
	if err := requireStatus(result, http.StatusOK, "clear fixture interlock"); err != nil {
		return nil, err
	}
	command, err := objectField(result.Body, "command")
	if err != nil {
		return nil, fmt.Errorf("clear fixture interlock response: %w", err)
	}
	return command, nil
}

func fixtureOperator(client *apiClient, id, name string) error {
	result, err := client.post("/operators", map[string]any{
		"id":   id,
		"name": name,
	})
	if err != nil {
		return err
	}
	return requireStatus(result, http.StatusCreated, "register fixture operator")
}

func fixtureArea(client *apiClient, id, name string, gateIDs []string) error {
	result, err := client.post("/areas", map[string]any{
		"id":       id,
		"name":     name,
		"gate_ids": gateIDs,
	})
	if err != nil {
		return err
	}
	return requireStatus(result, http.StatusCreated, "create fixture area")
}

// fixtureShift creates a shift whose [starts_at, ends_at) window contains now,
// unless a custom window is supplied.
func fixtureShift(
	client *apiClient,
	id, name string,
	start, end *time.Time,
) error {
	now := time.Now().UTC()
	windowStart := now.Add(-time.Hour)
	windowEnd := now.Add(time.Hour)
	if start != nil {
		windowStart = *start
	}
	if end != nil {
		windowEnd = *end
	}
	result, err := client.post("/shifts", map[string]any{
		"id":        id,
		"name":      name,
		"starts_at": windowStart.Format(time.RFC3339),
		"ends_at":   windowEnd.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	return requireStatus(result, http.StatusCreated, "create fixture shift")
}

func fixtureAssignment(client *apiClient, shiftID, areaID, operatorID string) (string, error) {
	result, err := client.post("/assignments", map[string]any{
		"shift_id":    shiftID,
		"area_id":     areaID,
		"operator_id": operatorID,
	})
	if err != nil {
		return "", err
	}
	if err := requireStatus(result, http.StatusCreated, "create fixture assignment"); err != nil {
		return "", err
	}
	assignment, err := objectField(result.Body, "assignment")
	if err != nil {
		return "", fmt.Errorf("create fixture assignment response: %w", err)
	}
	return stringField(assignment, "id")
}

func expectErrorCode(result apiResult, expectedCode, operation string) error {
	code, err := stringField(result.Body, "code")
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if code != expectedCode {
		return fmt.Errorf("%s: expected error code %q, got %q", operation, expectedCode, code)
	}
	return nil
}
