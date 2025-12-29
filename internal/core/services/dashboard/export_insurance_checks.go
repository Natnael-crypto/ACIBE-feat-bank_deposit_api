package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportInsuranceChecksToExcel exports insurance checks data to Excel
func (s *service) ExportInsuranceChecksToExcel(ctx context.Context, year int, month int, search string) ([]byte, string, error) {
	// Get all insurance checks (limit 0 means all)
	checksResponse, err := s.GetInsuranceChecks(ctx, year, month, search, 0, "")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get insurance checks: %w", err)
	}

	checks := checksResponse.Data

	// Create new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			s.logger.Error("Failed to close Excel file", "error", err)
		}
	}()

	sheetName := "Insurance Checks"
	// Use default Sheet1 and rename it
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, "", fmt.Errorf("failed to rename sheet: %w", err)
	}

	// Set headers
	headers := []string{
		"ID", "Check Number", "Check Payer", "Check Date",
		"Deposit Date", "Check Amount", "Check Applied Amount",
		"Provider Adjustment Amount", "Check Remaining Amount",
		"Check Committed", "Month",
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
	for i, check := range checks {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), check.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), check.CheckNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), check.CheckPayer)
		// Handle optional CheckDate (*string)
		if check.CheckDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), *check.CheckDate)
		}
		// Handle optional DepositDate (*string)
		if check.DepositDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), *check.DepositDate)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), check.CheckAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), check.CheckAppliedAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), check.ProviderAdjustmentAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), check.CheckRemainingAmount)
		committed := "No"
		if check.CheckCommitted {
			committed = "Yes"
		}
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), committed)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), check.MonthName)
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
	monthLabel := time.Month(month).String()
	filename := fmt.Sprintf("Insurance_Checks_%s_%d.xlsx", monthLabel, year)

	// Write to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buf.Bytes(), filename, nil
}
