package claim

import (
	"time"

	"github.com/shopspring/decimal"
)

type PlaceOfService string

const (
	PlaceOfServiceOnsite  PlaceOfService = "1540"
	PlaceOfServiceOffSite PlaceOfService = "1080"
)

type Claim struct {
	ClaimId             string
	ClaimFrequency      string
	OriginalClaimNumber string
	TotalCharge         decimal.Decimal
	//
	ClaimSubittionDate time.Time
	//
	ServiceDateFrom time.Time
	ServiceDateTo   time.Time
	//

	PrimaryInsuranceName   string
	PrimaryInsuranceId     string
	SecondaryInsuranceName string
	SecondaryInsuranceId   string
	// ThirityInsuranceName   string
	//
	PatientId   string
	PatientName string
	//
	TypeOfService string
	//
	PlaceOfService PlaceOfService
	ServiceLines   []ServiceLine
	RawText        string
}
type ServiceLine struct {
	LineNumber string
	CPTCode    string
	Modifiers  []string
	Amount     decimal.Decimal
	Units      int
	Code       string
}

type NewClaim struct {
	ClaimId          string
	ChargeId         string
	TotalClaim       decimal.Decimal
	ClaimEnteredDate time.Time
	ClaimFromDate    time.Time
	ClaimToDate      time.Time
	ClaimType        string
	//
	PrimaryPayer   *Payer
	SecondaryPayer *Payer
	TertiaryPayer  *Payer
	//
	PatientId   string
	PatientName string
	//
	CptCode         string
	Modifiers       []string
	Amount          decimal.Decimal
	Units           string
	RevenueCode     string
	FacilityPosCode string
	ChargePosCode   string
	ValueCode1      string
	ValueAmount1    int32
	ClaimStatus     string
	//
	ExpectedAmount decimal.Decimal
}
type Payer struct {
	Id             string
	Name           string
	Type           string
	ExpectedAmount decimal.Decimal
}
