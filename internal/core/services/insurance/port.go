package insurance

import (
	"context"
)

type repository interface {
	CreateInsuranceChecks(ctx context.Context, checks []InsuranceCheck) error
}
type collaborate interface {
	GetInsuranceChecks(ctx context.Context) ([]InsuranceCheck, error)
}
