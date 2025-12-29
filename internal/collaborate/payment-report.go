package collaborate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/aci/backend/internal/core/services/payment"
)

const (
	baseURL = "https://webapi.collaboratemd.com/v1"
)

// based on payment entered
const (
	customerID = "10028881"
	reportID   = "10079213"
	filterID   = "10141946"
)

// yearly
// reportID   = "10079213"
//
//	filterID   = "10141963"
type GenericResponseXML struct {
	XMLName       xml.Name `xml:"http://www.collaboratemd.com/api/v1/ GenericResponseXML"`
	Status        string   `xml:"Status"`
	StatusMessage string   `xml:"StatusMessage"`
	Identifier    string   `xml:"Identifier"`
}

type ReportResponse struct {
	XMLName       xml.Name `xml:"http://www.collaboratemd.com/api/v1/ ReportResponse"`
	Status        string   `xml:"Status"`
	StatusMessage string   `xml:"StatusMessage"`
	Data          string   `xml:"Data"`
}

func (a *api) GetPaymentsZip(ctx context.Context, zipPath string) ([]payment.NewPayment, error) {
	if zipPath == "" {
		return nil, fmt.Errorf("zip file path is required")
	}
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}
	payments, err := a.ExtractAndParsePaymentCSV(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and parse payment CSV: %w", err)
	}

	return payments, nil
}
func (a *api) GetPayments(ctx context.Context) ([]payment.NewPayment, error) {
	identifier, err := a.startReport(ctx, customerID, reportID, filterID)
	if err != nil {
		return nil, fmt.Errorf("failed to start report: %w", err)
	}

	a.Logger.Info("Report started", "identifier", identifier)

	zipData, err := a.pollReportResults(ctx, customerID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get report results: %w", err)
	}

	payments, err := a.ExtractAndParsePaymentCSV(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and parse payment CSV: %w", err)
	}

	return payments, nil
}

func (a *api) parsePaymentCSVRecords(records [][]string) ([]payment.NewPayment, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var payments []payment.NewPayment
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 31 {
			continue
		}
		if row[1] == "" {
			continue
		}

		payment := payment.NewPayment{
			PaymentReceivedDate:  a.parseDate(row[0]),
			DepositDate:          a.parseDate(row[1]),
			PaymentEnteredDate:   a.parseDate(row[2]),
			ChargeFromDate:       a.parseDate(row[3]),
			ClaimId:              strings.TrimSpace(row[4]),
			CreditId:             strings.TrimSpace(row[5]),
			ChargeId:             strings.TrimSpace(row[6]),
			CptCode:              strings.TrimSpace(row[7]),
			Modifiers:            a.parseModifiers(row[8], row[9], row[10], row[11]),
			PatientId:            strings.TrimSpace(row[12]),
			PatientName:          strings.TrimSpace(row[13]),
			IsRefund:             strings.TrimSpace(row[15]) == "Yes",
			IsProviderAdjustment: strings.TrimSpace(row[16]) == "Yes",
			IsEraAdjustment:      strings.TrimSpace(row[17]) == "Yes",
			Remark:               strings.TrimSpace(row[20]),
			UnpaidReason:         strings.TrimSpace(row[21]),
			PaymentCheck:         strings.TrimSpace(row[22]),
			InsurancePaidA:       a.parseMoney(row[25]),
			ChargeAmount:         a.parseMoney(row[26]),
			InsuranceAdjustment:  a.parseMoney(row[27]),
			PayerId:              strings.TrimSpace(row[28]),
			PaymentName:          strings.TrimSpace(row[29]),
			RevenueCode:          strings.TrimSpace(row[30]),
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

func (a *api) ExtractAndParsePaymentCSV(zipData []byte) ([]payment.NewPayment, error) {
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

	payments, err := a.parsePaymentCSVRecords(records)
	if err != nil {
		return nil, err
	}

	a.Logger.Info("Parsed payment CSV from memory", "total_payments", len(payments))
	return payments, nil
}
