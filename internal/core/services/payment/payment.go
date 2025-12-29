package payment

import (
	"time"

	"github.com/shopspring/decimal"
)

type Payment struct {
	ClaimId              string
	PatientId            string
	Trn                  string
	TotalClaimCharge     decimal.Decimal
	TotalPaidAmount      decimal.Decimal
	ClaimStatus          string
	InsuranceName        string
	InsuranceId          string
	RawText              string
	ServiceLines         []ServiceLine
	PaymentDate          time.Time        // from BPR16
	PostedDate           time.Time        // from DTM segment
	ChargeDate           time.Time        // from DTM segment
	BPRDate              time.Time        // from BPR segment
	ClaimLevelAdjustment []AdjustmentCode `json:"claim_level_adjustment"`
}

type ServiceLine struct {
	LineNumber     string           `json:"line_number"`
	CptCode        string           `json:"cpt_code"`
	Modifiers      []string         `json:"modifiers"`
	Units          int              `json:"units"`
	Amount         decimal.Decimal  `json:"amount"`
	PaidAmount     decimal.Decimal  `json:"paid_amount"`
	AdjustmentCode []AdjustmentCode `json:"adjustment_code"`
}
type AdjustmentCode struct {
	GroupCode  string          `json:"group_code"`
	ReasonCode string          `json:"reason_code"`
	Amount     decimal.Decimal `json:"amount"`
	Quantity   string          `json:"quantity"`
}

type NewPayment struct {
	PaymentReceivedDate time.Time
	DepositDate         time.Time
	PaymentEnteredDate  time.Time
	ChargeFromDate      time.Time
	ClaimId             string
	ChargeId            string
	CptCode             string
	Modifiers           []string
	// money
	InsurancePaidA      decimal.Decimal
	InsuranceAdjustment decimal.Decimal
	ChargeAmount        decimal.Decimal
	//
	CreditId     string
	PaymentCheck string
	Remark       string
	UnpaidReason string
	//
	PayerId     string
	PaymentName string
	//
	PatientId   string
	PatientName string
	RevenueCode string
	//
	IsRefund             bool
	IsProviderAdjustment bool
	IsEraAdjustment      bool
}
