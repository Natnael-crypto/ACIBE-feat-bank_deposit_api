package collaborate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/aci/backend/internal/core/services/insurance"
)

// these are weekly
// based on check date
const (
	insuranceCustomerID = "10028881"
	insuranceReportID   = "10079605"
	insuranceFilterID   = "10142096"
)

func (a *api) GetInsuranceChecksZip(ctx context.Context, zipPath string) ([]insurance.InsuranceCheck, error) {
	if zipPath == "" {
		return nil, fmt.Errorf("zip file path is required")
	}
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}
	checks, err := a.ExtractAndParseInsuranceCSV(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and parse insurance CSV: %w", err)
	}
	return checks, nil
}

func (a *api) GetInsuranceChecks(ctx context.Context) ([]insurance.InsuranceCheck, error) {

	identifier, err := a.startReport(ctx, insuranceCustomerID, insuranceReportID, insuranceFilterID)
	if err != nil {
		return nil, fmt.Errorf("failed to start report: %w", err)
	}

	a.Logger.Info("Report started", "identifier", identifier)

	zipData, err := a.pollReportResults(ctx, insuranceCustomerID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get report results: %w", err)
	}
	checks, err := a.ExtractAndParseInsuranceCSV(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and parse insurance CSV: %w", err)
	}
	return checks, nil
}

func (a *api) parseInsuranceCSVRecords(records [][]string) ([]insurance.InsuranceCheck, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var checks []insurance.InsuranceCheck
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 10 {
			continue
		}
		if row[0] == "" {
			continue
		}

		checkCommitted := false
		if strings.TrimSpace(strings.ToLower(row[9])) == "yes" {
			checkCommitted = true
		}

		check := insurance.InsuranceCheck{
			Id:                       strings.TrimSpace(row[0]), // ID
			CheckPayer:               strings.TrimSpace(row[1]), // Check Payer
			CheckDate:                a.parseDate(row[2]),       // Check Date
			DepositDate:              a.parseDate(row[3]),       // Deposit Date
			CheckNumber:              strings.TrimSpace(row[4]), // Check Number
			CheckAmount:              a.parseMoney(row[5]),      // Check Amount
			CheckAppliedAmount:       a.parseMoney(row[6]),      // Check Applied Amount
			ProviderAdjustmentAmount: a.parseMoney(row[7]),      // Provider Adjustment Amount
			CheckRemainingAmount:     a.parseMoney(row[8]),      // Check Remaining Amount
			CheckCommitted:           checkCommitted,            // Check Committed
		}

		checks = append(checks, check)
	}

	return checks, nil
}

func (a *api) ExtractAndParseInsuranceCSV(zipData []byte) ([]insurance.InsuranceCheck, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	csvFile, err := a.findCSVFileInZipReader(zipReader)
	if err != nil {
		return nil, err
	}

	rc, err := csvFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	checks, err := a.parseInsuranceCSVRecords(records)
	if err != nil {
		return nil, err
	}

	a.Logger.Info("Parsed insurance CSV from memory", "total_checks", len(checks))
	return checks, nil
}
