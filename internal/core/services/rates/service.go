package rateservice

import (
	"context"

	ratesdomain "github.com/aci/backend/internal/core/domain/rates"
)

type repository interface {
	CreateOutPatientRate(ctx context.Context, rate ratesdomain.OutPatientRate) error
	GetOutPatientRate(ctx context.Context, code string, modifiers string) (ratesdomain.OutPatientRate, error)
	GetAllOutPatientRates(ctx context.Context) ([]ratesdomain.OutPatientRate, error)
}

type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateOutPatientRate(ctx context.Context, rate ratesdomain.OutPatientRate) error {
	return s.repo.CreateOutPatientRate(ctx, rate)
}

func (s *Service) GetOutPatientRate(ctx context.Context, code string, modifiers string) (ratesdomain.OutPatientRate, error) {
	return s.repo.GetOutPatientRate(ctx, code, modifiers)
}

func (s *Service) GetAllOutPatientRates(ctx context.Context) ([]ratesdomain.OutPatientRate, error) {
	return s.repo.GetAllOutPatientRates(ctx)
}
