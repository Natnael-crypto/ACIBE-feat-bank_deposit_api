package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportPaymentsToExcel exports payments data to Excel
func (s *service) ExportPaymentsToExcel(ctx context.Context, year *int, month *int, excludePeer bool, peerOnly bool, search *string, claimType string) ([]byte, string, error) {
	// Get all payments (limit 0 means all)
	// Export all data without filters (pass nil for filter parameters)
	paymentsResponse, err := s.GetPayments(ctx, year, month, excludePeer, peerOnly, search, claimType, 1, 0, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get payments: %w", err)
	}

	payments := paymentsResponse.Data

	// Create new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			s.logger.Error("Failed to close Excel file", "error", err)
		}
	}()

	sheetName := "Payments"
	// Use default Sheet1 and rename it
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, "", fmt.Errorf("failed to rename sheet: %w", err)
	}

	// Set headers
	headers := []string{
		"Claim ID", "Patient ID", "Patient Name", "Charge ID",
		"CPT Code", "Revenue Code", "Payer Name", "Payer ID",
		"Payment Check/TRN", "Payment Date", "Posted Date",
		"Charge Date", "BPR Date", "Total Charge", "Total Paid",
		"Expected Amount", "Insurance Adjustment", "Units",
		"Claim Status", "Credit ID", "Remark", "Unpaid Reason",
		"Is Refund", "Is Provider Adjustment", "Is ERA Adjustment", "Modifiers",
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
	for i, payment := range payments {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), payment.ClaimID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), payment.PatientID)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), payment.PatientName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), payment.ChargeID)
		if payment.CptCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), *payment.CptCode)
		}
		if payment.RevenueCode != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), *payment.RevenueCode)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), payment.InsuranceName)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), payment.InsuranceID)
		if payment.Trn != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), *payment.Trn)
		}
		// Handle optional PaymentDate (*string)
		if payment.PaymentDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), *payment.PaymentDate)
		}
		// Handle optional PostedDate (*string)
		if payment.PostedDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), *payment.PostedDate)
		}
		// Handle optional ChargeDate (*string)
		if payment.ChargeDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), *payment.ChargeDate)
		}
		if payment.BprDate != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), *payment.BprDate)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), payment.TotalClaimCharge)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), payment.TotalPaidAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), payment.TotalExpectedAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), payment.InsuranceAdjustment)
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), payment.Units)
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), payment.ClaimStatus)
		// Handle optional CreditID (*string)
		if payment.CreditID != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), *payment.CreditID)
		}
		// Handle optional Remark (*string)
		if payment.Remark != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("U%d", row), *payment.Remark)
		}
		// Handle optional UnpaidReason (*string)
		if payment.UnpaidReason != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("V%d", row), *payment.UnpaidReason)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("W%d", row), payment.IsRefund)
		f.SetCellValue(sheetName, fmt.Sprintf("X%d", row), payment.IsProviderAdjustment)
		f.SetCellValue(sheetName, fmt.Sprintf("Y%d", row), payment.IsEraAdjustment)
		f.SetCellValue(sheetName, fmt.Sprintf("Z%d", row), strings.Join(payment.Modifiers, ", "))
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
	filename := fmt.Sprintf("Payments_%s_%s_%s.xlsx", typeLabel, monthLabel, yearLabel)

	// Write to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buf.Bytes(), filename, nil
}
