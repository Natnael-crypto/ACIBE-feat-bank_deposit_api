package dashboard

import (
	domain "github.com/aci/backend/internal/core/domain/common"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
)

// DashboardResponse matches the frontend expected format
type DashboardResponse struct {
	Data dashboardtypes.DashboardData `json:"data"`
}

// ClaimsResponse matches the service response format for paginated claims
type ClaimsResponse struct {
	Data       []dashboardtypes.Claim      `json:"data"`
	Pagination domain.PaginationMetadata   `json:"pagination"`
}

// ClaimCharge is an alias for the ClaimCharge type from the types package
type ClaimCharge = dashboardtypes.ClaimCharge

// Claim is an alias for the Claim type from the types package
type Claim = dashboardtypes.Claim

// PaymentsResponse is an alias for the PaymentsResponse type from the types package
type PaymentsResponse = dashboardtypes.PaymentsResponse

// PaymentChecksResponse is an alias for the PaymentChecksResponse type from the types package
type PaymentChecksResponse = dashboardtypes.PaymentChecksResponse

// MonthData is an alias for the MonthData type from the types package
type MonthData = dashboardtypes.MonthData

// BankDepositMonthData is an alias for the BankDepositMonthData type from the types package
type BankDepositMonthData = dashboardtypes.BankDepositMonthData

// InsuranceChecksResponse is an alias for the InsuranceChecksResponse type from the types package
type InsuranceChecksResponse = dashboardtypes.InsuranceChecksResponse