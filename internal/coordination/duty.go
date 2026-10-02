package coordination

import (
	"context"
	"strings"
	"time"

	"example.com/spillway-gate-coordinator/internal/contracts"
	"example.com/spillway-gate-coordinator/internal/domain"
)

func (s *Service) RegisterOperator(
	ctx context.Context,
	request contracts.RegisterOperatorRequest,
) (domain.Operator, error) {
	operator, err := domain.NewOperator(request.ID, request.Name, s.now())
	if err != nil {
		return domain.Operator{}, err
	}
	if err := s.repository.RegisterOperator(ctx, operator); err != nil {
		return domain.Operator{}, err
	}
	return operator, nil
}

func (s *Service) Operator(ctx context.Context, id string) (domain.Operator, error) {
	return s.repository.Operator(ctx, id)
}

func (s *Service) RevokeOperator(ctx context.Context, id string) (domain.Operator, error) {
	return s.repository.RevokeOperator(ctx, id, s.now())
}

func (s *Service) CreateArea(
	ctx context.Context,
	request contracts.CreateAreaRequest,
) (domain.ControlArea, error) {
	area, err := domain.NewControlArea(request.ID, request.Name, request.GateIDs, s.now())
	if err != nil {
		return domain.ControlArea{}, err
	}
	return s.repository.CreateArea(ctx, area)
}

func (s *Service) Area(ctx context.Context, id string) (domain.ControlArea, error) {
	return s.repository.Area(ctx, id)
}

func (s *Service) CreateShift(
	ctx context.Context,
	request contracts.CreateShiftRequest,
) (domain.Shift, error) {
	now := s.now()

	// starts_at is optional: a shift without an explicit start begins now.
	startsAt := now
	if raw := strings.TrimSpace(request.StartsAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return domain.Shift{}, domain.Invalid("invalid_shift_start", "starts_at must be an RFC3339 timestamp")
		}
		startsAt = parsed
	}

	endsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.EndsAt))
	if err != nil {
		return domain.Shift{}, domain.Invalid("invalid_shift_end", "ends_at must be an RFC3339 timestamp")
	}

	shift, err := domain.NewShift(request.ID, request.Name, startsAt, endsAt, now)
	if err != nil {
		return domain.Shift{}, err
	}
	if err := s.repository.CreateShift(ctx, shift); err != nil {
		return domain.Shift{}, err
	}
	return shift, nil
}

func (s *Service) Shift(ctx context.Context, id string) (domain.Shift, error) {
	return s.repository.Shift(ctx, id)
}

func (s *Service) EndShift(ctx context.Context, id string) (domain.Shift, error) {
	return s.repository.EndShift(ctx, id, s.now())
}

func (s *Service) CreateAssignment(
	ctx context.Context,
	request contracts.CreateAssignmentRequest,
) (domain.ShiftAssignment, error) {
	assignmentID, err := s.newID("assign")
	if err != nil {
		return domain.ShiftAssignment{}, domain.Internal(
			"assignment_id_generation_failed",
			"assignment id could not be generated",
		)
	}

	assignment, err := domain.NewShiftAssignment(
		assignmentID,
		request.ShiftID,
		request.AreaID,
		request.OperatorID,
		s.now(),
	)
	if err != nil {
		return domain.ShiftAssignment{}, err
	}
	return s.repository.CreateAssignment(ctx, assignment)
}

func (s *Service) Assignment(ctx context.Context, id string) (domain.ShiftAssignment, error) {
	return s.repository.Assignment(ctx, id)
}
