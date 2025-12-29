package dashboard

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"

	dbgen "github.com/aci/backend/gen/db"
	common "github.com/aci/backend/internal/core/domain/common"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *service) GetBankDepositData(ctx context.Context, year int) ([]dashboardtypes.BankDepositMonthData, error) {
	s.logger.Info("Service: Getting bank deposit data", "year", year)

	// Fetch bank deposit data from repository
	rows, err := s.repo.GetBankDepositData(ctx, int32(year))
	if err != nil {
		s.logger.Error("Service: Failed to get bank deposit data from repository", "year", year, "error", err)
		return nil, err
	}
	s.logger.Info("Service: Retrieved bank deposit data from repository", "year", year, "rowCount", len(rows))

	// Initialize month data structure with all 12 months
	monthDataMap := make(map[string]dashboardtypes.BankDepositMonthData)
	for _, monthName := range dashboardtypes.MonthNames {
		monthDataMap[monthName] = dashboardtypes.BankDepositMonthData{
			Month:               monthName,
			TotalClaimsNumber:   0,
			TotalClaimsAmount:   0.0,
			TotalExpectedAmount: 0.0,
			TotalDepositAmount:  0.0,
			TotalAppliedAmount:  0.0,
		}
	}

	// Process rows and populate data
	for _, row := range rows {
		// Get month name from month number
		var monthName string
		if row.MonthNumber > 0 && row.MonthNumber <= 12 {
			monthName = dashboardtypes.MonthNames[row.MonthNumber-1]
		}
		if monthName == "" {
			// Fallback: use month name from DB
			monthName = strings.TrimSpace(row.MonthName)
		}

		// Skip if invalid month
		if monthName == "" {
			continue
		}

		// Check if month exists in data map
		if _, exists := monthDataMap[monthName]; !exists {
			continue
		}

		// Convert amounts to float64 using InexactFloat64 for reliable conversion
		totalClaimsAmountFloat := row.TotalClaimsAmount.InexactFloat64()
		totalExpectedAmountFloat := row.TotalExpectedAmount.InexactFloat64()
		totalDepositAmountFloat := row.TotalDepositAmount.InexactFloat64()
		totalAppliedAmountFloat := row.TotalAppliedAmount.InexactFloat64()
		
		// Handle NaN values (shouldn't happen, but safety check)
		if math.IsNaN(totalExpectedAmountFloat) || math.IsInf(totalExpectedAmountFloat, 0) {
			totalExpectedAmountFloat = 0.0
		}
		if math.IsNaN(totalAppliedAmountFloat) || math.IsInf(totalAppliedAmountFloat, 0) {
			totalAppliedAmountFloat = 0.0
		}

		// Update month data
		monthDataMap[monthName] = dashboardtypes.BankDepositMonthData{
			Month:               monthName,
			TotalClaimsNumber:   row.TotalClaimsNumber,
			TotalClaimsAmount:   totalClaimsAmountFloat,
			TotalExpectedAmount: totalExpectedAmountFloat,
			TotalDepositAmount:  totalDepositAmountFloat,
			TotalAppliedAmount:  totalAppliedAmountFloat,
		}
	}

	// Convert map to slice ordered by month
	result := make([]dashboardtypes.BankDepositMonthData, 0, 12)
	for _, monthName := range dashboardtypes.MonthNames {
		if data, exists := monthDataMap[monthName]; exists {
			result = append(result, data)
		}
	}

	return result, nil
}


func (s *service) GetBankDepositClaims(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.ClaimsResponse, error) {
	s.logger.Info("Service: Getting paginated bank deposit claims", "year", year, "month", month, "search", search, "limit", limit, "cursor", cursor)

	// Validate inputs
	if limit < 1 {
		limit = 15
	}
	if limit > 100 {
		limit = 100
	}

	// Run count and paginated queries in parallel for better performance
	var claimRows []dbgen.GetBankDepositClaimsPaginatedRow
	var totalCount int64
	var claimsErr, countErr error

	var wg sync.WaitGroup

	// Fetch paginated claims
	wg.Add(1)
	go func() {
		defer wg.Done()
		claimRows, claimsErr = s.repo.GetBankDepositClaimsPaginated(ctx, int32(year), int32(month), search, int32(limit), cursor)
		if claimsErr != nil {
			s.logger.Error("Service: Failed to get paginated bank deposit claims from repository", "error", claimsErr)
		}
	}()

	// Fetch total count (always fetch to ensure it's available on all pages)
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCount, countErr = s.repo.GetBankDepositClaimsCount(ctx, int32(year), int32(month), search)
		if countErr != nil {
			s.logger.Error("Service: Failed to get bank deposit claims count from repository", "error", countErr)
		}
	}()

	wg.Wait()

	// Check for errors
	if claimsErr != nil {
		return nil, claimsErr
	}
	if countErr != nil {
		return nil, countErr
	}

	// Convert claim rows to claims (similar to GetClaims method)
	claims := make([]dashboardtypes.Claim, 0, len(claimRows))
	var nextCursor string

	for i, claimRow := range claimRows {
		// Format dates
		var claimFromDate string
		if claimRow.ServiceDateFrom.Valid {
			claimFromDate = formatDate(claimRow.ServiceDateFrom)
		}
		var claimToDate string
		if claimRow.ServiceDateTo.Valid {
			claimToDate = formatDate(claimRow.ServiceDateTo)
		}
		var claimEnteredDate *string
		if claimRow.ClaimEnteredDate.Valid {
			dateStr := formatDate(claimRow.ClaimEnteredDate)
			claimEnteredDate = &dateStr
		}

		claimType := ""
		if claimRow.TypeOfService.Valid {
			claimType = claimRow.TypeOfService.String
		}

		// Convert aggregated values to float64
		totalClaimsAmount, _ := numericToDecimal(claimRow.TotalClaims).Float64()
		totalExpectedAmount, _ := numericToDecimal(claimRow.TotalExpectedAgg).Float64()
		totalPaymentAmount, _ := numericToDecimal(claimRow.TotalPayment).Float64()
		totalOutstandingAmount, _ := numericToDecimal(claimRow.OutstandingAmount).Float64()
		totalUnitsInt := numericToDecimal(claimRow.TotalUnits).Truncate(0).IntPart()

		// Convert interface{} fields to strings
		patientID := interfaceToString(claimRow.PatientID)
		patientName := interfaceToString(claimRow.PatientName)
		primaryPayerName := interfaceToString(claimRow.PrimaryPayerName)
		primaryPayerID := interfaceToString(claimRow.PrimaryPayerID)
		secondaryPayerName := interfaceToString(claimRow.SecondaryPayerName)
		secondaryPayerID := interfaceToString(claimRow.SecondaryPayerID)
		tertiaryPayerName := interfaceToString(claimRow.TertiaryPayerName)
		tertiaryPayerID := interfaceToString(claimRow.TertiaryPayerID)

		var cptCodes []string
		if val, ok := claimRow.CptCodes.([]string); ok {
			cptCodes = val
		}
		var modifiers []string
		if val, ok := claimRow.Modifiers.([]string); ok {
			modifiers = val
		}

		// Generate cursor from last row
		if i == len(claimRows)-1 {
			cursorDate := ""
			if claimRow.ServiceDateFrom.Valid {
				cursorDate = formatDate(claimRow.ServiceDateFrom)
			}
			nextCursor = fmt.Sprintf("%s|%s", cursorDate, claimRow.ClaimID)
		}

		// Get total check number count (always set, even if 0)
		checkNum := int(claimRow.TotalCheckNumber)
		totalCheckNumber := &checkNum

		claim := dashboardtypes.Claim{
			ClaimID:          claimRow.ClaimID,
			ChargeID:         "",
			PatientID:        patientID,
			PatientName:      patientName,
			TotalClaims:      &totalClaimsAmount,
			TotalExpected:    &totalExpectedAmount,
			TotalPayment:     &totalPaymentAmount,
			TotalOutstanding: &totalOutstandingAmount,
			TotalUnits:       &totalUnitsInt,
			ClaimFromDate:    claimFromDate,
			ClaimToDate:      claimToDate,
			ClaimEnteredDate: claimEnteredDate,
			ClaimType:        claimType,
			PrimaryPayerName: primaryPayerName,
			PrimaryPayerID:   primaryPayerID,
			ChargeCount:      int(claimRow.ChargeCount),
			TotalCharges:     int(claimRow.TotalCharges),
			CptCodes:         cptCodes,
			Modifiers:        modifiers,
			TotalCheckNumber: totalCheckNumber,
		}

		if secondaryPayerName != "" {
			claim.SecondaryPayerName = &secondaryPayerName
		}
		if secondaryPayerID != "" {
			claim.SecondaryPayerID = &secondaryPayerID
		}
		if tertiaryPayerName != "" {
			claim.TertiaryPayerName = &tertiaryPayerName
		}
		if tertiaryPayerID != "" {
			claim.TertiaryPayerID = &tertiaryPayerID
		}

		claims = append(claims, claim)
	}

	// Determine if there are more items
	hasMore := len(claimRows) == limit

	// Calculate page for response (for cursor-based, use 1 as default)
	page := 1
	if cursor != "" {
		// Estimate page based on total count and current position
		// This is approximate since we don't track exact page with cursor
		page = 1
	}

	return &dashboardtypes.ClaimsResponse{
		Data: claims,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCount,
			Page:         page,
			Limit:        limit,
			TotalPages:   0, // Not applicable for cursor-based pagination
			HasMore:      hasMore,
			NextCursor:   nextCursor,
		},
	}, nil
}


func (s *service) GetBankDepositAppliedPayments(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.PaymentsResponse, error) {
	s.logger.Info("Service: Getting paginated bank deposit applied payments", "year", year, "month", month, "search", search, "limit", limit, "cursor", cursor)

	// Validate inputs
	if limit < 1 {
		limit = 15
	}
	if limit > 100 {
		limit = 100
	}

	// Run count and paginated queries in parallel for better performance
	var paymentRows []dbgen.GetBankDepositAppliedPaymentsPaginatedRow
	var totalCount int64
	var paymentsErr, countErr error

	var wg sync.WaitGroup

	// Fetch paginated payments
	wg.Add(1)
	go func() {
		defer wg.Done()
		paymentRows, paymentsErr = s.repo.GetBankDepositAppliedPaymentsPaginated(ctx, int32(year), int32(month), search, int32(limit), cursor)
		if paymentsErr != nil {
			s.logger.Error("Service: Failed to get paginated bank deposit applied payments from repository", "error", paymentsErr)
		}
	}()

	// Fetch total count (always fetch to ensure it's available on all pages)
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCount, countErr = s.repo.GetBankDepositAppliedPaymentsCount(ctx, int32(year), int32(month), search)
		if countErr != nil {
			s.logger.Error("Service: Failed to get bank deposit applied payments count from repository", "error", countErr)
		}
	}()

	wg.Wait()

	// Check for errors
	if paymentsErr != nil {
		return nil, paymentsErr
	}
	if countErr != nil {
		return nil, countErr
	}

	// Convert payment rows to payments (similar to GetPayments method)
	payments := make([]dashboardtypes.Payment, 0, len(paymentRows))
	var nextCursor string

	for i, paymentRow := range paymentRows {
		// Convert numeric types to float64
		totalClaimCharge, _ := numericToDecimal(paymentRow.TotalClaimCharge).Float64()
		totalPaidAmount, _ := numericToDecimal(paymentRow.TotalPaidAmount).Float64()
		insuranceAdjustment, _ := numericToDecimal(paymentRow.InsuranceAdjustment).Float64()
		totalExpectedAmount, _ := numericToDecimal(paymentRow.TotalExpectedAmount).Float64()
		unitsInt := numericToDecimal(paymentRow.Units).Truncate(0).IntPart()
		
		// Handle MonthNumber (interface{}) - need type assertion
		monthNumberInt := int64(0)
		if paymentRow.MonthNumber != nil {
			if monthNum, ok := paymentRow.MonthNumber.(pgtype.Numeric); ok {
				monthNumberInt = numericToDecimal(monthNum).Truncate(0).IntPart()
			}
		}

		// Convert interface{} fields to strings
		id := interfaceToString(paymentRow.ID)
		patientID := interfaceToString(paymentRow.PatientID)
		patientName := interfaceToString(paymentRow.PatientName)
		insuranceName := interfaceToString(paymentRow.InsuranceName)
		insuranceID := interfaceToString(paymentRow.InsuranceID)
		
		// Handle MonthName (interface{})
		monthName := interfaceToString(paymentRow.MonthName)

		// Format dates
		var paymentDate, postedDate, chargeDate, bprDate *string
		if paymentRow.PaymentDate != nil {
			dateStr := formatDateInterface(paymentRow.PaymentDate)
			if dateStr != "" {
				paymentDate = &dateStr
			}
		}
		if paymentRow.PostedDate != nil {
			dateStr := formatDateInterface(paymentRow.PostedDate)
			if dateStr != "" {
				postedDate = &dateStr
			}
		}
		if paymentRow.ChargeDate != nil {
			dateStr := formatDateInterface(paymentRow.ChargeDate)
			if dateStr != "" {
				chargeDate = &dateStr
			}
		}
		if paymentRow.BprDate != nil {
			dateStr := formatDateInterface(paymentRow.BprDate)
			if dateStr != "" {
				bprDate = &dateStr
			}
		}

		// Handle optional string fields
		var trn, cptCode, creditID, remark, unpaidReason, adjustmentReasons, revenueCode *string
		if paymentRow.Trn != nil {
			trnStr := interfaceToString(paymentRow.Trn)
			if trnStr != "" {
				trn = &trnStr
			}
		}
		if paymentRow.CptCode != nil {
			cptCodeStr := interfaceToString(paymentRow.CptCode)
			if cptCodeStr != "" {
				cptCode = &cptCodeStr
			}
		}
		if paymentRow.CreditID != nil {
			creditIDStr := interfaceToString(paymentRow.CreditID)
			if creditIDStr != "" {
				creditID = &creditIDStr
			}
		}
		if paymentRow.Remark != nil {
			remarkStr := interfaceToString(paymentRow.Remark)
			if remarkStr != "" {
				remark = &remarkStr
			}
		}
		if paymentRow.UnpaidReason != nil {
			unpaidReasonStr := interfaceToString(paymentRow.UnpaidReason)
			if unpaidReasonStr != "" {
				unpaidReason = &unpaidReasonStr
			}
		}
		// Get adjustment_reasons from paymentRow (field will be available after sqlc regenerate)
		paymentRowValue := reflect.ValueOf(paymentRow)
		if paymentRowValue.Kind() == reflect.Struct {
			adjReasonsField := paymentRowValue.FieldByName("AdjustmentReasons")
			if adjReasonsField.IsValid() {
				if adjReasonsInterface := adjReasonsField.Interface(); adjReasonsInterface != nil {
					adjReasonsStr := interfaceToString(adjReasonsInterface)
					if adjReasonsStr != "" {
						adjustmentReasons = &adjReasonsStr
					}
				}
			}
		}
		if paymentRow.RevenueCode != nil {
			revenueCodeStr := interfaceToString(paymentRow.RevenueCode)
			if revenueCodeStr != "" {
				revenueCode = &revenueCodeStr
			}
		}

		// Generate cursor from last row
		if i == len(paymentRows)-1 {
			paymentDateStr := ""
			if paymentDate != nil {
				paymentDateStr = *paymentDate
			}
			nextCursor = fmt.Sprintf("%s|%s|%s", paymentDateStr, paymentRow.ClaimID, paymentRow.ChargeID)
		}

		// Get total check number count (always set, even if 0)
		checkNum := int(paymentRow.TotalCheckNumber)
		totalCheckNumber := &checkNum

		payment := dashboardtypes.Payment{
			ID:                   id,
			ClaimID:              paymentRow.ClaimID,
			PatientID:            patientID,
			PatientName:          patientName,
			TotalClaimCharge:     totalClaimCharge,
			TotalPaidAmount:      totalPaidAmount,
			ClaimStatus:          paymentRow.ClaimStatus,
			InsuranceName:        insuranceName,
			InsuranceID:          insuranceID,
			PaymentDate:          paymentDate,
			PostedDate:           postedDate,
			ChargeDate:           chargeDate,
			BprDate:              bprDate,
			Trn:                  trn,
			MonthNumber:          int(monthNumberInt),
			MonthName:            monthName,
			ChargeID:             paymentRow.ChargeID,
			CptCode:              cptCode,
			Modifiers:            paymentRow.Modifiers,
			InsuranceAdjustment:  insuranceAdjustment,
			CreditID:             creditID,
			Remark:               remark,
			UnpaidReason:         unpaidReason,
			AdjustmentReasons:    adjustmentReasons,
			RevenueCode:          revenueCode,
			TotalExpectedAmount:  totalExpectedAmount,
			Units:                unitsInt,
			IsRefund:             paymentRow.IsRefund,
			IsProviderAdjustment: paymentRow.IsProviderAdjustment,
			IsEraAdjustment:      paymentRow.IsEraAdjustment,
			TotalCheckNumber:     totalCheckNumber,
		}

		payments = append(payments, payment)
	}

	// Determine if there are more items
	hasMore := len(paymentRows) == limit

	return &dashboardtypes.PaymentsResponse{
		Data: payments,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCount,
			Page:         1, // Cursor-based pagination doesn't use page numbers
			Limit:        limit,
			TotalPages:   0, // Not applicable for cursor-based pagination
			HasMore:      hasMore,
			NextCursor:   nextCursor,
		},
	}, nil
}

