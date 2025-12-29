package payment

import (
	"context"
	"time"
)

func (s *Service) SyncPayments(ctx context.Context, date time.Time) {
	payments, err := s.api.GetPaymentsFileDebug(ctx, date)
	if err != nil {
		s.logger.Error("failed to get payments from collaborate", "error", err)
		return
	}

	if err := s.repo.CreatePayments(ctx, payments); err != nil {
		s.logger.Error("failed to create payments", "error", err)
		return
	}
	s.logger.Info("successfully synced payments", "count", len(payments))

}
func (s *Service) SyncNewPayments(ctx context.Context) {
	payments, err := s.api.GetPayments(ctx)
	if err != nil {
		s.logger.Error("failed to get new payments from collaborate", "error", err)
		return
	}
	for _, payment := range payments {
		if err := s.repo.CreateNewPayments(ctx, payment); err != nil {
			s.logger.Error("failed to create new payment", "error", err)
			return
		}
	}
	s.logger.Info("successfully synced new payments", "count", len(payments))
}
