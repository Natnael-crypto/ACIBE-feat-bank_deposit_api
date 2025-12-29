package payment

import (
	"context"
	"time"
)

type repository interface {
	CreatePayments(ctx context.Context, payments []Payment) error
	CreateNewPayments(ctx context.Context, payments NewPayment) error
}
type collaborate interface {
	GetPaymentsApi(ctx context.Context, date time.Time) ([]Payment, error)
	GetPaymentsFileDebug(ctx context.Context, date time.Time) ([]Payment, error)
	GetPayments(ctx context.Context) ([]NewPayment, error)
}
