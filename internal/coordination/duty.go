package coordination

import (
	"context"
	"time"

	"example.com/spillway-gate-coordinator/internal/contracts"
	"example.com/spillway-gate-coordinator/internal/domain"
)

func (s *Service) RegisterOperator(
	ctx context.Context,
	request contracts.RegisterOperatorRequest,
) (domain.Operator, error) {
	var validUntil time.Time
	if request.ValidUntil != "" {
		parsed, err := parseShiftTime(request.ValidUntil, "valid_until")
		if err != nil {
			return domain.Operator{}, err
		}
		validUntil = parsed
	}

	operator, err := domain.NewOperator(request.ID, request.Name, validUntil, s.now())
	if err != nil {
		return domain.Operator{}, err
	}
	if err := s.repository.CreateOperator(ctx, operator); err != nil {
		return domain.Operator{}, err
	}
	return operator, nil
}

func (s *Service) Operator(ctx context.Context, id string) (domain.Operator, error) {
	return s.repository.Operator(ctx, id)
}

func (s *Service) RevokeOperator(ctx context.Context, operatorID string) (domain.Operator, error) {
	return s.repository.RevokeOperator(ctx, operatorID, s.now())
}

func (s *Service) CreateShift(
	ctx context.Context,
	request contracts.CreateShiftRequest,
) (domain.DutyShift, error) {
	startsAt, err := parseShiftTime(request.StartsAt, "starts_at")
	if err != nil {
		return domain.DutyShift{}, err
	}
	endsAt, err := parseShiftTime(request.EndsAt, "ends_at")
	if err != nil {
		return domain.DutyShift{}, err
	}

	shift, err := domain.NewDutyShift(request.ID, request.Label, startsAt, endsAt, request.GateIDs, s.now())
	if err != nil {
		return domain.DutyShift{}, err
	}
	if err := s.repository.CreateShift(ctx, shift); err != nil {
		return domain.DutyShift{}, err
	}
	return shift, nil
}

func (s *Service) Shift(ctx context.Context, id string) (domain.DutyShift, error) {
	return s.repository.Shift(ctx, id, s.now())
}

func (s *Service) AssignShiftOperator(
	ctx context.Context,
	shiftID string,
	request contracts.AssignShiftRequest,
) (domain.DutyShift, error) {
	return s.repository.AssignShiftOperator(ctx, shiftID, request.OperatorID, s.now())
}

func (s *Service) EndShift(ctx context.Context, shiftID string) (domain.DutyShift, error) {
	return s.repository.EndShift(ctx, shiftID, s.now())
}

func (s *Service) GateCoverage(ctx context.Context, gateID string) (domain.GateCoverage, error) {
	return s.repository.GateCoverage(ctx, gateID, s.now())
}
