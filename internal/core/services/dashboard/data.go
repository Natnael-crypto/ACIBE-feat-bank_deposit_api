package dashboard

import (
	"context"
	"strings"

	dashboardtypes "github.com/aci/backend/internal/types/dashboard"
)

func (s *service) GetDashboardData(ctx context.Context, year int) (*dashboardtypes.DashboardData, error) {
	s.logger.Info("Service: Getting dashboard data", "year", year)
	
	// Fetch dashboard data from repository (new query based on claim_from_date)
	rows, err := s.repo.GetDashboardData(ctx, int32(year))
	if err != nil {
		s.logger.Error("Service: Failed to get dashboard data from repository", "year", year, "error", err)
		return nil, err
	}
	s.logger.Info("Service: Retrieved dashboard data from repository", "year", year, "rowCount", len(rows))

	// Initialize month data structure with all 12 months
	data := make(dashboardtypes.DashboardData)
	for _, monthName := range dashboardtypes.MonthNames {
		data[monthName] = dashboardtypes.MonthData{
			TotalClaims: dashboardtypes.CategoryData{
				IN:   dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
				OUT:  dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
				PEER: dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
			},
			TotalPayments: dashboardtypes.CategoryData{
				IN:   dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
				OUT:  dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
				PEER: dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
			},
			Outstanding: dashboardtypes.CategoryData{
				IN:   dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
				OUT:  dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
				PEER: dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
			},
			TotalExpected: dashboardtypes.CategoryData{
				IN:   dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
				OUT:  dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
				PEER: dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
			},
		}
	}

	// Pre-compute month name mapping for faster lookups
	monthNameMap := make(map[int]string, 12)
	for i, name := range dashboardtypes.MonthNames {
		monthNameMap[i+1] = name
	}

	// Process rows and populate totals by category (IN/OUT/PEER)
	for _, row := range rows {
		// Get month name from month number
		var monthName string
		if row.MonthNumber > 0 && row.MonthNumber <= 12 {
			monthName = monthNameMap[row.MonthNumber]
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
		monthData, exists := data[monthName]
		if !exists {
			continue
		}

		// Convert amounts to appropriate types
		totalUnitsInt := row.TotalUnits.Truncate(0).IntPart()
		totalExpectedAmountFloat, _ := row.TotalExpectedAmount.Float64()
		totalExpectedAmountWithPaymentsFloat, _ := row.TotalExpectedAmountWithPayments.Float64()
		totalPaymentsFloat, _ := row.TotalPaymetsAmount.Float64()
		totalOutstandingAmountFloat, _ := row.TotalOutstandingAmount.Float64()
		
		// TotalClaims expects int64 (dollar amount as integer)
		totalAmountInt := row.TotalAmount.Truncate(0).IntPart()
		
		// Get units for claims with payments and outstanding
		totalUnitsWithPaymentsInt := row.TotalUnitsWithPayments.Truncate(0).IntPart()
		totalUnitsWithOutstandingInt := row.TotalUnitsWithOutstanding.Truncate(0).IntPart()

		// Determine which category to populate based on ClaimCategory
		var categoryClaims *dashboardtypes.TypeData
		var categoryPayments *dashboardtypes.TypeData
		var categoryExpected *dashboardtypes.TypeData
		var categoryOutstanding *dashboardtypes.TypeData

		switch row.ClaimCategory {
		case "IN":
			categoryClaims = &monthData.TotalClaims.IN
			categoryPayments = &monthData.TotalPayments.IN
			categoryExpected = &monthData.TotalExpected.IN
			categoryOutstanding = &monthData.Outstanding.IN
		case "OUT":
			categoryClaims = &monthData.TotalClaims.OUT
			categoryPayments = &monthData.TotalPayments.OUT
			categoryExpected = &monthData.TotalExpected.OUT
			categoryOutstanding = &monthData.Outstanding.OUT
		case "PEER":
			categoryClaims = &monthData.TotalClaims.PEER
			categoryPayments = &monthData.TotalPayments.PEER
			categoryExpected = &monthData.TotalExpected.PEER
			categoryOutstanding = &monthData.Outstanding.PEER
		default:
			// Skip unknown categories
			continue
		}

		// Populate TotalClaims: unique_claims, total_amount (as int64), total_units
		categoryClaims.Value = row.UniqueClaims
		categoryClaims.TotalClaims = totalAmountInt
		categoryClaims.TotalUnits = totalUnitsInt
		
		// Populate TotalPayments: unique_claims_with_payments, total_payments, total_units_with_payments
		categoryPayments.Value = row.UniqueClaimsWithPayments
		categoryPayments.TotalPayments = totalPaymentsFloat
		categoryPayments.TotalExpectedWithPayments = &totalExpectedAmountWithPaymentsFloat
		categoryPayments.TotalUnits = totalUnitsWithPaymentsInt
		
		// Populate TotalExpected: unique_claims, total_expected_amount, total_units
		categoryExpected.Value = row.UniqueClaims
		categoryExpected.TotalExpected = totalExpectedAmountFloat
		categoryExpected.TotalUnits = totalUnitsInt
		
		// Populate Outstanding: unique_claims_with_outstanding, total_outstanding_amount, total_units_with_outstanding
		categoryOutstanding.Value = row.UniqueClaimsWithOutstanding
		categoryOutstanding.TotalOutstanding = &totalOutstandingAmountFloat
		categoryOutstanding.TotalUnits = totalUnitsWithOutstandingInt

		data[monthName] = monthData
	}

	return &data, nil
}


func (s *service) GetYearlyDashboardData(ctx context.Context) (map[int]dashboardtypes.MonthData, error) {
	s.logger.Info("Service: Getting yearly dashboard data")
	
	rows, err := s.repo.GetYearlyDashboardData(ctx)
	if err != nil {
		s.logger.Error("Service: Failed to get yearly dashboard data from repository", "error", err)
		return nil, err
	}

	// Group by year
	data := make(map[int]dashboardtypes.MonthData)
	
	for _, row := range rows {
		year := row.Year
		
		// Initialize year data if it doesn't exist
		if _, exists := data[year]; !exists {
			data[year] = dashboardtypes.MonthData{
				TotalClaims: dashboardtypes.CategoryData{
					IN:   dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
					OUT:  dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
					PEER: dashboardtypes.TypeData{Value: 0, TotalClaims: 0, TotalUnits: 0},
				},
				TotalPayments: dashboardtypes.CategoryData{
					IN:   dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
					OUT:  dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
					PEER: dashboardtypes.TypeData{Value: 0, TotalPayments: 0.0, TotalUnits: 0},
				},
				Outstanding: dashboardtypes.CategoryData{
					IN:   dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
					OUT:  dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
					PEER: dashboardtypes.TypeData{Value: 0, TotalOutstanding: floatPtr(0.0), TotalUnits: 0},
				},
				TotalExpected: dashboardtypes.CategoryData{
					IN:   dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
					OUT:  dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
					PEER: dashboardtypes.TypeData{Value: 0, TotalExpected: 0.0, TotalUnits: 0},
				},
			}
		}

		yearData := data[year]

		// Convert amounts
		totalUnitsInt := row.TotalUnits.Truncate(0).IntPart()
		totalExpectedAmountFloat, _ := row.TotalExpectedAmount.Float64()
		totalExpectedAmountWithPaymentsFloat, _ := row.TotalExpectedAmountWithPayments.Float64()
		totalPaymentsFloat, _ := row.TotalPaymetsAmount.Float64()
		totalOutstandingAmountFloat, _ := row.TotalOutstandingAmount.Float64()
		totalAmountInt := row.TotalAmount.Truncate(0).IntPart()
		totalUnitsWithPaymentsInt := row.TotalUnitsWithPayments.Truncate(0).IntPart()
		totalUnitsWithOutstandingInt := row.TotalUnitsWithOutstanding.Truncate(0).IntPart()

		var categoryClaims *dashboardtypes.TypeData
		var categoryPayments *dashboardtypes.TypeData
		var categoryExpected *dashboardtypes.TypeData
		var categoryOutstanding *dashboardtypes.TypeData

		switch row.ClaimCategory {
		case "IN":
			categoryClaims = &yearData.TotalClaims.IN
			categoryPayments = &yearData.TotalPayments.IN
			categoryExpected = &yearData.TotalExpected.IN
			categoryOutstanding = &yearData.Outstanding.IN
		case "OUT":
			categoryClaims = &yearData.TotalClaims.OUT
			categoryPayments = &yearData.TotalPayments.OUT
			categoryExpected = &yearData.TotalExpected.OUT
			categoryOutstanding = &yearData.Outstanding.OUT
		case "PEER":
			categoryClaims = &yearData.TotalClaims.PEER
			categoryPayments = &yearData.TotalPayments.PEER
			categoryExpected = &yearData.TotalExpected.PEER
			categoryOutstanding = &yearData.Outstanding.PEER
		default:
			continue
		}

		categoryClaims.Value = row.UniqueClaims
		categoryClaims.TotalClaims = totalAmountInt
		categoryClaims.TotalUnits = totalUnitsInt
		
		categoryPayments.Value = row.UniqueClaimsWithPayments
		categoryPayments.TotalPayments = totalPaymentsFloat
		categoryPayments.TotalExpectedWithPayments = &totalExpectedAmountWithPaymentsFloat
		categoryPayments.TotalUnits = totalUnitsWithPaymentsInt
		
		categoryExpected.Value = row.UniqueClaims
		categoryExpected.TotalExpected = totalExpectedAmountFloat
		categoryExpected.TotalUnits = totalUnitsInt
		
		categoryOutstanding.Value = row.UniqueClaimsWithOutstanding
		categoryOutstanding.TotalOutstanding = &totalOutstandingAmountFloat
		categoryOutstanding.TotalUnits = totalUnitsWithOutstandingInt

		data[year] = yearData
	}

	return data, nil
}


func (s *service) GetAvailableYears(ctx context.Context) ([]int, error) {
	s.logger.Info("Service: Getting available years")
	
	years, err := s.repo.GetAvailableYears(ctx)
	if err != nil {
		s.logger.Error("Service: Failed to get available years from repository", "error", err)
		return nil, err
	}
	
	// Convert int32 to int
	result := make([]int, len(years))
	for i, year := range years {
		result[i] = int(year)
	}
	
	s.logger.Info("Service: Retrieved available years", "count", len(result))
	return result, nil
}

