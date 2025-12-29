package collaborate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aci/backend/internal/core/services/payment"
)

func (a *api) GetPaymentsApi(ctx context.Context, date time.Time) ([]payment.Payment, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Search for xof.txt in paymentzips directory (including subdirectories)
	paymentzipsDir := filepath.Join(cwd, "paymentzips")
	var xofFilePath string

	err = filepath.Walk(paymentzipsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "R51003287.1aef1491-abe6-4221-9942-d1610ac4d2fa.251113.txt" {
			xofFilePath = path
			return filepath.SkipAll // Stop walking once found
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search for xof.txt: %w", err)
	}

	if xofFilePath == "" {
		return nil, fmt.Errorf("xof.txt not found in paymentzips directory")
	}

	// Open and read the file
	file, err := os.Open(xofFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open xof.txt: %w", err)
	}
	defer file.Close()

	text, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read xof.txt: %w", err)
	}

	parsedPayment, err := a.parsePaymentEdi(string(text))
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment EDI: %w", err)
	}

	return parsedPayment, nil
}
func (a *api) GetPaymentsFileDebug(ctx context.Context, date time.Time) ([]payment.Payment, error) {

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Process all zip files in the claimzips folder
	claimzipsDir := filepath.Join(cwd, "paymentzips")
	entries, err := os.ReadDir(claimzipsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read claimzips directory: %w", err)
	}

	var payments []payment.Payment

	// Process each file in the claimzips directory
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		zipFilePath := filepath.Join(claimzipsDir, entry.Name())
		r, err := zip.OpenReader(zipFilePath)
		if err != nil {
			a.Logger.Error("failed to open zip file", "file", zipFilePath, "error", err)
			continue
		}

		// Process each file inside the zip
		for _, f := range r.File {
			rc, err := f.Open()
			if err != nil {
				a.Logger.Error("failed to open file in zip", "zip", entry.Name(), "file", f.Name, "error", err)
				continue
			}

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			rc.Close()
			if err != nil {
				a.Logger.Error("failed to read file content", "zip", entry.Name(), "file", f.Name, "error", err)
				continue
			}

			text := buf.String()
			parsedPayment, err := a.parsePaymentEdi(text)
			if err != nil {
				a.Logger.Error("failed to parse 837 file", "zip", entry.Name(), "file", f.Name, "error", err)
				continue
			}

			payments = append(payments, parsedPayment...)
		}

		r.Close()
	}

	if len(payments) > 0 {

	}
	return payments, nil
}

func (a *api) parsePaymentEdi(raw string) ([]payment.Payment, error) {
	if strings.Contains(raw, "Please contact customer") {
		return nil, fmt.Errorf("invalid EDI data: contains error message")
	}

	url := "http://localhost:3000/parse"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parseEdiRes EDIResponse
	if err := json.Unmarshal(body, &parseEdiRes); err != nil {
		return nil, err
	}
	paymentparsed := a.mapPaymentsFromSegments(parseEdiRes.Segments, raw)

	return paymentparsed, nil
}
func (a *api) mapPaymentsFromSegments(segments []RawSegment837, raw string) []payment.Payment {
	var payments []payment.Payment
	var currentPayment payment.Payment
	currentPayment.RawText = raw

	state := &paymentProcessingState{
		insuranceId: "",
		// trn:               "",
		currentLineNumber: "",
		lineCounter:       0,
	}

	for _, seg := range segments {
		name := getString(seg, "name")

		switch name {
		case "BPR":
			a.handleBPRSegment(&currentPayment, seg)
		case "REF":
			a.handleREFSegment(state, seg)
		case "TRN":
			a.handleTRNSegment(state, seg)
		case "N1":
			a.handleN1Segment(&currentPayment, seg)
		case "NM1":
			a.handleNM1Segment(&currentPayment, seg)
		case "DTM":
			a.handleDTMSegment(&currentPayment, seg)
		case "CLP":
			if currentPayment.ClaimId != "" {
				payments = append(payments, currentPayment)
				currentPayment = payment.Payment{}
			}
			a.handleCLPSegment(&currentPayment, state, seg)
		case "SVC":
			a.handleSVCSegment(&currentPayment, state, seg)
		case "CAS":
			a.handleCASSegment(&currentPayment, state, seg)
		}
	}

	if currentPayment.ClaimId != "" {
		payments = append(payments, currentPayment)
	}

	return payments
}

// paymentProcessingState holds state during segment processing
type paymentProcessingState struct {
	insuranceId string
	// trn               string
	currentLineNumber string
	lineCounter       int
}

// handleBPRSegment processes BPR (Financial Information) segment
func (a *api) handleBPRSegment(p *payment.Payment, seg RawSegment837) {
	p.BPRDate = parseDate(getString(seg, "16"))
}

// handleREFSegment processes REF (Reference Identification) segment
func (a *api) handleREFSegment(state *paymentProcessingState, seg RawSegment837) {
	if getString(seg, "1") == "2U" {
		state.insuranceId = getString(seg, "2")
	}
}

// handleTRNSegment processes TRN (Trace) segment
func (a *api) handleTRNSegment(state *paymentProcessingState, seg RawSegment837) {
	// state.trn = getString(seg, "2")
}

// handleN1Segment processes N1 (Party Identification) segment
func (a *api) handleN1Segment(p *payment.Payment, seg RawSegment837) {
	if getString(seg, "1") == "PR" {
		p.InsuranceName = getString(seg, "2")
		p.InsuranceId = getString(seg, "4")
	}
}

// handleNM1Segment processes NM1 (Individual or Organizational Name) segment
func (a *api) handleNM1Segment(p *payment.Payment, seg RawSegment837) {
	if getString(seg, "1") == "QC" {
		p.PatientId = getString(seg, "9")
	}
}

// handleDTMSegment processes DTM (Date/Time Reference) segment
func (a *api) handleDTMSegment(p *payment.Payment, seg RawSegment837) {
	qualifier := getString(seg, "1")
	dateValue := parseDate(getString(seg, "2"))

	switch qualifier {
	case "232":
		p.ChargeDate = dateValue
	case "472":
		p.PostedDate = dateValue
	case "050":
		p.PaymentDate = dateValue
	}
}

// handleCLPSegment processes CLP (Claim Level Data) segment
func (a *api) handleCLPSegment(p *payment.Payment, state *paymentProcessingState, seg RawSegment837) {
	p.ClaimId = getString(seg, "1")
	p.Trn = getString(seg, "7")
	p.InsuranceId = state.insuranceId
	p.TotalClaimCharge = a.parseDecimal(getString(seg, "3"))
	p.TotalPaidAmount = a.parseDecimal(getString(seg, "4"))
	p.ClaimStatus = getString(seg, "2")
	p.PatientId = ""
	p.PostedDate = time.Time{}
	p.ServiceLines = []payment.ServiceLine{}
	state.currentLineNumber = ""
	state.lineCounter = 0
}

// handleSVCSegment processes SVC (Service Information) segment
func (a *api) handleSVCSegment(p *payment.Payment, state *paymentProcessingState, seg RawSegment837) {
	state.lineCounter++

	sl := payment.ServiceLine{
		LineNumber: strconv.Itoa(state.lineCounter),
		CptCode:    getString(seg, "1-1"),
		Amount:     a.parseDecimal(getString(seg, "2")),
		PaidAmount: a.parseDecimal(getString(seg, "3")),
		Modifiers:  extractModifiers(seg),
		Units:      extractUnits(seg),
	}

	state.currentLineNumber = sl.LineNumber
	p.ServiceLines = append(p.ServiceLines, sl)
}

// extractModifiers extracts modifiers from SVC segment
func extractModifiers(seg RawSegment837) []string {
	var modifiers []string
	for _, position := range []string{"1-2", "1-3", "1-4", "1-5"} {
		if modifierStr := getString(seg, position); modifierStr != "" && isAllowedModifier(modifierStr) {
			modifiers = append(modifiers, modifierStr)
		}
	}
	return modifiers
}

func extractUnits(seg RawSegment837) int {
	if unitsStr := getString(seg, "5"); unitsStr != "" {
		if units, err := strconv.Atoi(unitsStr); err == nil {
			return units
		}
	}

	if unitsStr := getString(seg, "7"); unitsStr != "" {
		if units, err := strconv.Atoi(unitsStr); err == nil {
			return units
		}
		os.Exit(1)
	}

	return 0
}

func (a *api) handleCASSegment(p *payment.Payment, state *paymentProcessingState, seg RawSegment837) {
	if state.currentLineNumber == "" {
		adjustments := a.extractAdjustments(seg)

		p.ClaimLevelAdjustment = append(p.ClaimLevelAdjustment, adjustments...)

		return
	}

	for i := range p.ServiceLines {
		if p.ServiceLines[i].LineNumber == state.currentLineNumber {
			adjustments := a.extractAdjustments(seg)
			p.ServiceLines[i].AdjustmentCode = append(p.ServiceLines[i].AdjustmentCode, adjustments...)
			break
		}
	}
}

func (a *api) extractAdjustments(seg RawSegment837) []payment.AdjustmentCode {
	var adjustments []payment.AdjustmentCode
	groupCode := getString(seg, "1")
	segLen := len(seg)

	// Each adjustment is 3 elements: Reason, Amount, Quantity (optional) didn't see qanity
	for idx := 2; idx < segLen; idx += 3 {
		reason := getString(seg, strconv.Itoa(idx))
		if reason == "" {
			continue
		}

		amount := a.parseDecimal(getString(seg, strconv.Itoa(idx+1)))
		quantity := ""
		if idx+2 < segLen {
			quantity = getString(seg, strconv.Itoa(idx+2))
		}

		adjustments = append(adjustments, payment.AdjustmentCode{
			GroupCode:  groupCode,
			ReasonCode: reason,
			Amount:     amount,
			Quantity:   quantity,
		})
	}

	return adjustments
}

func getString(seg RawSegment837, key string) string {
	if val, ok := seg[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
