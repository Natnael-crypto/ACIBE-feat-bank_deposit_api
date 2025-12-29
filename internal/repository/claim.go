package repository

import (
	"context"
	"encoding/json"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/core/services/claim"

	"github.com/jackc/pgx/v5/pgtype"
)

// claim_id, --1
// claim_type, --2
// total_charge, --3
// claim_submission_date, --4
// service_date_from, --5
// service_date_to, --6
// patient_id, --7
// patient_name, --8
// place_of_service, --9
// service_lines, --10
// type_of_service, --11
// primary_insurance_name, --12
// secondary_insurance_id, --13
// raw_text --14

func (s *Store) CreateClaims(ctx context.Context, claims []claim.Claim) error {
	params := dbgen.CreateClaimParams{
		Column1:  make([]string, len(claims)),         // claim_id
		Column2:  make([]string, len(claims)),         // claim_frequency
		Column3:  make([]string, len(claims)),         // original_claim_number
		Column4:  make([]pgtype.Numeric, len(claims)), // total_charge
		Column5:  make([]pgtype.Date, len(claims)),    // claim_submission_date
		Column6:  make([]pgtype.Date, len(claims)),    // service_date_from
		Column7:  make([]pgtype.Date, len(claims)),    // service_date_to
		Column8:  make([]string, len(claims)),         // patient_id
		Column9:  make([]string, len(claims)),         // patient_name
		Column10: make([][]byte, len(claims)),         // service_lines
		Column11: make([]string, len(claims)),         // type_of_service
		Column12: make([]string, len(claims)),         // primary_insurance_name
		Column13: make([]string, len(claims)),         // primary_insurance_id
		Column14: make([]string, len(claims)),         // secondary_insurance_name
		Column15: make([]string, len(claims)),         // secondary_insurance_id
		Column16: make([]string, len(claims)),         // raw_text
	}

	for i, c := range claims {

		jsonServicelines, err := json.Marshal(c.ServiceLines)
		if err != nil {
			return err
		}

		// Set params (Column4 will be set after computation)
		params.Column1[i] = c.ClaimId
		params.Column2[i] = c.ClaimFrequency
		params.Column3[i] = c.OriginalClaimNumber
		params.Column4[i] = DecimalToNumeric(c.TotalCharge)
		params.Column5[i] = toDate(c.ClaimSubittionDate)
		params.Column6[i] = toDate(c.ServiceDateFrom)
		params.Column7[i] = toDate(c.ServiceDateTo)
		params.Column8[i] = c.PatientId
		params.Column9[i] = c.PatientName
		params.Column10[i] = jsonServicelines
		params.Column11[i] = c.TypeOfService
		params.Column12[i] = c.PrimaryInsuranceName
		params.Column13[i] = c.PrimaryInsuranceId
		params.Column14[i] = c.SecondaryInsuranceName
		params.Column15[i] = c.SecondaryInsuranceId
		params.Column16[i] = c.RawText
	}

	return s.queries.CreateClaim(ctx, params)
}

func (s *Store) CreateNewClaims(ctx context.Context, claims []claim.NewClaim) error {
	if len(claims) == 0 {
		return nil
	}

	params := dbgen.CreateNewClaimsParams{
		Column1:  make([]string, len(claims)),         // claim_id
		Column2:  make([]string, len(claims)),         // charge_id
		Column3:  make([]pgtype.Numeric, len(claims)), // total_claim
		Column4:  make([]pgtype.Date, len(claims)),    // claim_entered_date
		Column5:  make([]pgtype.Date, len(claims)),    // claim_from_date
		Column6:  make([]pgtype.Date, len(claims)),    // claim_to_date
		Column7:  make([]string, len(claims)),         // claim_type
		Column8:  make([][]byte, len(claims)),         // primary_payer
		Column9:  make([][]byte, len(claims)),         // secondary_payer
		Column10: make([][]byte, len(claims)),         // tertiary_payer
		Column11: make([]string, len(claims)),         // patient_id
		Column12: make([]string, len(claims)),         // patient_name
		Column13: make([]string, len(claims)),         // cpt_code
		Column14: make([][]byte, len(claims)),         // modifiers (JSONB array)
		Column15: make([]pgtype.Numeric, len(claims)), // amount
		Column16: make([]string, len(claims)),         // units
		Column17: make([]string, len(claims)),         // revenue_code
		Column18: make([]string, len(claims)),         // facility_pos_code
		Column19: make([]string, len(claims)),         // charge_pos_code
		Column20: make([]string, len(claims)),         // value_code_1
		Column21: make([]int32, len(claims)),          // value_amount_1
		Column22: make([]string, len(claims)),         // claim_status
		Column23: make([]pgtype.Numeric, len(claims)), // expected_amount
	}

	for i, c := range claims {
		params.Column1[i] = c.ClaimId
		params.Column2[i] = c.ChargeId
		params.Column3[i] = DecimalToNumeric(c.TotalClaim)
		params.Column4[i] = toDate(c.ClaimEnteredDate)
		params.Column5[i] = toDate(c.ClaimFromDate)
		params.Column6[i] = toDate(c.ClaimToDate)
		params.Column7[i] = c.ClaimType

		// Handle nullable JSONB fields
		if c.PrimaryPayer != nil {
			primaryPayerBytes, err := json.Marshal(c.PrimaryPayer)
			if err != nil {
				return err
			}
			params.Column8[i] = primaryPayerBytes
		}
		if c.SecondaryPayer != nil {
			secondaryPayerBytes, err := json.Marshal(c.SecondaryPayer)
			if err != nil {
				return err
			}
			params.Column9[i] = secondaryPayerBytes
		}
		if c.TertiaryPayer != nil {
			tertiaryPayerBytes, err := json.Marshal(c.TertiaryPayer)
			if err != nil {
				return err
			}
			params.Column10[i] = tertiaryPayerBytes
		}

		params.Column11[i] = c.PatientId
		params.Column12[i] = c.PatientName
		params.Column13[i] = c.CptCode

		// Marshal modifiers array to JSONB
		modifiersJSON, err := json.Marshal(c.Modifiers)
		if err != nil {
			return err
		}
		params.Column14[i] = modifiersJSON

		params.Column15[i] = DecimalToNumeric(c.Amount)
		params.Column16[i] = c.Units
		params.Column17[i] = c.RevenueCode
		params.Column18[i] = c.FacilityPosCode
		params.Column19[i] = c.ChargePosCode
		params.Column20[i] = c.ValueCode1
		params.Column21[i] = c.ValueAmount1
		params.Column22[i] = c.ClaimStatus
		params.Column23[i] = DecimalToNumeric(c.ExpectedAmount)
	}

	return s.queries.CreateNewClaims(ctx, params)
}
