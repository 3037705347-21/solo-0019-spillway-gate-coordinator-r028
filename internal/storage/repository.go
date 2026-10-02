package storage

import (
	"context"
	"time"

	"example.com/spillway-gate-coordinator/internal/domain"
)

type Repository interface {
	CreateGate(ctx context.Context, gate domain.Gate) error
	Gate(ctx context.Context, id string) (domain.Gate, error)
	CreateCommand(ctx context.Context, command domain.GateCommand) error
	Command(ctx context.Context, id string) (domain.GateCommand, error)
	ReviewCommand(
		ctx context.Context,
		commandID string,
		decision domain.InterlockDecision,
		token string,
		now time.Time,
	) (domain.GateCommand, domain.Gate, error)
	ExecuteCommand(
		ctx context.Context,
		commandID string,
		token string,
		operatorID string,
		simulateFault bool,
		now time.Time,
	) (domain.GateCommand, domain.Gate, error)

	RegisterOperator(ctx context.Context, operator domain.Operator) error
	Operator(ctx context.Context, id string) (domain.Operator, error)
	RevokeOperator(ctx context.Context, id string, now time.Time) (domain.Operator, error)

	CreateArea(ctx context.Context, area domain.ControlArea) (domain.ControlArea, error)
	Area(ctx context.Context, id string) (domain.ControlArea, error)

	CreateShift(ctx context.Context, shift domain.Shift) error
	Shift(ctx context.Context, id string) (domain.Shift, error)
	EndShift(ctx context.Context, id string, now time.Time) (domain.Shift, error)

	CreateAssignment(ctx context.Context, assignment domain.ShiftAssignment) (domain.ShiftAssignment, error)
	Assignment(ctx context.Context, id string) (domain.ShiftAssignment, error)
}
