package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportClaimsToExcel exports claims data to Excel
func (s *service) ExportClaimsToExcel(ctx context.Context, year *int, month *int, category string, claimType string, excludePeer bool, search *string, peerOnly bool) ([]byte, string, error) {
	// Log input parameters
	s.logger.Info("ExportClaimsToExcel called", "year", year, "month", month, "category", category, "claimType", claimType, "excludePeer", excludePeer, "peerOnly", peerOnly, "search", search)
	
	// Normalize claimType: when peerOnly is true or type is "all", use empty string
	// This ensures the SQL query handles peer filtering correctly
	originalClaimType := claimType
	if peerOnly || claimType == "all" {
		claimType = ""
	}
	s.logger.Info("Normalized claimType", "original", originalClaimType, "normalized", claimType)
	
	// Get all claims (limit 0 means all)
	// Export all data without filters (pass nil for filter parameters)
	claimsResponse, err := s.GetClaims(ctx, year, month, category, claimType, excludePeer, search, peerOnly, 1, 0, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get claims: %w", err)
	}

	claims := claimsResponse.Data

	// Log for debugging
	s.logger.Info("Exporting claims to Excel", "count", len(claims), "totalCount", claimsResponse.Pagination.TotalRecords)

	// Create new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			s.logger.Error("Failed to close Excel file", "error", err)
		}
	}()

	sheetName := "Claims"
	// Use default Sheet1 and rename it
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, "", fmt.Errorf("failed to rename sheet: %w", err)
	}

	// Set headers
	headers := []string{
		"Claim ID", "Charge ID", "Patient ID", "Patient Name",
		"Claim From Date", "Claim To Date", "Claim Entered Date",
		"Claim Type", "Primary Payer Name", "Primary Payer ID",
		"Secondary Payer Name", "Secondary Payer ID",
		"Tertiary Payer Name", "Tertiary Payer ID",
		"Total Claims", "Total Expected", "Total Payment", "Total Outstanding",
		"Total Charges",
		"CPT Codes", "Modifiers", "Units",
		"Facility POS Code", "Charge POS Code",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Set header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	if err == nil {
		f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)
	}

	// Write data
	for i, claim := range claims {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), claim.ClaimID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), claim.ChargeID)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), claim.PatientID)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), claim.PatientName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), claim.ClaimFromDate)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), claim.ClaimToDate)
		// Handle optional ClaimEnteredDate
		if claim.ClaimEnteredDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), *claim.ClaimEnteredDate)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), claim.ClaimType)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), claim.PrimaryPayerName)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), claim.PrimaryPayerID)
		// Handle optional SecondaryPayerName
		if claim.SecondaryPayerName != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), *claim.SecondaryPayerName)
		}
		// Handle optional SecondaryPayerID
		if claim.SecondaryPayerID != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), *claim.SecondaryPayerID)
		}
		// Handle optional TertiaryPayerName
		if claim.TertiaryPayerName != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), *claim.TertiaryPayerName)
		}
		// Handle optional TertiaryPayerID
		if claim.TertiaryPayerID != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), *claim.TertiaryPayerID)
		}
		// Handle optional TotalClaims (*float64)
		if claim.TotalClaims != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), *claim.TotalClaims)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), 0)
		}
		// Handle optional TotalExpected (*float64)
		if claim.TotalExpected != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), *claim.TotalExpected)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), 0)
		}
		// Handle optional TotalPayment (*float64)
		if claim.TotalPayment != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), *claim.TotalPayment)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), 0)
		}
		// Handle optional TotalOutstanding (*float64)
		if claim.TotalOutstanding != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), *claim.TotalOutstanding)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), 0)
		}
		// Total Charges (removed Charge Count column)
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), claim.TotalCharges)
		// Handle optional CptCodes slice
		if len(claim.CptCodes) > 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), strings.Join(claim.CptCodes, ", "))
		}
		// Handle optional Modifiers slice
		if len(claim.Modifiers) > 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("U%d", row), strings.Join(claim.Modifiers, ", "))
		}
		// Handle Units: use Units if available, otherwise use TotalUnits
		unitsValue := ""
		if claim.Units != nil && *claim.Units != "" {
			unitsValue = *claim.Units
		} else if claim.TotalUnits != nil {
			// Convert TotalUnits (int64) to string
			unitsValue = fmt.Sprintf("%d", *claim.TotalUnits)
		}
		if unitsValue != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("V%d", row), unitsValue)
		}
		// Handle optional FacilityPosCode (*string)
		if claim.FacilityPosCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("W%d", row), *claim.FacilityPosCode)
		}
		// Handle optional ChargePosCode (*string)
		if claim.ChargePosCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("X%d", row), *claim.ChargePosCode)
		}
	}

	// Set active sheet (get index of our sheet)
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get sheet index: %w", err)
	}
	f.SetActiveSheet(sheetIndex)

	// Auto-fit columns
	for i := 0; i < len(headers); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Generate filename
	categoryLabel := getCategoryLabel(category)
	typeLabel := claimType
	if peerOnly {
		typeLabel = "Peer"
	}
	if typeLabel == "" {
		typeLabel = "All"
	}
	monthLabel := "All"
	if month != nil {
		monthLabel = time.Month(*month).String()
	}
	yearLabel := "All"
	if year != nil {
		yearLabel = fmt.Sprintf("%d", *year)
	}
	filename := fmt.Sprintf("Claims_%s_%s_%s_%s.xlsx", categoryLabel, typeLabel, monthLabel, yearLabel)

	// Write to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buf.Bytes(), filename, nil
}
