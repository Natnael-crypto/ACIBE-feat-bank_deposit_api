package dashboard

import (
	domain "github.com/aci/backend/internal/core/domain/common"
	"github.com/shopspring/decimal"
)

type Claim struct {
	ClaimID            string   `json:"claimId"`
	ChargeID           string   `json:"chargeId"`
	PatientID          string   `json:"patientId"`
	PatientName        string   `json:"patientName"`
	TotalClaim         *float64 `json:"totalClaim,omitempty"`
	TotalClaims        *float64 `json:"totalClaims,omitempty"`      // Sum of amount for duplicate claims
	TotalExpected      *float64 `json:"totalExpected,omitempty"`    // Sum of expected_amount for duplicate claims
	TotalPayment       *float64 `json:"totalPayment,omitempty"`     // Sum of insurance_paid_a from payments
	TotalOutstanding   *float64 `json:"totalOutstanding,omitempty"` // Outstanding amount (expected - billed)
	TotalUnits         *int64   `json:"totalUnits,omitempty"`       // Sum of units for duplicate claims
	ClaimFromDate      string   `json:"claimFromDate"`
	ClaimToDate        string   `json:"claimToDate"`
	ClaimEnteredDate   *string  `json:"claimEnteredDate,omitempty"`
	ClaimType          string   `json:"claimType"`
	PrimaryPayerName   string   `json:"primaryPayerName"`
	PrimaryPayerID     string   `json:"primaryPayerId"`
	SecondaryPayerName *string  `json:"secondaryPayerName,omitempty"`
	SecondaryPayerID   *string  `json:"secondaryPayerId,omitempty"`
	ChargeCount        int      `json:"chargeCount"`
	TotalCharges       int      `json:"totalCharges"` // Count of duplicate charges for this claim
	CptCodes           []string `json:"cptCodes,omitempty"`
	Modifiers          []string `json:"modifiers,omitempty"`
	Units              *string  `json:"units,omitempty"`
	FacilityPosCode    *string  `json:"facilityPosCode,omitempty"`
	ChargePosCode      *string  `json:"chargePosCode,omitempty"`
	TertiaryPayerName  *string  `json:"tertiaryPayerName,omitempty"`
	TertiaryPayerID    *string  `json:"tertiaryPayerId,omitempty"`
	CptCode            string   `json:"cptCode,omitempty"`           // Individual CPT code for this row
	UnpaidReason       string   `json:"unpaidReason,omitempty"`      // Unpaid reason from payments
	AdjustmentReasons  string   `json:"adjustmentReasons,omitempty"` // Adjustment reasons from payments
	Remark             string   `json:"remark,omitempty"`            // Remark from payments
	TotalCheckNumber   *int     `json:"totalCheckNumber,omitempty"`  // Count of unique check numbers for this charge
}

type ClaimsTotals struct {
	Claims           int64   `json:"claims"`
	TotalUnits       int64   `json:"totalUnits"`
	ChargeCount      int64   `json:"chargeCount"`
	TotalCheckNumber int64   `json:"totalCheckNumber"`
	Charge           float64 `json:"charge"`
	Expected         float64 `json:"expected"`
	Payment          float64 `json:"payment"`
	Outstanding      float64 `json:"outstanding"`
}

type ClaimsResponse struct {
	Data       []Claim                   `json:"data"`
	Pagination domain.PaginationMetadata `json:"pagination"`
	Totals     *ClaimsTotals             `json:"totals,omitempty"`
}

type ClaimCharge struct {
	ClaimID             string        `json:"claimId"`
	CptCode             string        `json:"cptCode"`
	ChargeID            string        `json:"chargeId"`
	Modifiers           []string      `json:"modifiers,omitempty"`
	Amount              float64       `json:"amount"`
	ExpectedAmount      float64       `json:"expectedAmount"`
	InsurancePaidAmount float64       `json:"insurancePaidAmount"`
	OutstandingAmount   float64       `json:"outstandingAmount"` // ExpectedAmount - InsurancePaidAmount
	RevenueCode         string        `json:"revenueCode"`
	Units               *string       `json:"units,omitempty"`
	Remark              *string       `json:"remark,omitempty"`            // Remark from payments
	UnpaidReason        *string       `json:"unpaidReason,omitempty"`      // Unpaid reason from payments
	AdjustmentReasons   *string       `json:"adjustmentReasons,omitempty"` // Adjustment reasons from payments
	TotalCheckNumber    *int          `json:"totalCheckNumber,omitempty"`  // Count of unique check numbers for this charge
	ClaimDetails        *ClaimDetails `json:"claimDetails,omitempty"`
}

type ClaimDetails struct {
	PatientID          string  `json:"patientId"`
	PatientName        string  `json:"patientName"`
	ClaimFromDate      *string `json:"claimFromDate,omitempty"`
	ClaimToDate        *string `json:"claimToDate,omitempty"`
	ClaimEnteredDate   *string `json:"claimEnteredDate,omitempty"`
	ClaimType          string  `json:"claimType"`
	PrimaryPayerName   string  `json:"primaryPayerName"`
	PrimaryPayerID     string  `json:"primaryPayerId"`
	SecondaryPayerName *string `json:"secondaryPayerName,omitempty"`
	SecondaryPayerID   *string `json:"secondaryPayerId,omitempty"`
	TertiaryPayerName  *string `json:"tertiaryPayerName,omitempty"`
	TertiaryPayerID    *string `json:"tertiaryPayerId,omitempty"`
	FacilityPosCode    *string `json:"facilityPosCode,omitempty"`
	ChargePosCode      *string `json:"chargePosCode,omitempty"`
}

type Payment struct {
	ID                     string   `json:"id"`
	ClaimID                string   `json:"claimId"`
	PatientID              string   `json:"patientId"`
	PatientName            string   `json:"patientName"`
	TotalClaimCharge       float64  `json:"totalClaimCharge"`
	TotalPaidAmount        float64  `json:"totalPaidAmount"`
	ClaimStatus            string   `json:"claimStatus"`
	InsuranceName          string   `json:"insuranceName"`
	InsuranceID            string   `json:"insuranceId"`
	PaymentDate            *string  `json:"paymentDate,omitempty"`
	PostedDate             *string  `json:"postedDate,omitempty"`
	ChargeDate             *string  `json:"chargeDate,omitempty"`
	BprDate                *string  `json:"bprDate,omitempty"`
	Trn                    *string  `json:"trn,omitempty"`
	MonthNumber            int      `json:"monthNumber"`
	MonthName              string   `json:"monthName"`
	ChargeID               string   `json:"chargeId"`
	CptCode                *string  `json:"cptCode,omitempty"`
	Modifiers              []string `json:"modifiers,omitempty"`
	InsuranceAdjustment    float64  `json:"insuranceAdjustment"`
	CreditID               *string  `json:"creditId,omitempty"`
	Remark                 *string  `json:"remark,omitempty"`
	UnpaidReason           *string  `json:"unpaidReason,omitempty"`
	AdjustmentReasons      *string  `json:"adjustmentReasons,omitempty"`
	RevenueCode            *string  `json:"revenueCode,omitempty"`
	TotalExpectedAmount    float64  `json:"totalExpectedAmount"`
	TotalOutstandingAmount float64  `json:"totalOutstandingAmount"`
	Units                  int64    `json:"units"`
	IsRefund               bool     `json:"isRefund"`
	IsProviderAdjustment   bool     `json:"isProviderAdjustment"`
	IsEraAdjustment        bool     `json:"isEraAdjustment"`
	TotalCheckNumber       *int     `json:"totalCheckNumber,omitempty"` // Count of unique check numbers for this payment
}

type PaymentsResponse struct {
	Data       []Payment                 `json:"data"`
	Pagination domain.PaginationMetadata `json:"pagination"`
}

type InsuranceCheck struct {
	ID                       string  `json:"id"`
	CheckPayer               string  `json:"checkPayer"`
	CheckDate                *string `json:"checkDate,omitempty"`
	DepositDate              *string `json:"depositDate,omitempty"`
	CheckNumber              string  `json:"checkNumber"`
	CheckAmount              float64 `json:"checkAmount"`
	CheckAppliedAmount       float64 `json:"checkAppliedAmount"`
	ProviderAdjustmentAmount float64 `json:"providerAdjustmentAmount"`
	CheckRemainingAmount     float64 `json:"checkRemainingAmount"`
	CheckCommitted           bool    `json:"checkCommitted"`
	MonthNumber              int     `json:"monthNumber"`
	MonthName                string  `json:"monthName"`
}

type InsuranceChecksResponse struct {
	Data               []InsuranceCheck          `json:"data"`
	Pagination         domain.PaginationMetadata `json:"pagination"`
	TotalDepositAmount *float64                  `json:"totalDepositAmount,omitempty"`
}

type BankDepositMonthData struct {
	Month               string  `json:"month"`
	TotalClaimsNumber   int64   `json:"total_claims_number"`
	TotalClaimsAmount   float64 `json:"total_claims_amount"`
	TotalExpectedAmount float64 `json:"total_expected_amount"`
	TotalDepositAmount  float64 `json:"total_deposit_amount"`
	TotalAppliedAmount  float64 `json:"total_applied_amount"`
}

// DashboardData represents the dashboard data structure organized by month
type DashboardData map[string]MonthData

// MonthData represents aggregated data for a single month
type MonthData struct {
	TotalClaims   CategoryData `json:"totalClaims"`
	TotalPayments CategoryData `json:"totalPayments"`
	Outstanding   CategoryData `json:"outstanding"`
	TotalExpected CategoryData `json:"totalExpected"`
}

// CategoryData represents data broken down by category (IN/OUT/PEER)
type CategoryData struct {
	IN   TypeData `json:"IN"`
	OUT  TypeData `json:"OUT"`
	PEER TypeData `json:"PEER"`
}

// TypeData represents aggregated metrics for a category
type TypeData struct {
	Value                     int64    `json:"value"`                                  // Count of unique claims for this category
	TotalClaims               int64    `json:"total_claims,omitempty"`                 // Sum of total_claim values for unique claims (for totalClaims category)
	TotalAmount               float64  `json:"total_amount,omitempty"`                 // Sum of total_amount values for unique claims (for totalAmount category) - float64 with 2 decimals
	TotalPayments             float64  `json:"total_payments,omitempty"`               // Sum of total_payments values for unique claims (for totalPayments category) - float64 with 2 decimals
	TotalExpected             float64  `json:"total_expected,omitempty"`               // Sum of total_expected values for unique claims (for totalExpected category) - float64 with 2 decimals
	TotalExpectedWithPayments *float64 `json:"total_expected_with_payments,omitempty"` // Sum of expected_amount for claims with payments (even if payment is 0.0) - float64 with 2 decimals
	TotalOutstanding          *float64 `json:"total_outstanding,omitempty"`            // Sum of total_outstanding values for unique claims (for outstanding category) - float64 with 2 decimals (only set for outstanding category)
	TotalUnits                int64    `json:"total_units,omitempty"`                  // Sum of units for unique claims (for all categories)
}

// MonthNames is a constant array of month names
var MonthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

type PaymentCheck struct {
	ClaimID           string  `json:"claimId"`
	ChargeID          string  `json:"chargeId"`
	CheckNumber       string  `json:"checkNumber"`
	Amount            float64 `json:"amount"`
	Remark            string  `json:"remark"`
	UnpaidReason      string  `json:"unpaidReason"`
	AdjustmentReasons string  `json:"adjustmentReasons"`
	DepositDate       *string `json:"depositDate,omitempty"`
}

type PaymentChecksResponse struct {
	Data       []PaymentCheck            `json:"data"`
	Pagination domain.PaginationMetadata `json:"pagination"`
}

// DashboardDataRow represents aggregated dashboard data for claims
type DashboardDataRow struct {
	Year                            int
	MonthNumber                     int
	MonthName                       string
	ClaimCategory                   string
	UniqueClaims                    int64
	TotalUnits                      decimal.Decimal
	TotalAmount                     decimal.Decimal
	UniqueClaimsWithPayments        int64
	TotalUnitsWithPayments          decimal.Decimal
	TotalPaymetsAmount              decimal.Decimal
	TotalExpectedAmount             decimal.Decimal
	TotalExpectedAmountWithPayments decimal.Decimal
	UniqueClaimsWithOutstanding     int64
	TotalUnitsWithOutstanding       decimal.Decimal
	TotalOutstandingAmount          decimal.Decimal
}

// BankDepositDataRow represents aggregated bank deposit data
type BankDepositDataRow struct {
	MonthNumber         int
	MonthName           string
	TotalClaimsNumber   int64
	TotalClaimsAmount   decimal.Decimal
	TotalExpectedAmount decimal.Decimal
	TotalDepositAmount  decimal.Decimal
	TotalAppliedAmount  decimal.Decimal
}
