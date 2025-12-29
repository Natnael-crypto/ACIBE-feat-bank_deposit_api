package dashboard

import (
	dbgen "github.com/aci/backend/gen/db"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
)

// Helper function to get category label
func getCategoryLabel(category string) string {
	switch category {
	case "totalClaims":
		return "TotalClaims"
	case "totalPayments":
		return "TotalPayments"
	case "outstanding":
		return "Outstanding"
	case "totalExpected":
		return "TotalExpected"
	default:
		return category
	}
}

// Helper function to convert payment row to payment (needed for ExportPaymentsByClaimAndChargeToExcel)
func convertPaymentRowToPayment(row dbgen.GetPaymentsByClaimAndChargeRow) dashboardtypes.Payment {
	// Convert numeric types to float64
	totalClaimCharge, _ := numericToDecimal(row.TotalClaimCharge).Float64()
	totalPaidAmount, _ := numericToDecimal(row.TotalPaidAmount).Float64()
	insuranceAdjustment, _ := numericToDecimal(row.InsuranceAdjustment).Float64()
	totalExpectedAmount, _ := numericToDecimal(row.TotalExpectedAmount).Float64()
	unitsInt := numericToDecimal(row.Units).Truncate(0).IntPart()
	monthNumberInt := numericToDecimal(row.MonthNumber).Truncate(0).IntPart()

	// Convert interface{} fields to strings
	id := interfaceToString(row.ID)
	patientID := interfaceToString(row.PatientID)
	patientName := interfaceToString(row.PatientName)
	insuranceName := interfaceToString(row.InsuranceName)
	insuranceID := interfaceToString(row.InsuranceID)

	// Format dates - handle different types
	var paymentDate, postedDate, chargeDate, bprDate *string
	if row.PaymentDate != nil {
		dateStr := formatDateInterface(row.PaymentDate)
		if dateStr != "" {
			paymentDate = &dateStr
		}
	}
	if row.PostedDate != nil {
		dateStr := formatDateInterface(row.PostedDate)
		if dateStr != "" {
			postedDate = &dateStr
		}
	}
	if row.ChargeDate != nil {
		dateStr := formatDateInterface(row.ChargeDate)
		if dateStr != "" {
			chargeDate = &dateStr
		}
	}
	if row.BprDate != nil {
		dateStr := formatDateInterface(row.BprDate)
		if dateStr != "" {
			bprDate = &dateStr
		}
	}

	// Handle optional string fields
	var trn, cptCode, creditID, remark, unpaidReason, revenueCode *string
	if row.Trn != nil {
		trnStr := interfaceToString(row.Trn)
		if trnStr != "" {
			trn = &trnStr
		}
	}
	if row.CptCode != nil {
		cptCodeStr := interfaceToString(row.CptCode)
		if cptCodeStr != "" {
			cptCode = &cptCodeStr
		}
	}
	if row.CreditID != nil {
		creditIDStr := interfaceToString(row.CreditID)
		if creditIDStr != "" {
			creditID = &creditIDStr
		}
	}
	if row.Remark != nil {
		remarkStr := interfaceToString(row.Remark)
		if remarkStr != "" {
			remark = &remarkStr
		}
	}
	if row.UnpaidReason != nil {
		unpaidReasonStr := interfaceToString(row.UnpaidReason)
		if unpaidReasonStr != "" {
			unpaidReason = &unpaidReasonStr
		}
	}
	if row.RevenueCode != nil {
		revenueCodeStr := interfaceToString(row.RevenueCode)
		if revenueCodeStr != "" {
			revenueCode = &revenueCodeStr
		}
	}

	return dashboardtypes.Payment{
		ID:                  id,
		ClaimID:             row.ClaimID,
		PatientID:           patientID,
		PatientName:         patientName,
		ChargeID:            row.ChargeID,
		CptCode:             cptCode,
		RevenueCode:         revenueCode,
		InsuranceName:       insuranceName,
		InsuranceID:         insuranceID,
		Trn:                 trn,
		PaymentDate:         paymentDate,
		PostedDate:          postedDate,
		ChargeDate:          chargeDate,
		BprDate:             bprDate,
		TotalClaimCharge:    totalClaimCharge,
		TotalPaidAmount:     totalPaidAmount,
		TotalExpectedAmount: totalExpectedAmount,
		InsuranceAdjustment: insuranceAdjustment,
		Units:               unitsInt,
		MonthNumber:         int(monthNumberInt),
		MonthName:           row.MonthName,
		ClaimStatus:         row.ClaimStatus,
		CreditID:            creditID,
		Remark:              remark,
		UnpaidReason:        unpaidReason,
		Modifiers:           row.Modifiers,
		IsRefund:            row.IsRefund,
		IsProviderAdjustment: row.IsProviderAdjustment,
		IsEraAdjustment:     row.IsEraAdjustment,
	}
}
