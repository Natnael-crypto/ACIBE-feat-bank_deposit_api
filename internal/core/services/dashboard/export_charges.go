package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportChargesToExcel exports charges data to Excel
func (s *service) ExportChargesToExcel(ctx context.Context, claimID string, year *int, month *int) ([]byte, string, error) {
	// Get charges
	charges, err := s.GetClaimCharges(ctx, claimID, year, month)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get charges: %w", err)
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			s.logger.Error("Failed to close Excel file", "error", err)
		}
	}()

	sheetName := "Charges"
	// Use default Sheet1 and rename it
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, "", fmt.Errorf("failed to rename sheet: %w", err)
	}

	// Set headers
	headers := []string{
		"Claim ID", "Charge ID", "CPT Code", "Revenue Code",
		"Units", "Amount", "Expected Amount", "Insurance Paid Amount",
		"Modifiers", "Patient ID", "Patient Name", "Claim Type",
		"Service From Date", "Service To Date", "Claim Entered Date",
		"Primary Payer Name", "Primary Payer ID",
		"Secondary Payer Name", "Secondary Payer ID",
		"Tertiary Payer Name", "Tertiary Payer ID",
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
	for i, charge := range charges {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), charge.ClaimID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), charge.ChargeID)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), charge.CptCode)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), charge.RevenueCode)
		// Handle optional Units (*string)
		if charge.Units != nil && *charge.Units != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), *charge.Units)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), charge.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), charge.ExpectedAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), charge.InsurancePaidAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), strings.Join(charge.Modifiers, ", "))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), charge.ClaimDetails.PatientID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), charge.ClaimDetails.PatientName)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), charge.ClaimDetails.ClaimType)
		// Handle optional ClaimFromDate (*string)
		if charge.ClaimDetails.ClaimFromDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), *charge.ClaimDetails.ClaimFromDate)
		}
		// Handle optional ClaimToDate (*string)
		if charge.ClaimDetails.ClaimToDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), *charge.ClaimDetails.ClaimToDate)
		}
		// Handle optional ClaimEnteredDate (*string)
		if charge.ClaimDetails.ClaimEnteredDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), *charge.ClaimDetails.ClaimEnteredDate)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), charge.ClaimDetails.PrimaryPayerName)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), charge.ClaimDetails.PrimaryPayerID)
		// Handle optional SecondaryPayerName (*string)
		if charge.ClaimDetails.SecondaryPayerName != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), *charge.ClaimDetails.SecondaryPayerName)
		}
		// Handle optional SecondaryPayerID (*string)
		if charge.ClaimDetails.SecondaryPayerID != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), *charge.ClaimDetails.SecondaryPayerID)
		}
		// Handle optional TertiaryPayerName (*string)
		if charge.ClaimDetails.TertiaryPayerName != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), *charge.ClaimDetails.TertiaryPayerName)
		}
		// Handle optional TertiaryPayerID (*string)
		if charge.ClaimDetails.TertiaryPayerID != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("U%d", row), *charge.ClaimDetails.TertiaryPayerID)
		}
		// Handle optional FacilityPosCode (*string)
		if charge.ClaimDetails.FacilityPosCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("V%d", row), *charge.ClaimDetails.FacilityPosCode)
		}
		// Handle optional ChargePosCode (*string)
		if charge.ClaimDetails.ChargePosCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("W%d", row), *charge.ClaimDetails.ChargePosCode)
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
	monthLabel := "All"
	if month != nil {
		monthLabel = time.Month(*month).String()
	}
	yearLabel := "All"
	if year != nil {
		yearLabel = fmt.Sprintf("%d", *year)
	}
	filename := fmt.Sprintf("Charges_%s_%s_%s.xlsx", claimID, monthLabel, yearLabel)

	// Write to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buf.Bytes(), filename, nil
}
