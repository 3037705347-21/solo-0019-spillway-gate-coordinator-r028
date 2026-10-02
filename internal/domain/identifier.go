package domain

import (
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,31}$`)

// validateIdentifier normalizes and checks a lowercase identifier used by
// gates, operators and duty shifts.
func validateIdentifier(value string, code string) (string, error) {
	value = strings.TrimSpace(value)
	if !identifierPattern.MatchString(value) {
		return "", Invalid(code, "id must use lowercase letters, digits, and hyphens")
	}
	return value, nil
}
