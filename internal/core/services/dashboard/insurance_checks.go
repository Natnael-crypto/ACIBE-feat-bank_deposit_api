package dashboard

import (
	"context"
	"fmt"
	"strings"
	"sync"

	dbgen "github.com/aci/backend/gen/db"
	common "github.com/aci/backend/internal/core/domain/common"
	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *service) GetInsuranceChecks(ctx context.Context, year int, month int, search string, limit int, cursor string) (*dashboardtypes.InsuranceChecksResponse, error) {
	s.logger.Info("Service: Getting paginated insurance checks", "year", year, "month", month, "search", search, "limit", limit, "cursor", cursor)

	// Validate inputs
	if limit < 1 {
		limit = 15
	}
	if limit > 100 {
		limit = 100
	}

	// Run count, paginated queries, and total deposit in parallel for better performance
	var checkRows []dbgen.GetInsuranceChecksPaginatedRow
	var totalCount int64
	var totalDepositAmount pgtype.Numeric
	var checksErr, countErr, totalDepositErr error

	var wg sync.WaitGroup

	// Fetch paginated insurance checks
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkRows, checksErr = s.repo.GetInsuranceChecksPaginated(ctx, int32(year), int32(month), search, int32(limit), cursor)
		if checksErr != nil {
			s.logger.Error("Service: Failed to get paginated insurance checks from repository", "error", checksErr)
		}
	}()

	// Fetch total count and total deposit (only on first page, when cursor is empty)
	if cursor == "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			totalCount, countErr = s.repo.GetInsuranceChecksCount(ctx, int32(year), int32(month), search)
			if countErr != nil {
				s.logger.Error("Service: Failed to get insurance checks count from repository", "error", countErr)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			totalDepositAmount, totalDepositErr = s.repo.GetInsuranceChecksTotalDeposit(ctx, int32(year), int32(month), search)
			if totalDepositErr != nil {
				s.logger.Error("Service: Failed to get insurance checks total deposit from repository", "error", totalDepositErr)
			}
		}()
	}

	wg.Wait()

	// Check for errors
	if checksErr != nil {
		return nil, checksErr
	}
	if countErr != nil {
		return nil, countErr
	}
	if totalDepositErr != nil {
		return nil, totalDepositErr
	}

	// Convert database rows to InsuranceCheck structs
	checks := make([]dashboardtypes.InsuranceCheck, 0, len(checkRows))
	var nextCursor string

	for i, checkRow := range checkRows {
		// Convert numeric types to float64
		checkAmount, _ := numericToDecimal(checkRow.CheckAmount).Float64()
		checkAppliedAmount, _ := numericToDecimal(checkRow.CheckAppliedAmount).Float64()
		providerAdjustmentAmount, _ := numericToDecimal(checkRow.ProviderAdjustmentAmount).Float64()
		checkRemainingAmount, _ := numericToDecimal(checkRow.CheckRemainingAmount).Float64()

		// Format dates
		var checkDate, depositDate *string
		if checkRow.CheckDate.Valid {
			dateStr := formatDate(checkRow.CheckDate)
			checkDate = &dateStr
		}
		if checkRow.DepositDate.Valid {
			dateStr := formatDate(checkRow.DepositDate)
			depositDate = &dateStr
		}

		// Generate cursor from last row: "claim_month_number|deposit_date|check_number|id"
		if i == len(checkRows)-1 {
			depositDateStr := ""
			if checkRow.DepositDate.Valid {
				depositDateStr = checkRow.DepositDate.Time.Format("2006-01-02")
			}
			nextCursor = fmt.Sprintf("%d|%s|%s|%s", checkRow.MonthNumber, depositDateStr, checkRow.CheckNumber, checkRow.ID)
		}

		// Trim MonthName string
		monthName := strings.TrimSpace(checkRow.MonthName)

		check := dashboardtypes.InsuranceCheck{
			ID:                       checkRow.ID,
			CheckPayer:               checkRow.CheckPayer,
			CheckDate:                checkDate,
			DepositDate:              depositDate,
			CheckNumber:              checkRow.CheckNumber,
			CheckAmount:              checkAmount,
			CheckAppliedAmount:        checkAppliedAmount,
			ProviderAdjustmentAmount:  providerAdjustmentAmount,
			CheckRemainingAmount:      checkRemainingAmount,
			CheckCommitted:           checkRow.CheckCommitted,
			MonthNumber:              int(checkRow.MonthNumber),
			MonthName:                monthName,
		}

		checks = append(checks, check)
	}

	// Determine if there are more items
	hasMore := len(checkRows) == limit

	// Convert total deposit amount to float64
	var totalDepositAmountPtr *float64
	if cursor == "" {
		totalDepositFloat, _ := numericToDecimal(totalDepositAmount).Float64()
		totalDepositAmountPtr = &totalDepositFloat
	}

	return &dashboardtypes.InsuranceChecksResponse{
		Data:              checks,
		Pagination:        common.PaginationMetadata{
			TotalRecords: totalCount,
			Limit:        limit,
			HasMore:      hasMore,
			NextCursor:   nextCursor,
		},
		TotalDepositAmount: totalDepositAmountPtr,
	}, nil
}

