package dashboard

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	common "github.com/aci/backend/internal/core/domain/common"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *service) GetClaims(ctx context.Context, year *int, month *int, category string, claimType string, excludePeer bool, search *string, peerOnly bool, page int, limit int, claimsExpectedFilter *string, paymentStatusFilter *string) (*dashboardtypes.ClaimsResponse, error) {
	s.logger.Info("Service: Getting paginated claims", "year", year, "month", month, "category", category, "type", claimType, "search", search, "page", page, "limit", limit)

	// Convert month name to number if needed
	var monthNum *int32
	if month != nil {
		m := int32(*month)
		monthNum = &m
	}

	var yearNum *int32
	if year != nil {
		y := int32(*year)
		yearNum = &y
	}

	var claimTypeFilter *string
	// Handle "all" or empty string as all types (no filter)
	// Normalize claim type: IN -> inpatient, OUT -> outpatient, Peer -> peer
	if claimType != "" && claimType != "all" {
		normalizedType := strings.ToLower(claimType)
		// Map common variations to expected values
		switch normalizedType {
		case "in":
			normalizedType = "inpatient"
		case "out":
			normalizedType = "outpatient"
		case "peer":
			normalizedType = "peer"
		}
		claimTypeFilter = &normalizedType
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Handle limit=0 case (export all data) - use max int32 to get all records
	effectiveLimit := limit
	if limit == 0 {
		effectiveLimit = 2147483647 // Max int32 value to get all records
	}

	// Prepare search filter
	var searchFilter *string
	if search != nil && *search != "" {
		searchFilter = search
	}

	// Run count and paginated queries in parallel for better performance
	var claimRows []dbgen.GetClaimsRow
	var totalCountRow dbgen.GetClaimsCountRow
	var claimsErr, countErr error

	var wg sync.WaitGroup

	// Fetch paginated claims
	wg.Add(1)
	go func() {
		defer wg.Done()
		claimRows, claimsErr = s.repo.GetClaims(ctx, yearNum, monthNum, category, claimTypeFilter, excludePeer, searchFilter, peerOnly, int32(effectiveLimit), int32(offset), claimsExpectedFilter, paymentStatusFilter)
		if claimsErr != nil {
			s.logger.Error("Service: Failed to get paginated claims from repository", "error", claimsErr)
		}
	}()

	// Fetch total count
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCountRow, countErr = s.repo.GetDashboardClaimsCount(ctx, yearNum, monthNum, category, claimTypeFilter, excludePeer, searchFilter, peerOnly, claimsExpectedFilter, paymentStatusFilter)
		if countErr != nil {
			s.logger.Error("Service: Failed to get claims count from repository", "error", countErr)
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

	// Convert claim rows to claims
	claims := make([]dashboardtypes.Claim, 0, len(claimRows))
	for _, claimRow := range claimRows {
		// Format dates - type assert interface{} to pgtype.Date
		// Handle different possible types: pgtype.Date, time.Time, string, or nil
		// Format all dates as MM/DD/YYYY (US date format)
		var claimFromDate string
		if claimRow.ServiceDateFrom != nil {
			switch v := claimRow.ServiceDateFrom.(type) {
			case pgtype.Date:
				claimFromDate = formatDate(v)
			case time.Time:
				claimFromDate = v.Format("01/02/2006") // MM/DD/YYYY
			case string:
				// If it's already a string, try to parse and reformat if needed
				if t, err := time.Parse("2006-01-02", v); err == nil {
					claimFromDate = t.Format("01/02/2006")
				} else if _, err := time.Parse("01/02/2006", v); err == nil {
					claimFromDate = v // Already in correct format
				} else {
					claimFromDate = v // Keep as is if can't parse
				}
			}
		}
		var claimToDate string
		if claimRow.ServiceDateTo != nil {
			switch v := claimRow.ServiceDateTo.(type) {
			case pgtype.Date:
				claimToDate = formatDate(v)
			case time.Time:
				claimToDate = v.Format("01/02/2006") // MM/DD/YYYY
			case string:
				// If it's already a string, try to parse and reformat if needed
				if t, err := time.Parse("2006-01-02", v); err == nil {
					claimToDate = t.Format("01/02/2006")
				} else if _, err := time.Parse("01/02/2006", v); err == nil {
					claimToDate = v // Already in correct format
				} else {
					claimToDate = v // Keep as is if can't parse
				}
			}
		}
		var claimEnteredDate *string
		if claimRow.ClaimSubmissionDate != nil {
			switch v := claimRow.ClaimSubmissionDate.(type) {
			case pgtype.Date:
				if v.Valid {
					dateStr := formatDate(v)
					claimEnteredDate = &dateStr
				}
			case time.Time:
				dateStr := v.Format("01/02/2006") // MM/DD/YYYY
				claimEnteredDate = &dateStr
			case string:
				if v != "" {
					// Try to parse and reformat if needed
					if t, err := time.Parse("2006-01-02", v); err == nil {
						dateStr := t.Format("01/02/2006")
						claimEnteredDate = &dateStr
					} else if _, err := time.Parse("01/02/2006", v); err == nil {
						dateStr := v // Copy to local variable before taking address
						claimEnteredDate = &dateStr
					} else {
						dateStr := v // Copy to local variable before taking address
						claimEnteredDate = &dateStr
					}
				}
			default:
				// Handle interface{} that might contain a string or other types
				// Try to convert to string using formatDateInterface
				if dateStr := formatDateInterface(claimRow.ClaimSubmissionDate); dateStr != "" {
					claimEnteredDate = &dateStr
				}
			}
		}

		// Use original database claim_type value (not the transformed one) - type assert interface{} to string
		claimType := ""
		if val, ok := claimRow.TypeOfService.(string); ok {
			claimType = val
		}

		// Convert aggregated values to float64
		totalClaimsAmount, _ := numericToDecimal(claimRow.TotalClaims).Float64()
		totalExpectedAmount, _ := numericToDecimal(claimRow.TotalExpectedAgg).Float64()
		totalPaymentAmount, _ := numericToDecimal(claimRow.TotalPayment).Float64()
		totalOutstandingAmount, _ := numericToDecimal(claimRow.OutstandingAmount).Float64()
		totalUnitsInt := numericToDecimal(claimRow.TotalUnits).Truncate(0).IntPart()

		// Convert interface{} fields to strings
		primaryPayerName := interfaceToString(claimRow.PrimaryPayerName)
		primaryPayerID := interfaceToString(claimRow.PrimaryPayerID)
		secondaryPayerName := interfaceToString(claimRow.SecondaryPayerName)
		secondaryPayerID := interfaceToString(claimRow.SecondaryPayerID)
		tertiaryPayerName := interfaceToString(claimRow.TertiaryPayerName)
		tertiaryPayerID := interfaceToString(claimRow.TertiaryPayerID)

		// Type assert PatientID and PatientName from interface{} to string
		patientID := ""
		if val, ok := claimRow.PatientID.(string); ok {
			patientID = val
		}
		patientName := ""
		if val, ok := claimRow.PatientName.(string); ok {
			patientName = val
		}

		// Type assert CptCodes and Modifiers from interface{} to []string
		var cptCodes []string
		if val, ok := claimRow.CptCodes.([]string); ok {
			cptCodes = val
		}
		var modifiers []string
		if val, ok := claimRow.Modifiers.([]string); ok {
			modifiers = val
		}

		// Get total_check_number (int32 from query) and convert to *int
		// Always set, even if 0, so frontend can display it
		var totalCheckNumber *int
		if claimRow.TotalCheckNumber > 0 {
			checkNum := int(claimRow.TotalCheckNumber)
			totalCheckNumber = &checkNum
		} else if claimRow.TotalCheckNumber == 0 {
			// Set to 0 explicitly (not nil) so frontend can display it
			zero := 0
			totalCheckNumber = &zero
		}

		// Create claim object
		claim := dashboardtypes.Claim{
			ClaimID: claimRow.ClaimID,
			// ChargeID is not available at the aggregated claim level in GetClaimsRow.
			// It will be populated when fetching charges for a specific claim.
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
			// TotalCheckNumber: number of unique checks associated with this claim
			TotalCheckNumber: totalCheckNumber,
		}

		// Add optional fields
		if secondaryPayerName != "" {
			claim.SecondaryPayerName = &secondaryPayerName
		}
		if secondaryPayerID != "" {
			claim.SecondaryPayerID = &secondaryPayerID
		}
		// Units is pgtype.Text, check Valid and get String
		if claimRow.Units.Valid && claimRow.Units.String != "" {
			unitsStr := claimRow.Units.String
			claim.Units = &unitsStr
		}
		// FacilityPosCode is pgtype.Text
		if claimRow.FacilityPosCode.Valid && claimRow.FacilityPosCode.String != "" {
			facilityPosCodeStr := claimRow.FacilityPosCode.String
			claim.FacilityPosCode = &facilityPosCodeStr
		}
		// ChargePosCode is pgtype.Text
		if claimRow.ChargePosCode.Valid && claimRow.ChargePosCode.String != "" {
			chargePosCodeStr := claimRow.ChargePosCode.String
			claim.ChargePosCode = &chargePosCodeStr
		}
		if tertiaryPayerName != "" {
			claim.TertiaryPayerName = &tertiaryPayerName
		}
		if tertiaryPayerID != "" {
			claim.TertiaryPayerID = &tertiaryPayerID
		}

		claims = append(claims, claim)
	}

	// Calculate total pages (handle limit = 0 case for exports)
	var totalPages int
	if limit == 0 {
		totalPages = 1 // When limit is 0, all data is on one "page"
	} else {
		totalPages = int((totalCountRow.TotalClaims + int64(limit) - 1) / int64(limit))
	}

	numericToFloat := func(n pgtype.Numeric) float64 {
		if !n.Valid {
			return 0
		}
		f, _ := n.Float64Value()
		if f.Valid {
			return f.Float64
		}
		return 0
	}

	totals := &dashboardtypes.ClaimsTotals{
		Claims:           totalCountRow.TotalClaims,
		TotalUnits:       totalCountRow.TotalUnits,
		ChargeCount:      totalCountRow.TotalCharges,
		TotalCheckNumber: totalCountRow.TotalCheckNumber,
		Charge:           numericToFloat(totalCountRow.TotalAmount),
		Expected:         numericToFloat(totalCountRow.TotalExpected),
		Payment:          numericToFloat(totalCountRow.TotalPayment),
		Outstanding:      numericToFloat(totalCountRow.OutstandingAmount),
	}

	return &dashboardtypes.ClaimsResponse{
		Data: claims,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCountRow.TotalClaims,

			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
		Totals: totals,
	}, nil
}

func (s *service) GetClaimCharges(ctx context.Context, claimID string, year *int, month *int) ([]dashboardtypes.ClaimCharge, error) {
	s.logger.Info("Service: Getting claim charges", "claimID", claimID, "year", year, "month", month)

	var yearNum *int32
	if year != nil {
		y := int32(*year)
		yearNum = &y
	}

	var monthNum *int32
	if month != nil {
		m := int32(*month)
		monthNum = &m
	}

	chargeRows, err := s.repo.GetClaimCharges(ctx, claimID, yearNum, monthNum)
	if err != nil {
		s.logger.Error("Service: Failed to get claim charges from repository", "error", err)
		return nil, err
	}

	charges := make([]dashboardtypes.ClaimCharge, 0, len(chargeRows))
	var claimDetails *dashboardtypes.ClaimDetails

	for i, chargeRow := range chargeRows {
		// Convert amounts to float64 with 2 decimal places (truncated, not rounded)
		amount, _ := numericToDecimal(chargeRow.Amount).Float64()
		expectedAmount, _ := numericToDecimal(chargeRow.ExpectedAmount).Float64()
		insurancePaidAmount, _ := numericToDecimal(chargeRow.InsurancePaidAmount).Float64()
		// Calculate outstanding amount (expected - paid)
		outstandingAmount := expectedAmount - insurancePaidAmount

		// Populate claim details from first row (all rows have same claim details)
		if i == 0 {
			var claimFromDate, claimToDate, claimEnteredDate *string

			if chargeRow.ClaimFromDate.Valid {
				dateStr := formatDate(chargeRow.ClaimFromDate)
				claimFromDate = &dateStr
			}
			if chargeRow.ClaimToDate.Valid {
				dateStr := formatDate(chargeRow.ClaimToDate)
				claimToDate = &dateStr
			}
			if chargeRow.ClaimEnteredDate.Valid {
				dateStr := formatDate(chargeRow.ClaimEnteredDate)
				claimEnteredDate = &dateStr
			}

			// Type assert payer fields from interface{} to string
			primaryPayerName := interfaceToString(chargeRow.PrimaryPayerName)
			primaryPayerID := interfaceToString(chargeRow.PrimaryPayerID)
			secondaryPayerName := interfaceToString(chargeRow.SecondaryPayerName)
			secondaryPayerID := interfaceToString(chargeRow.SecondaryPayerID)
			tertiaryPayerName := interfaceToString(chargeRow.TertiaryPayerName)
			tertiaryPayerID := interfaceToString(chargeRow.TertiaryPayerID)

			var secondaryPayerNamePtr, secondaryPayerIDPtr *string
			if secondaryPayerName != "" {
				secondaryPayerNamePtr = &secondaryPayerName
			}
			if secondaryPayerID != "" {
				secondaryPayerIDPtr = &secondaryPayerID
			}

			var tertiaryPayerNamePtr, tertiaryPayerIDPtr *string
			if tertiaryPayerName != "" {
				tertiaryPayerNamePtr = &tertiaryPayerName
			}
			if tertiaryPayerID != "" {
				tertiaryPayerIDPtr = &tertiaryPayerID
			}

			var facilityPosCodePtr, chargePosCodePtr *string
			if chargeRow.FacilityPosCode != "" {
				facilityPosCodePtr = &chargeRow.FacilityPosCode
			}
			if chargeRow.ChargePosCode != "" {
				chargePosCodePtr = &chargeRow.ChargePosCode
			}

			claimDetails = &dashboardtypes.ClaimDetails{
				PatientID:          chargeRow.PatientID,
				PatientName:        chargeRow.PatientName,
				ClaimFromDate:      claimFromDate,
				ClaimToDate:        claimToDate,
				ClaimEnteredDate:   claimEnteredDate,
				ClaimType:          chargeRow.ClaimType,
				PrimaryPayerName:   primaryPayerName,
				PrimaryPayerID:     primaryPayerID,
				SecondaryPayerName: secondaryPayerNamePtr,
				SecondaryPayerID:   secondaryPayerIDPtr,
				TertiaryPayerName:  tertiaryPayerNamePtr,
				TertiaryPayerID:    tertiaryPayerIDPtr,
				FacilityPosCode:    facilityPosCodePtr,
				ChargePosCode:      chargePosCodePtr,
			}
		}

		var unitsPtr *string
		if chargeRow.Units != "" {
			unitsPtr = &chargeRow.Units
		}

		var remarkPtr, unpaidReasonPtr, adjustmentReasonsPtr *string
		if chargeRow.Remark != "" {
			remarkPtr = &chargeRow.Remark
		}
		if chargeRow.UnpaidReason != "" {
			unpaidReasonPtr = &chargeRow.UnpaidReason
		}
		// Get adjustment_reasons from chargeRow (field will be available after sqlc regenerate)
		// Using reflection to safely access the field
		chargeRowValue := reflect.ValueOf(chargeRow)
		if chargeRowValue.Kind() == reflect.Struct {
			adjReasonsField := chargeRowValue.FieldByName("AdjustmentReasons")
			if adjReasonsField.IsValid() && adjReasonsField.Kind() == reflect.String {
				adjReasons := adjReasonsField.String()
				if adjReasons != "" {
					adjustmentReasonsPtr = &adjReasons
				}
			}
		}

		var totalCheckNumberPtr *int
		if chargeRow.TotalCheckNumberCount > 0 {
			totalCheckNumberInt := int(chargeRow.TotalCheckNumberCount)
			totalCheckNumberPtr = &totalCheckNumberInt
		}

		charges = append(charges, dashboardtypes.ClaimCharge{
			ClaimID:             chargeRow.ClaimID,
			CptCode:             chargeRow.CptCode,
			ChargeID:            chargeRow.ChargeID,
			Modifiers:           chargeRow.Modifiers,
			Amount:              amount,
			ExpectedAmount:      expectedAmount,
			InsurancePaidAmount: insurancePaidAmount,
			OutstandingAmount:   outstandingAmount,
			RevenueCode:         chargeRow.RevenueCode,
			Units:               unitsPtr,
			Remark:              remarkPtr,
			UnpaidReason:        unpaidReasonPtr,
			AdjustmentReasons:   adjustmentReasonsPtr,
			TotalCheckNumber:    totalCheckNumberPtr,
			ClaimDetails:        claimDetails,
		})
	}

	return charges, nil
}

func (s *service) GetClaimById(ctx context.Context, claimID string) ([]dashboardtypes.Claim, error) {
	s.logger.Info("Service: Getting claim by ID", "claimID", claimID)

	claimRows, err := s.repo.GetClaimById(ctx, claimID)
	if err != nil {
		s.logger.Error("Service: Failed to get claim by ID from repository", "error", err)
		return nil, err
	}

	claims := make([]dashboardtypes.Claim, 0, len(claimRows))
	for _, claimRow := range claimRows {
		// Format dates - claimRow has pgtype.Date fields
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

		// Convert numeric types to pointers (matching Claim struct)
		totalClaimsAmount, _ := numericToDecimal(claimRow.TotalClaims).Float64()
		totalExpectedAmount, _ := numericToDecimal(claimRow.TotalExpected).Float64()
		totalPaymentAmount, _ := numericToDecimal(claimRow.TotalPayment).Float64()
		totalOutstandingAmount, _ := numericToDecimal(claimRow.OutstandingAmount).Float64()
		totalUnitsInt := numericToDecimal(claimRow.TotalUnits).Truncate(0).IntPart()

		// Convert interface{} fields to strings
		patientID := interfaceToString(claimRow.PatientID)
		patientName := interfaceToString(claimRow.PatientName)
		claimType := interfaceToString(claimRow.TypeOfService)
		primaryPayerName := interfaceToString(claimRow.PrimaryPayerName)
		primaryPayerID := interfaceToString(claimRow.PrimaryPayerID)
		secondaryPayerName := interfaceToString(claimRow.SecondaryPayerName)
		secondaryPayerID := interfaceToString(claimRow.SecondaryPayerID)
		tertiaryPayerName := interfaceToString(claimRow.TertiaryPayerName)
		tertiaryPayerID := interfaceToString(claimRow.TertiaryPayerID)

		// Arrays are already []string
		cptCodes := claimRow.CptCodes
		modifiers := claimRow.Modifiers

		// Get individual CPT code (from new field)
		cptCode := claimRow.CptCode

		// Get payment fields
		unpaidReason := claimRow.UnpaidReason
		remark := claimRow.Remark
		// Get adjustment_reasons from claimRow (field will be available after sqlc regenerate)
		adjustmentReasons := ""
		claimRowValue := reflect.ValueOf(claimRow)
		if claimRowValue.Kind() == reflect.Struct {
			adjReasonsField := claimRowValue.FieldByName("AdjustmentReasons")
			if adjReasonsField.IsValid() && adjReasonsField.Kind() == reflect.String {
				adjustmentReasons = adjReasonsField.String()
			}
		}

		var totalCheckNumberPtr *int
		if claimRow.TotalCheckNumberCount > 0 {
			totalCheckNumberInt := int(claimRow.TotalCheckNumberCount)
			totalCheckNumberPtr = &totalCheckNumberInt
		}

		claim := dashboardtypes.Claim{
			ClaimID:           claimRow.ClaimID,
			ChargeID:          claimRow.ChargeID,
			PatientID:         patientID,
			PatientName:       patientName,
			TotalClaims:       &totalClaimsAmount,
			TotalExpected:     &totalExpectedAmount,
			TotalPayment:      &totalPaymentAmount,
			TotalOutstanding:  &totalOutstandingAmount,
			TotalUnits:        &totalUnitsInt,
			ClaimFromDate:     claimFromDate,
			ClaimToDate:       claimToDate,
			ClaimEnteredDate:  claimEnteredDate,
			ClaimType:         claimType,
			PrimaryPayerName:  primaryPayerName,
			PrimaryPayerID:    primaryPayerID,
			ChargeCount:       int(claimRow.ChargeCount),
			TotalCharges:      int(claimRow.TotalCharges),
			CptCodes:          cptCodes,
			Modifiers:         modifiers,
			CptCode:           cptCode,
			UnpaidReason:      unpaidReason,
			AdjustmentReasons: adjustmentReasons,
			Remark:            remark,
			TotalCheckNumber:  totalCheckNumberPtr,
		}

		// Add optional fields
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

		// Units and POS codes (if present) - these are strings in GetClaimByIdRow
		if claimRow.Units != "" {
			claim.Units = &claimRow.Units
		}
		if claimRow.FacilityPosCode != "" {
			claim.FacilityPosCode = &claimRow.FacilityPosCode
		}
		if claimRow.ChargePosCode != "" {
			claim.ChargePosCode = &claimRow.ChargePosCode
		}

		claims = append(claims, claim)
	}

	return claims, nil
}
