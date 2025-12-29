package claim

import (
	"context"
	"time"
)

type repository interface {
	CreateClaims(ctx context.Context, claims []Claim) error
	CreateNewClaims(ctx context.Context, claims []NewClaim) error
}
type collaborate interface {
	GetClaimsApi(ctx context.Context, date time.Time) ([]Claim, error)
	GetClaimsFileDebug(ctx context.Context, date time.Time) ([]Claim, error)
	GetClaims(ctx context.Context) ([]NewClaim, error)
}
