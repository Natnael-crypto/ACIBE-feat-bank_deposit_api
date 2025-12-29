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

func (s *service) GetPaymentChecksByClaimAndCharge(ctx context.Context, claimID string, chargeID string, year *int, month *int, page int, limit int) (*dashboardtypes.PaymentChecksResponse, error) {
	s.logger.Info("Service: Getting payment checks by claim and charge", "claimID", claimID, "chargeID", chargeID, "year", year, "month", month, "page", page, "limit", limit)

	// Validate inputs
	if claimID == "" {
		return nil, fmt.Errorf("claim_id is required")
	}

	if page < 1 {
		page = 1
	}

	effectiveLimit := limit
	if limit < 1 {
		effectiveLimit = 25
	} else if limit > 100 {
		effectiveLimit = 100
	}

	offset := (page - 1) * effectiveLimit

	yearInt := int32(0)
	if year != nil {
		yearInt = int32(*year)
	}

	monthInt := int32(0)
	if month != nil {
		monthInt = int32(*month)
	}

	var checkRows []dbgen.GetPaymentChecksByClaimAndChargeRow
	var totalCount int64
	var checksErr, countErr error

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		checkRows, checksErr = s.repo.GetPaymentChecksByClaimAndCharge(ctx, dbgen.GetPaymentChecksByClaimAndChargeParams{
			Column1: claimID,
			Column2: chargeID,
			Column3: yearInt,
			Column4: monthInt,
			Column5: int32(effectiveLimit),
			Column6: int32(offset),
		})
		if checksErr != nil {
			s.logger.Error("Service: Failed to get payment checks from repository", "error", checksErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		totalCount, countErr = s.repo.GetPaymentChecksByClaimAndChargeCount(ctx, dbgen.GetPaymentChecksByClaimAndChargeCountParams{
			Column1: claimID,
			Column2: chargeID,
			Column3: yearInt,
			Column4: monthInt,
		})
		if countErr != nil {
			s.logger.Error("Service: Failed to get payment checks count from repository", "error", countErr)
		}
	}()

	wg.Wait()

	if checksErr != nil {
		return nil, fmt.Errorf("failed to get payment checks: %w", checksErr)
	}
	if countErr != nil {
		return nil, fmt.Errorf("failed to get payment checks count: %w", countErr)
	}

	// Convert database rows to PaymentCheck structs
	checks := make([]dashboardtypes.PaymentCheck, 0, len(checkRows))
	for _, checkRow := range checkRows {
		amount, _ := numericToDecimal(checkRow.Amount).Float64()

		var depositDate *string
		if checkRow.DepositDate.Valid {
			dateStr := formatDate(checkRow.DepositDate)
			depositDate = &dateStr
		}

		// Get adjustment_reasons from checkRow (field will be available after sqlc regenerate)
		adjustmentReasons := ""
		checkRowValue := reflect.ValueOf(checkRow)
		if checkRowValue.Kind() == reflect.Struct {
			adjReasonsField := checkRowValue.FieldByName("AdjustmentReasons")
			if adjReasonsField.IsValid() && adjReasonsField.Kind() == reflect.String {
				adjustmentReasons = adjReasonsField.String()
			}
		}

		check := dashboardtypes.PaymentCheck{
			ClaimID:          checkRow.ClaimID,
			ChargeID:         checkRow.ChargeID,
			CheckNumber:      checkRow.CheckNumber,
			Amount:           amount,
			Remark:           checkRow.Remark,
			UnpaidReason:     checkRow.UnpaidReason,
			AdjustmentReasons: adjustmentReasons,
			DepositDate:      depositDate,
		}

		checks = append(checks, check)
	}

	// Calculate total pages
	var totalPages int
	if effectiveLimit == 0 {
		totalPages = 1
	} else {
		totalPages = int((totalCount + int64(effectiveLimit) - 1) / int64(effectiveLimit))
	}

	return &dashboardtypes.PaymentChecksResponse{
		Data: checks,
		Pagination: common.PaginationMetadata{
			TotalRecords: totalCount,
			Page:         page,
			Limit:        effectiveLimit,
			TotalPages:   totalPages,
		},
	}, nil
}
