package coordination

import (
	"strings"
	"time"

	"example.com/spillway-gate-coordinator/internal/domain"
)

// parseShiftTime accepts an RFC3339 (optionally nano) timestamp, which is the
// same format the service emits.
func parseShiftTime(value string, field string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, domain.Invalid("invalid_"+field, field+" is required and must use RFC3339 time")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, domain.Invalid("invalid_"+field, field+" must use RFC3339 time")
	}
	return parsed.UTC(), nil
}
