package domain

import (
	"strings"
	"time"
)

type OperatorStatus string

const (
	OperatorActive  OperatorStatus = "active"
	OperatorRevoked OperatorStatus = "revoked"
)

// Operator is the registered gate operator credential. A credential must be
// active and within its validity window for the holder to cover a post.
type Operator struct {
	ID           string
	Name         string
	Status       OperatorStatus
	ValidUntil   time.Time
	RegisteredAt time.Time
	UpdatedAt    time.Time
	Version      int64
}

func NewOperator(id, name string, validUntil time.Time, now time.Time) (Operator, error) {
	id, err := validateIdentifier(id, "invalid_operator_id")
	if err != nil {
		return Operator{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Operator{}, Invalid("invalid_operator_name", "operator name is required")
	}

	validUntil = validUntil.UTC()
	if !validUntil.IsZero() && !validUntil.After(now.UTC()) {
		return Operator{}, Invalid("invalid_valid_until", "valid_until must be in the future")
	}

	return Operator{
		ID:           id,
		Name:         name,
		Status:       OperatorActive,
		ValidUntil:   validUntil,
		RegisteredAt: now.UTC(),
		UpdatedAt:    now.UTC(),
		Version:      1,
	}, nil
}

// IsQualifiedAt reports whether the credential is active and unexpired at the
// given moment.
func (o Operator) IsQualifiedAt(at time.Time) bool {
	if o.Status != OperatorActive {
		return false
	}
	if !o.ValidUntil.IsZero() && !at.UTC().Before(o.ValidUntil) {
		return false
	}
	return true
}

// Revoke invalidates the credential. Revocation is terminal: historical
// commands keep their own attestation snapshot and are unaffected.
func (o *Operator) Revoke(now time.Time) error {
	if o.Status == OperatorRevoked {
		return Conflict("operator_already_revoked", "operator credential is already revoked")
	}
	o.Status = OperatorRevoked
	o.UpdatedAt = now.UTC()
	o.Version++
	return nil
}
