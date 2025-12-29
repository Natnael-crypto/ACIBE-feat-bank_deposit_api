package claim

import (
	"context"
	"time"
)

func (s *Service) SyncClaims(ctx context.Context, date time.Time) {
	claims, err := s.api.GetClaimsFileDebug(ctx, date)
	if err != nil {
		s.logger.Error("failed to get claims from collaborate", "error", err)
		return
	}

	if err := s.repo.CreateClaims(ctx, claims); err != nil {
		s.logger.Error("failed to create claims", "error", err)
		return
	}
	s.logger.Info("successfully synced claims", "count", len(claims))

}

func (s *Service) SyncNewClaims(ctx context.Context) {
	claims, err := s.api.GetClaims(ctx)
	if err != nil {
		s.logger.Error("failed to get new claims from collaborate", "error", err)
		return
	}
	if err := s.repo.CreateNewClaims(ctx, claims); err != nil {
		s.logger.Error("failed to create new claims", "error", err)
		return
	}
	s.logger.Info("successfully synced new claims", "count", len(claims))
}
