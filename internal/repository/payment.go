package repository

import (
	"context"
	"encoding/json"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/core/services/payment"
	"github.com/jackc/pgx/v5/pgtype"
)

// unnest($1::text[]), -- claim_id
// unnest($2::text[]), -- patient_id
// unnest($3::numeric[]), -- total_claim_charge
// unnest($4::numeric[]), -- total_paid_amount
// unnest($5::text[]), -- claim_status
// unnest($6::text[]), -- insurance_name
// unnest($7::text[]), -- insurance_id
// unnest($8::text[]), -- raw_text
// unnest($9::jsonb[]), -- service_lines
// unnest($10::timestamptz[]), -- payment_date
// unnest($11::timestamptz[]), -- posted_date
// unnest($12::timestamptz[]), -- charge_date
// unnest($13::timestamptz[]) -- bpr_date
func (s *Store) CreatePayments(ctx context.Context, payments []payment.Payment) error {
	params := dbgen.CreatePaymentParams{
		Column1:  make([]string, len(payments)),         // claim_id
		Column2:  make([]string, len(payments)),         // patient_id
		Column3:  make([]pgtype.Numeric, len(payments)), // total_claim_charge
		Column4:  make([]pgtype.Numeric, len(payments)), // total_paid_amount
		Column5:  make([]string, len(payments)),         // claim_status
		Column6:  make([]string, len(payments)),         // insurance_name
		Column7:  make([]string, len(payments)),         // insurance_id
		Column8:  make([]string, len(payments)),         // raw_text
		Column9:  make([][]byte, len(payments)),         // service_lines
		Column10: make([]pgtype.Date, len(payments)),    // payment_date
		Column11: make([]pgtype.Date, len(payments)),    // posted_date
		Column12: make([]pgtype.Date, len(payments)),    // charge_date
		Column13: make([]pgtype.Date, len(payments)),    // bpr_date
		Column14: make([]string, len(payments)),         // trn
		Column15: make([][]byte, len(payments)),         // claim_level_adjustment
	}
	for i, p := range payments {
		jsonServicelines, err := json.Marshal(p.ServiceLines)
		if err != nil {
			return err
		}
		jsonClaimLevelAdjustment, err := json.Marshal(p.ClaimLevelAdjustment)
		if err != nil {
			return err
		}
		params.Column1[i] = p.ClaimId
		params.Column2[i] = p.PatientId
		params.Column3[i] = DecimalToNumeric(p.TotalClaimCharge)
		params.Column4[i] = DecimalToNumeric(p.TotalPaidAmount)
		params.Column5[i] = p.ClaimStatus
		params.Column6[i] = p.InsuranceName
		params.Column7[i] = p.InsuranceId
		params.Column8[i] = p.RawText
		params.Column9[i] = jsonServicelines
		params.Column10[i] = toDate(p.PaymentDate) // pgtype.Timestamptz{Time: p.PaymentDate, Valid: true}
		params.Column11[i] = toDate(p.PostedDate)  // pgtype.Timestamptz{Time: p.PostedDate, Valid: true}
		params.Column12[i] = toDate(p.ChargeDate)  // pgtype.Timestamptz{Time: p.ChargeDate, Valid: true}
		params.Column13[i] = toDate(p.BPRDate)     // pgtype.Timestamptz{Time: p.BPRDate, Valid: true}
		params.Column14[i] = p.Trn
		params.Column15[i] = jsonClaimLevelAdjustment
	}
	return s.queries.CreatePayment(ctx, params)
}

func (s *Store) CreateNewPayments(ctx context.Context, payments payment.NewPayment) error {
	// if deplist date is 1/1/0001

	// if payments.DepositDate.IsZero() {
	// 	return nil
	// }
	/// i want to add but if there is conflict and deposit date is 1/1/0001, then update the record write me trigger for this

	// if payments.DepositDate.IsZero() {
	// 	return nil
	// }
	err := s.queries.CreateNewPayment(ctx, dbgen.CreateNewPaymentParams{
		PaymentReceivedDate:  toDate(payments.PaymentReceivedDate),
		DepositDate:          toDate(payments.DepositDate),
		PaymentEnteredDate:   toDate(payments.PaymentEnteredDate),
		ChargeFromDate:       toDate(payments.ChargeFromDate),
		ClaimID:              payments.ClaimId,
		ChargeID:             payments.ChargeId,
		CptCode:              payments.CptCode,
		Modifiers:            payments.Modifiers,
		InsurancePaidA:       DecimalToNumeric(payments.InsurancePaidA),
		InsuranceAdjustment:  DecimalToNumeric(payments.InsuranceAdjustment),
		ChargeAmount:         DecimalToNumeric(payments.ChargeAmount),
		CreditID:             payments.CreditId,
		PaymentCheck:         payments.PaymentCheck,
		Remark:               payments.Remark,
		UnpaidReason:         payments.UnpaidReason,
		PayerID:              payments.PayerId,
		PaymentName:          payments.PaymentName,
		PatientID:            payments.PatientId,
		PatientName:          payments.PatientName,
		RevenueCode:          payments.RevenueCode,
		IsRefund:             payments.IsRefund,
		IsProviderAdjustment: payments.IsProviderAdjustment,
		IsEraAdjustment:      payments.IsEraAdjustment,
	})
	if err != nil {
		return err
	}
	return nil
}
