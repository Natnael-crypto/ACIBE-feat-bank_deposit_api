package dashboard

import (
	"context"
	"strings"

	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/internal/types/dashboard"
)

func (r *repositoryImpl) GetAvailableYears(ctx context.Context) ([]int32, error) {
	return r.store.Queries().GetAvailableYears(ctx)
}

func (r *repositoryImpl) GetDashboardData(ctx context.Context, year int32) ([]dashboard.DashboardDataRow, error) {
	// Materialized view query is already optimized - no need for work_mem adjustment
	// The materialized view provides sub-100ms response times without additional configuration
	rows, err := r.store.Queries().GetDashboardData(ctx, year)
	if err != nil {
		return nil, err
	}

	result := make([]dashboard.DashboardDataRow, 0, len(rows))
	for _, row := range rows {
		// Type assert MonthName from interface{} to string
		monthName := ""
		if row.MonthName != nil {
			if str, ok := row.MonthName.(string); ok {
				monthName = strings.TrimSpace(str)
			}
		}
		
		result = append(result, dashboard.DashboardDataRow{
			MonthNumber:                 int(row.MonthNumber),
			MonthName:                   monthName,
			ClaimCategory:               row.ClaimCategory,
			UniqueClaims:                row.UniqueClaims,
			TotalUnits:                  repository.NumericToDecimal(row.TotalUnits),
			TotalAmount:                 repository.NumericToDecimal(row.TotalAmount),
			UniqueClaimsWithPayments:    row.UniqueClaimsWithPayments,
			TotalUnitsWithPayments:      repository.NumericToDecimal(row.TotalUnitsWithPayments),
			TotalExpectedAmount:         repository.NumericToDecimal(row.TotalExpectedAmount),
			TotalPaymetsAmount:          repository.NumericToDecimal(row.TotalPaymentsAmount),
			TotalExpectedAmountWithPayments: repository.NumericToDecimal(row.TotalExpectedAmountWithPayments),
			UniqueClaimsWithOutstanding: row.UniqueClaimsWithOutstanding,
			TotalUnitsWithOutstanding:   repository.NumericToDecimal(row.TotalUnitsWithOutstanding),
			TotalOutstandingAmount:      repository.NumericToDecimal(row.TotalOutstandingAmount),
		})
	}

	return result, nil
}

func (r *repositoryImpl) GetYearlyDashboardData(ctx context.Context) ([]dashboard.DashboardDataRow, error) {
	// The materialized view dashboard_yearly_data provides pre-aggregated yearly data
	rows, err := r.store.Queries().GetYearlyDashboardData(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dashboard.DashboardDataRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, dashboard.DashboardDataRow{
			Year:                        int(row.Year),
			ClaimCategory:               row.ClaimCategory,
			UniqueClaims:                row.UniqueClaims,
			TotalUnits:                  repository.NumericToDecimal(row.TotalUnits),
			TotalAmount:                 repository.NumericToDecimal(row.TotalAmount),
			UniqueClaimsWithPayments:    row.UniqueClaimsWithPayments,
			TotalUnitsWithPayments:      repository.NumericToDecimal(row.TotalUnitsWithPayments),
			TotalExpectedAmount:         repository.NumericToDecimal(row.TotalExpectedAmount),
			TotalPaymetsAmount:          repository.NumericToDecimal(row.TotalPaymentsAmount),
			TotalExpectedAmountWithPayments: repository.NumericToDecimal(row.TotalExpectedAmountWithPayments),
			UniqueClaimsWithOutstanding: row.UniqueClaimsWithOutstanding,
			TotalUnitsWithOutstanding:   repository.NumericToDecimal(row.TotalUnitsWithOutstanding),
			TotalOutstandingAmount:      repository.NumericToDecimal(row.TotalOutstandingAmount),
		})
	}

	return result, nil
}

