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

func fixtureOperator(client *apiClient, id, name string) (map[string]any, error) {
	result, err := client.post("/operators", map[string]any{
		"id":   id,
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	if err := requireStatus(result, http.StatusCreated, "register fixture operator"); err != nil {
		return nil, err
	}
	return objectField(result.Body, "operator")
}

// fixtureActiveShift creates a shift covering gateIDs that is active now and
// assigns operators to it.
func fixtureActiveShift(
	client *apiClient,
	id string,
	gateIDs []string,
	operators ...string,
) (map[string]any, error) {
	start := nowOffset(-1 * time.Minute)
	end := nowOffset(8 * time.Hour)
	result, err := client.post("/shifts", map[string]any{
		"id":        id,
		"label":     "fixture shift " + id,
		"starts_at": start,
		"ends_at":   end,
		"gate_ids":  gateIDs,
	})
	if err != nil {
		return nil, err
	}
	if err := requireStatus(result, http.StatusCreated, "create fixture shift"); err != nil {
		return nil, err
	}
	shift, err := objectField(result.Body, "shift")
	if err != nil {
		return nil, fmt.Errorf("create fixture shift response: %w", err)
	}
	for _, operatorID := range operators {
		assigned, err := client.post("/shifts/"+id+"/assignments", map[string]any{
			"operator_id": operatorID,
		})
		if err != nil {
			return nil, err
		}
		if err := requireStatus(assigned, http.StatusOK, "assign fixture operator"); err != nil {
			return nil, err
		}
		shift, err = objectField(assigned.Body, "shift")
		if err != nil {
			return nil, err
		}
	}
	return shift, nil
}
