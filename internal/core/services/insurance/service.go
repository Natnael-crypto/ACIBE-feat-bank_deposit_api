package insurance

import (
	"context"
	"log/slog"
)

type Service struct {
	repo   repository
	api    collaborate
	logger *slog.Logger
}

func NewService(
	repo repository,
	api collaborate,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:   repo,
		api:    api,
		logger: logger,
	}
}

func (s *Service) HandleInsuranceReport(ctx context.Context) error {
	checks, err := s.api.GetInsuranceChecks(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.CreateInsuranceChecks(ctx, checks); err != nil {
		return err
	}
	s.logger.Info("successfully processed insurance report", "count", len(checks))
	return nil
}
