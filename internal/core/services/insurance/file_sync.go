package insurance

import (
	"context"
)

func (s *Service) SyncInsuranceChecks(ctx context.Context) {

	checks, err := s.api.GetInsuranceChecks(ctx)
	if err != nil {
		s.logger.Error("failed to get insurance checks from collaborate", "error", err)
		return
	}
	if err := s.repo.CreateInsuranceChecks(ctx, checks); err != nil {
		s.logger.Error("failed to create insurance checks", "error", err)
		return
	}
	s.logger.Info("successfully synced insurance checks", "count", len(checks))

}
