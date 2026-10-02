package domain

import (
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,31}$`)

func normalizeIdentifier(value, code string) (string, error) {
	value = strings.TrimSpace(value)
	if !identifierPattern.MatchString(value) {
		return "", Invalid(code, "id must use 3-32 lowercase letters, digits, or hyphens")
	}
	return value, nil
}
