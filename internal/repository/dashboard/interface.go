package dashboard

import (
	"context"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/types/dashboard"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface {
	GetDashboardData(ctx context.Context, year int32) ([]dashboard.DashboardDataRow, error)
	GetYearlyDashboardData(ctx context.Context) ([]dashboard.DashboardDataRow, error)
	GetAvailableYears(ctx context.Context) ([]int32, error)
	GetClaims(ctx context.Context, year *int32, month *int32, category string, claimType *string, excludePeer bool, search *string, peerOnly bool, limit int32, offset int32, claimsExpectedFilter *string, paymentStatusFilter *string) ([]dbgen.GetClaimsRow, error)
	GetDashboardClaimsCount(ctx context.Context, year *int32, month *int32, category string, claimType *string, excludePeer bool, search *string, peerOnly bool, claimsExpectedFilter *string, paymentStatusFilter *string) (dbgen.GetClaimsCountRow, error)
	GetClaimCharges(ctx context.Context, claimID string, year *int32, month *int32) ([]dbgen.GetClaimChargesRow, error)
	GetClaimById(ctx context.Context, claimID string) ([]dbgen.GetClaimByIdRow, error)
	GetDashboardPaymentsPaginated(ctx context.Context, year int32, month int32, excludePeer bool, peerOnly bool, search string, limit int32, offset int32, claimType string, claimsExpectedFilter *string, paymentStatusFilter *string) ([]dbgen.GetDashboardPaymentsPaginatedRow, error)
	GetDashboardPaymentsCount(ctx context.Context, year int32, month int32, excludePeer bool, peerOnly bool, search string, claimType string, claimsExpectedFilter *string, paymentStatusFilter *string) (int64, error)
	GetPaymentsByCheckNumberPaginated(ctx context.Context, checkNumber string, year int32, limit int32, offset int32, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) ([]dbgen.GetPaymentsByCheckNumberRow, error)
	GetPaymentsByCheckNumberCount(ctx context.Context, checkNumber string, year int32, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) (int64, error)
	GetBankDepositData(ctx context.Context, year int32) ([]dashboard.BankDepositDataRow, error)
	GetBankDepositClaimsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositClaimsPaginatedRow, error)
	GetBankDepositClaimsCount(ctx context.Context, year int32, month int32, search string) (int64, error)
	GetBankDepositAppliedPaymentsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositAppliedPaymentsPaginatedRow, error)
	GetBankDepositAppliedPaymentsCount(ctx context.Context, year int32, month int32, search string) (int64, error)
	GetInsuranceChecksPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetInsuranceChecksPaginatedRow, error)
	GetInsuranceChecksCount(ctx context.Context, year int32, month int32, search string) (int64, error)
	GetInsuranceChecksTotalDeposit(ctx context.Context, year int32, month int32, search string) (pgtype.Numeric, error)
	GetPaymentsByClaimAndCharge(ctx context.Context, params dbgen.GetPaymentsByClaimAndChargeParams) ([]dbgen.GetPaymentsByClaimAndChargeRow, error)
	GetPaymentChecksByClaimAndCharge(ctx context.Context, params dbgen.GetPaymentChecksByClaimAndChargeParams) ([]dbgen.GetPaymentChecksByClaimAndChargeRow, error)
	GetPaymentChecksByClaimAndChargeCount(ctx context.Context, params dbgen.GetPaymentChecksByClaimAndChargeCountParams) (int64, error)
}
