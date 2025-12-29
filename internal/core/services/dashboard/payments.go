package dashboard

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	dbgen "github.com/aci/backend/gen/db"
	common "github.com/aci/backend/internal/core/domain/common"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
)

func (s *service) GetPayments(ctx context.Context, year *int, month *int, excludePeer bool, peerOnly bool, search *string, claimType string, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string) (*dashboardtypes.PaymentsResponse, error) {
	s.logger.Info("Service: Getting paginated payments", "year", year, "month", month, "excludePeer", excludePeer, "peerOnly", peerOnly, "search", search, "claimType", claimType, "page", page, "limit", limit)

	// Convert 'all' to empty string so SQL query returns all types (NULL/empty means all)
	if claimType == "all" {
		claimType = ""
	}

	// If peerOnly is true, override claimType to an empty string to get all peer payments regardless of type
	if peerOnly {
		claimType = ""
	}

	var yearNum int32 = 0
	if year != nil {
		yearNum = int32(*year)
	}

	var monthNum int32 = 0
	if month != nil {
		monthNum = int32(*month)
	}

	searchVal := ""
	if search != nil {
		searchVal = *search
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Handle limit=0 case (export all data) - use max int32 to get all records
	effectiveLimit := limit
	if limit == 0 {
		effectiveLimit = 2147483647 // Max int32 value to get all records
	}

	// Run count and paginated queries in parallel for better performance
	var paymentRows []dbgen.GetDashboardPaymentsPaginatedRow
	var totalCount int64
	var paymentsErr, countErr error

	var wg sync.WaitGroup

	// Fetch paginated payments
	wg.Add(1)
	go func() {
		defer wg.Done()
		paymentRows, paymentsErr = s.repo.GetDashboardPaymentsPaginated(ctx, yearNum, monthNum, excludePeer, peerOnly, searchVal, int32(effectiveLimit), int32(offset), claimType, claimsExpectedFilter, paymentStatusFilter)
		if paymentsErr != nil {
			s.logger.Error("Service: Failed to get paginated payments from repository", "error", paymentsErr)
		}
	}()

	// Fetch total count
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCount, countErr = s.repo.GetDashboardPaymentsCount(ctx, yearNum, monthNum, excludePeer, peerOnly, searchVal, claimType, claimsExpectedFilter, paymentStatusFilter)
		if countErr != nil {
			s.logger.Error("Service: Failed to get payments count from repository", "error", countErr)
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

	// Convert payment rows to payments
	payments := make([]dashboardtypes.Payment, 0, len(paymentRows))
	for _, paymentRow := range paymentRows {
		// Convert numeric types to float64
		totalClaimCharge, _ := numericToDecimal(paymentRow.TotalClaimCharge).Float64()
		totalPaidAmount, _ := numericToDecimal(paymentRow.TotalPaidAmount).Float64()
		insuranceAdjustment, _ := numericToDecimal(paymentRow.InsuranceAdjustment).Float64()
		totalExpectedAmount, _ := numericToDecimal(paymentRow.TotalExpectedAmount).Float64()
		totalOutstandingAmount, _ := numericToDecimal(paymentRow.TotalOutstandingAmount).Float64()
		unitsInt := numericToDecimal(paymentRow.Units).Truncate(0).IntPart()
		monthNumberInt := numericToDecimal(paymentRow.MonthNumber).Truncate(0).IntPart()

		// Convert interface{} fields to strings
		id := interfaceToString(paymentRow.ID)
		patientID := interfaceToString(paymentRow.PatientID)
		patientName := interfaceToString(paymentRow.PatientName)
		insuranceName := interfaceToString(paymentRow.InsuranceName)
		insuranceID := interfaceToString(paymentRow.InsuranceID)

		// Format dates - handle different types
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
		if len(paymentRow.Trn) > 0 {
			trnStr := string(paymentRow.Trn)
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

		// Get total check number count (always set, even if 0)
		var totalCheckNumber *int
		if paymentRow.TotalCheckNumber > 0 {
			checkNum := int(paymentRow.TotalCheckNumber)
			totalCheckNumber = &checkNum
		} else if paymentRow.TotalCheckNumber == 0 {
			// Set to 0 explicitly (not nil) so frontend can display it
			zero := 0
			totalCheckNumber = &zero
		}

		// Create payment object
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
			MonthName:            paymentRow.MonthName,
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
			TotalOutstandingAmount: totalOutstandingAmount,
			Units:                unitsInt,
			IsRefund:             paymentRow.IsRefund,
			IsProviderAdjustment: paymentRow.IsProviderAdjustment,
			IsEraAdjustment:      paymentRow.IsEraAdjustment,
			TotalCheckNumber:     totalCheckNumber,
		}

		payments = append(payments, payment)
	}

	// Calculate total pages (handle limit = 0 case for exports)
	var totalPages int
	if limit == 0 {
		totalPages = 1 // When limit is 0, all data is on one "page"
	} else {
		totalPages = int((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &dashboardtypes.PaymentsResponse{
		Data: payments,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCount,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		},
	}, nil
}


func (s *service) GetPaymentsByCheckNumber(ctx context.Context, checkNumber string, year int, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) (*dashboardtypes.PaymentsResponse, error) {
	// Validate inputs
	if checkNumber == "" {
		return nil, fmt.Errorf("check number is required")
	}

	if page < 1 {
		page = 1
	}
	
	// Handle limit=0 case (export all data) - use max int32 to get all records
	effectiveLimit := limit
	if limit == 0 {
		effectiveLimit = 2147483647 // Max int32 value to get all records
	} else if limit < 1 {
		effectiveLimit = 15
	} else if limit > 100 {
		effectiveLimit = 100
	}

	offset := (page - 1) * effectiveLimit

	var paymentRows []dbgen.GetPaymentsByCheckNumberRow
	var totalCount int64
	var paymentsErr, countErr error

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		paymentRows, paymentsErr = s.repo.GetPaymentsByCheckNumberPaginated(ctx, checkNumber, int32(year), int32(effectiveLimit), int32(offset), claimsExpectedFilter, paymentStatusFilter, search)
		if paymentsErr != nil {
			s.logger.Error("Service: Failed to get paginated payments by check number from repository", "error", paymentsErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCount, countErr = s.repo.GetPaymentsByCheckNumberCount(ctx, checkNumber, int32(year), claimsExpectedFilter, paymentStatusFilter, search)
		if countErr != nil {
			s.logger.Error("Service: Failed to get payments by check number count from repository", "error", countErr)
		}
	}()

	wg.Wait()

	if paymentsErr != nil {
		return nil, fmt.Errorf("failed to get payments by check number: %w", paymentsErr)
	}
	if countErr != nil {
		return nil, fmt.Errorf("failed to get payments by check number count: %w", countErr)
	}

	// Convert database rows to Payment structs
	payments := make([]dashboardtypes.Payment, 0, len(paymentRows))
	for _, paymentRow := range paymentRows {
		// Convert numeric types to float64
		totalClaimCharge, _ := numericToDecimal(paymentRow.TotalClaimCharge).Float64()
		totalPaidAmount, _ := numericToDecimal(paymentRow.TotalPaidAmount).Float64()
		insuranceAdjustment, _ := numericToDecimal(paymentRow.InsuranceAdjustment).Float64()
		totalExpectedAmount, _ := numericToDecimal(paymentRow.TotalExpectedAmount).Float64()
		totalOutstandingAmount, _ := numericToDecimal(paymentRow.TotalOutstandingAmount).Float64()
		unitsInt := numericToDecimal(paymentRow.Units).Truncate(0).IntPart()
		monthNumberInt := numericToDecimal(paymentRow.MonthNumber).Truncate(0).IntPart()

		// Convert interface{} fields to strings
		id := interfaceToString(paymentRow.ID)
		patientID := interfaceToString(paymentRow.PatientID)
		patientName := interfaceToString(paymentRow.PatientName)
		insuranceName := interfaceToString(paymentRow.InsuranceName)
		insuranceID := interfaceToString(paymentRow.InsuranceID)

		// Format dates - handle different types
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
		if paymentRow.Trn != "" {
			trn = &paymentRow.Trn
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

		// Get total check number count (always set, even if 0)
		var totalCheckNumber *int
		paymentRowValueForCheck := reflect.ValueOf(paymentRow)
		if paymentRowValueForCheck.Kind() == reflect.Struct {
			checkNumField := paymentRowValueForCheck.FieldByName("TotalCheckNumber")
			if checkNumField.IsValid() && checkNumField.Kind() == reflect.Int32 {
				checkNumInt32 := int32(checkNumField.Int())
				if checkNumInt32 > 0 {
					checkNum := int(checkNumInt32)
					totalCheckNumber = &checkNum
				} else if checkNumInt32 == 0 {
					zero := 0
					totalCheckNumber = &zero
				}
			}
		}

		// Create payment object
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
			MonthName:            paymentRow.MonthName,
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
			TotalOutstandingAmount: totalOutstandingAmount,
			Units:                unitsInt,
			IsRefund:             paymentRow.IsRefund,
			IsProviderAdjustment: paymentRow.IsProviderAdjustment,
			IsEraAdjustment:      paymentRow.IsEraAdjustment,
			TotalCheckNumber:     totalCheckNumber,
		}

		payments = append(payments, payment)
	}

	// Calculate total pages (handle limit = 0 case for exports)
	var totalPages int
	if limit == 0 {
		totalPages = 1 // When limit is 0, all data is on one "page"
	} else {
		totalPages = int((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &dashboardtypes.PaymentsResponse{
		Data: payments,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCount,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		},
	}, nil
}

