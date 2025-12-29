package claim

import (
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
