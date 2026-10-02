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
		operator string,
		simulateFault bool,
		now time.Time,
	) (domain.GateCommand, domain.Gate, error)

	CreateOperator(ctx context.Context, operator domain.Operator) error
	Operator(ctx context.Context, id string) (domain.Operator, error)
	RevokeOperator(ctx context.Context, operatorID string, now time.Time) (domain.Operator, error)

	CreateShift(ctx context.Context, shift domain.DutyShift) error
	Shift(ctx context.Context, id string, now time.Time) (domain.DutyShift, error)
	AssignShiftOperator(ctx context.Context, shiftID, operatorID string, now time.Time) (domain.DutyShift, error)
	EndShift(ctx context.Context, shiftID string, now time.Time) (domain.DutyShift, error)
	GateCoverage(ctx context.Context, gateID string, now time.Time) (domain.GateCoverage, error)
}
