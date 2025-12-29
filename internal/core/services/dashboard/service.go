package dashboard

import (
	"context"
	"log/slog"

	dashboardrepo "github.com/aci/backend/internal/repository/dashboard"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
)

type Service interface {
	GetDashboardData(ctx context.Context, year int) (*dashboardtypes.DashboardData, error)
	GetAvailableYears(ctx context.Context) ([]int, error)
	GetYearlyDashboardData(ctx context.Context) (map[int]dashboardtypes.MonthData, error)
	GetClaims(ctx context.Context, year *int, month *int, category string, claimType string, excludePeer bool, search *string, peerOnly bool, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string) (*dashboardtypes.ClaimsResponse, error)
	GetClaimCharges(ctx context.Context, claimID string, year *int, month *int) ([]dashboardtypes.ClaimCharge, error)
	GetClaimById(ctx context.Context, claimID string) ([]dashboardtypes.Claim, error)
	GetPayments(ctx context.Context, year *int, month *int, excludePeer bool, peerOnly bool, search *string, claimType string, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string) (*dashboardtypes.PaymentsResponse, error)
	GetPaymentsByCheckNumber(ctx context.Context, checkNumber string, year int, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) (*dashboardtypes.PaymentsResponse, error)
	GetBankDepositData(ctx context.Context, year int) ([]dashboardtypes.BankDepositMonthData, error)
	GetBankDepositClaims(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.ClaimsResponse, error)
	GetBankDepositAppliedPayments(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.PaymentsResponse, error)
	GetInsuranceChecks(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.InsuranceChecksResponse, error)
	GetPaymentChecksByClaimAndCharge(ctx context.Context, claimID string, chargeID string, year *int, month *int, page int, limit int) (*dashboardtypes.PaymentChecksResponse, error)
	// Excel export methods
	ExportClaimsToExcel(ctx context.Context, year *int, month *int, category string, claimType string, excludePeer bool, search *string, peerOnly bool) ([]byte, string, error)
	ExportPaymentsToExcel(ctx context.Context, year *int, month *int, excludePeer bool, peerOnly bool, search *string, claimType string) ([]byte, string, error)
	ExportPaymentsByClaimAndChargeToExcel(ctx context.Context, claimID string, chargeID string, year *int, month *int) ([]byte, string, error)
	ExportInsuranceChecksToExcel(ctx context.Context, year int, month int, search string) ([]byte, string, error)
	ExportChargesToExcel(ctx context.Context, claimID string, year *int, month *int) ([]byte, string, error)
}

type service struct {
	repo   dashboardrepo.Repository
	logger *slog.Logger
}

func NewService(repo dashboardrepo.Repository, logger *slog.Logger) Service {
	return &service{repo: repo, logger: logger}
}
