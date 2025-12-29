package collaborate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	ratesdomain "github.com/aci/backend/internal/core/domain/rates"
	claim "github.com/aci/backend/internal/core/services/claim"
	"github.com/shopspring/decimal"
)

// https://webapi.collaboratemd.com/v1/customer/10028881/reports/10079179/filter/10141951/ru
// codes to add expected value tofixed to debug
const (
	couldntfindRate  = 0.001
	unknownPayerRate = 0.002
)

// the following are for week
// based on claim entered
const (
	CustomerID = "10028881"
	ReportID   = "10079179"
	FilterID   = "10142098"
)

// year
// ReportID   = "10079179"
//
//	FilterID   = "10141951"
//
// PayerRate represents a rate entry in payer_rates.json
type PayerRate struct {
	ID           []int64     `json:"id"`
	PrimaryPayer []string    `json:"primary_payer"`
	Rates        []RateEntry `json:"rates"`
}

// RateEntry represents a single rate period with date range
type RateEntry struct {
	DateRange string `json:"date_range"`
	Msws      any    `json:"msws"`  // Can be float64, string ("OON"), or null
	Rehab     any    `json:"rehab"` // Can be float64, string ("OON"), or null
}

// ParsedRateEntry represents a parsed rate entry with parsed dates
type ParsedRateEntry struct {
	StartDate time.Time
	EndDate   *time.Time // nil means open-ended
	Msws      *decimal.Decimal
	Rehab     *decimal.Decimal
}

// PayerRatesLookup stores parsed payer rates for quick lookup
type PayerRatesLookup struct {
	mu    sync.RWMutex
	rates map[int64][]ParsedRateEntry // payer ID -> list of rate periods
}

var payerRatesLookup *PayerRatesLookup
var payerRatesOnce sync.Once

// getPayerRatesLookup returns the singleton payer rates lookup, loading it if necessary
func getPayerRatesLookup() (*PayerRatesLookup, error) {
	var loadErr error
	payerRatesOnce.Do(func() {
		lookup := &PayerRatesLookup{
			rates: make(map[int64][]ParsedRateEntry),
		}
		loadErr = lookup.loadPayerRates()
		payerRatesLookup = lookup
	})
	return payerRatesLookup, loadErr
}

// loadPayerRates loads and parses payer_rates.json
func (p *PayerRatesLookup) loadPayerRates() error {
	data, err := os.ReadFile("payer_rates.json")
	if err != nil {
		return fmt.Errorf("failed to read payer_rates.json: %w", err)
	}

	var payerRates []PayerRate
	if err := json.Unmarshal(data, &payerRates); err != nil {
		return fmt.Errorf("failed to parse payer_rates.json: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, payerRate := range payerRates {
		parsedEntries := make([]ParsedRateEntry, 0, len(payerRate.Rates))
		for _, rateEntry := range payerRate.Rates {
			parsed, err := parseRateEntry(rateEntry)
			if err != nil {
				return fmt.Errorf("failed to parse rate entry for payer %v: %w", payerRate.ID, err)
			}
			parsedEntries = append(parsedEntries, parsed)
		}
		// Store the parsed entries for each ID in the array
		for _, id := range payerRate.ID {
			p.rates[id] = parsedEntries
		}
	}

	return nil
}

// parseRateEntry parses a RateEntry into a ParsedRateEntry
func parseRateEntry(entry RateEntry) (ParsedRateEntry, error) {
	parsed := ParsedRateEntry{}

	// Parse date range - can be "start-end" or just "start"
	dateRange := strings.TrimSpace(entry.DateRange)
	parts := strings.Split(dateRange, "-")
	if len(parts) == 0 {
		return parsed, fmt.Errorf("invalid date range: %s", dateRange)
	}

	startStr := strings.TrimSpace(parts[0])
	startDate, err := parsePayerRateDate(startStr)
	if err != nil {
		return parsed, fmt.Errorf("failed to parse start date %q: %w", startStr, err)
	}
	parsed.StartDate = startDate

	if len(parts) > 1 {
		endStr := strings.TrimSpace(parts[1])
		endDate, err := parsePayerRateDate(endStr)
		if err != nil {
			return parsed, fmt.Errorf("failed to parse end date %q: %w", endStr, err)
		}
		parsed.EndDate = &endDate
	}

	// Parse msws rate
	if entry.Msws != nil {
		if mswsRate, err := parseRateValue(entry.Msws); err == nil {
			parsed.Msws = mswsRate
		}
	}

	// Parse rehab rate
	if entry.Rehab != nil {
		if rehabRate, err := parseRateValue(entry.Rehab); err == nil {
			parsed.Rehab = rehabRate
		}
	}

	return parsed, nil
}

// parsePayerRateDate parses dates in formats like "1/1/2019", "4/1/2024", etc.
func parsePayerRateDate(dateStr string) (time.Time, error) {
	// Try with zero-padding first (MM/DD/YYYY)
	if t, err := time.Parse("01/02/2006", dateStr); err == nil {
		return t, nil
	}
	// Try without zero-padding (M/D/YYYY)
	if t, err := time.Parse("1/2/2006", dateStr); err == nil {
		return t, nil
	}
	// Try with zero-padding and 2-digit year (MM/DD/YY)
	if t, err := time.Parse("01/02/06", dateStr); err == nil {
		return t, nil
	}
	// Try without zero-padding and 2-digit year (M/D/YY)
	return time.Parse("1/2/06", dateStr)
}

// parseRateValue converts a rate value (float64, string, or null) to decimal.Decimal
// Returns nil if value is "OON", null, or invalid
func parseRateValue(value interface{}) (*decimal.Decimal, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case float64:
		rate := decimal.NewFromFloat(v)
		return &rate, nil
	case string:
		if strings.ToUpper(v) == "OON" {
			return nil, nil
		}
		// Try to parse as float
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rate := decimal.NewFromFloat(f)
			return &rate, nil
		}
		return nil, fmt.Errorf("invalid rate value: %s", v)
	default:
		return nil, fmt.Errorf("unexpected rate value type: %T", value)
	}
}

// getPayerRate looks up the rate for a given payer ID, date, and rate type (msws or rehab)
func (p *PayerRatesLookup) getPayerRate(payerID string, claimDate time.Time, rateType string) (*decimal.Decimal, error) {
	payerIDInt, err := strconv.ParseInt(payerID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid payer ID: %s", payerID)
	}

	p.mu.RLock()
	rateEntries, exists := p.rates[payerIDInt]
	p.mu.RUnlock()

	if !exists {
		// send 7777

		rate := decimal.NewFromFloat(unknownPayerRate)
		return &rate, nil
	}

	// Find the matching rate period - iterate in reverse to get the most recent match
	for i := len(rateEntries) - 1; i >= 0; i-- {
		entry := rateEntries[i]

		// Check if claim date is before start date
		if claimDate.Before(entry.StartDate) {
			continue
		}

		// Check if claim date is after end date (if end date exists)
		if entry.EndDate != nil && claimDate.After(*entry.EndDate) {
			continue
		}

		// Found matching period, return the appropriate rate (most recent one)
		if rateType == "msws" && entry.Msws != nil {
			return entry.Msws, nil
		}
		if rateType == "rehab" && entry.Rehab != nil {
			return entry.Rehab, nil
		}
	}

	return nil, fmt.Errorf("no rate found for payer %s, date %s, type %s", payerID, claimDate.Format("2006-01-02"), rateType)
}

// getRateTypeFromRevenueCode determines if a revenue code maps to msws or rehab
// 0126, 126, 0136, 136 → msws
// 0128, 128, 0138, 138 → rehab
func getRateTypeFromRevenueCode(revenueCode string) string {
	normalized := strings.TrimSpace(revenueCode)
	switch normalized {
	case "0126", "126", "0136", "136":
		return "msws"
	case "0128", "128", "0138", "138":
		return "rehab"
	default:
		// Default to rehab for other codes (can be adjusted based on requirements)
		return "rehab"
	}
}

func (a *api) GetClaimsZipFile(ctx context.Context, zipPath string) ([]claim.NewClaim, error) {
	if zipPath == "" {
		return nil, fmt.Errorf("zip file path is required")
	}
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}
	claims, err := a.ExtractAndParseClaimCSV(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and parse claim CSV: %w", err)
	}
	return claims, nil
}

func (a *api) GetClaims(ctx context.Context) ([]claim.NewClaim, error) {
	identifier, err := a.startReport(ctx, CustomerID, ReportID, FilterID)
	if err != nil {
		return nil, fmt.Errorf("failed to start report: %w", err)
	}

	a.Logger.Info("Report started", "identifier", identifier)

	zipData, err := a.pollReportResults(ctx, CustomerID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get report results: %w", err)
	}

	return a.ExtractAndParseClaimCSV(zipData)
}

func (a *api) parseClaimCSVRecords(records [][]string) ([]claim.NewClaim, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var claims []claim.NewClaim
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 33 {
			continue
		}
		if row[0] == "" {
			continue
		}

		modifierStr := strings.TrimSpace(row[10])
		modifiers := []string{}
		if modifierStr != "" {
			parts := strings.Fields(modifierStr)
			modifiers = append(modifiers, parts...)
		}

		newClaim := claim.NewClaim{
			ClaimId:          strings.TrimSpace(row[0]),  // Claim ID
			ClaimFromDate:    a.parseDate(row[1]),        // Claim From Date
			ClaimToDate:      a.parseDate(row[2]),        // Claim To Date
			ClaimType:        strings.TrimSpace(row[3]),  // Claim Type
			ClaimEnteredDate: a.parseDate(row[6]),        // Claim Date Entered
			TotalClaim:       a.parseMoney(row[7]),       // Claim Total Amount
			CptCode:          strings.TrimSpace(row[8]),  // Charge CPT Code
			ChargePosCode:    strings.TrimSpace(row[9]),  // Charge POS Code
			Modifiers:        modifiers,                  // Charge Modifier List
			Units:            strings.TrimSpace(row[15]), // Charge Units
			Amount:           a.parseMoney(row[17]),      // Charge Amount
			RevenueCode:      strings.TrimSpace(row[18]), // Revenue Code
			FacilityPosCode:  strings.TrimSpace(row[19]), // Facility POS Code
			ChargeId:         strings.TrimSpace(row[32]), // Charge ID (Column AG)
			ValueCode1:       strings.TrimSpace(row[33]), // Value Code 1
			ValueAmount1:     a.parseInt(row[34]),        // Value Amount 1
			ClaimStatus:      strings.TrimSpace(row[35]), // Claim Status
		}

		if strings.TrimSpace(row[20]) == "Yes" {
			newClaim.PrimaryPayer = &claim.Payer{
				Id:   strings.TrimSpace(row[21]),
				Name: strings.TrimSpace(row[22]),
				Type: strings.TrimSpace(row[23]),
			}
		}

		if strings.TrimSpace(row[24]) == "Yes" {
			newClaim.SecondaryPayer = &claim.Payer{
				Id:   strings.TrimSpace(row[25]),
				Name: strings.TrimSpace(row[26]),
				Type: strings.TrimSpace(row[27]),
			}
		}

		if strings.TrimSpace(row[28]) == "Yes" {
			newClaim.TertiaryPayer = &claim.Payer{
				Id:   strings.TrimSpace(row[29]),
				Name: strings.TrimSpace(row[30]),
				Type: strings.TrimSpace(row[31]),
			}
		}

		claims = append(claims, newClaim)
	}
	// only have 5000 claims in the file
	// if len(claims) > 500 {
	// 	claims = claims[:500]
	// }
	claims = changeoutpatientToInpatient(claims)
	for i := range claims {
		// if claims[i].ClaimId != "281717835" {
		// 	continue
		// }

		var err error
		claims[i], err = a.calculateExpectedAmount(claims[i])
		if err != nil {
			a.Logger.Error("failed to calculate expected amount", "claim", claims[i].ClaimId, "error", err)
			continue
		}

		// fmt.Println("claim", claims[i].ClaimId, "amount", claims[i].Amount, "expected amount", claims[i].ExpectedAmount, claims[i].Units)
		// if expectedAmount != claim.Amount {
		// 	os.Exit(0)
		// }
	}
	// print 5 claims amount and expected amount with revenue code not in 0126, 0128, 0136, 0138, 126, 128, 136, 138 and is
	// for i := 0; i < 500; i++ {
	// 	if claims[i].RevenueCode != "0126" && claims[i].RevenueCode != "0128" && claims[i].RevenueCode != "0136" && claims[i].RevenueCode != "0138" && claims[i].RevenueCode != "126" && claims[i].RevenueCode != "128" && claims[i].RevenueCode != "136" && claims[i].RevenueCode != "138" {
	// 		fmt.Println("claim", claims[i].ClaimId, "amount", claims[i].Amount, "expected amount", claims[i].ExpectedAmount)
	// 	}
	// }

	// os.Exit(0)
	return claims, nil
}
func changeoutpatientToInpatient(claims []claim.NewClaim) []claim.NewClaim {
	for i := range claims {
		switch claims[i].CptCode {
		case "H2019":
			claims[i].RevenueCode = "0128"
		case "H0017":
			claims[i].RevenueCode = "0128"
		case "H0019":
			claims[i].RevenueCode = "0128"
		case "99235":
			claims[i].RevenueCode = "0128"
		case "H0010":
			claims[i].RevenueCode = "0126"
		case "H2036":
			claims[i].RevenueCode = "0128"
		}
	}
	return claims
}

// parseRatePeriodDate parses dates in multiple formats: ISO (YYYY-MM-DD) or US format (M/D/YYYY, MM/DD/YYYY, M/D/YY, or MM/DD/YY)
func parseRatePeriodDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try ISO format first (YYYY-MM-DD)
	if matched, _ := regexp.MatchString(`^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$`, dateStr); matched {
		return time.Parse("2006-01-02", dateStr)
	}

	// Try US format with 4-digit year - handle both padded and non-padded
	if matched, _ := regexp.MatchString(`/[0-9]{4}$`, dateStr); matched {
		// Try with zero-padding first (MM/DD/YYYY)
		if t, err := time.Parse("01/02/2006", dateStr); err == nil {
			return t, nil
		}
		// Try without zero-padding (M/D/YYYY)
		return time.Parse("1/2/2006", dateStr)
	}

	// Try US format with 2-digit year - handle both padded and non-padded
	// Try with zero-padding first (MM/DD/YY)
	if t, err := time.Parse("01/02/06", dateStr); err == nil {
		return t, nil
	}
	// Try without zero-padding (M/D/YY)
	return time.Parse("1/2/06", dateStr)
}

func (a *api) ExtractAndParseClaimCSV(zipData []byte) ([]claim.NewClaim, error) {
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

	claims, err := a.parseClaimCSVRecords(records)
	if err != nil {
		return nil, err
	}

	a.Logger.Info("Parsed claim CSV from memory", "total_claims", len(claims))
	return claims, nil
}

// func (a *api) ExtractAndParseClaimCSV(zipPath string) ([]claim.NewClaim, error) {
// 	r, err := zip.OpenReader(zipPath)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to open zip file: %w", err)
// 	}
// 	defer r.Close()

// 	csvFile, err := a.findCSVFileInZipReader(&r.Reader, "custom report")
// 	if err != nil {
// 		return nil, err
// 	}

// 	rc, err := csvFile.Open()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to open CSV file: %w", err)
// 	}
// 	defer rc.Close()

// 	reader := csv.NewReader(rc)
// 	records, err := reader.ReadAll()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read CSV: %w", err)
// 	}

// 	claims, err := a.parseClaimCSVRecords(records)
// 	if err != nil {
// 		return nil, err
// 	}

// 	a.Logger.Info("Parsed claim CSV", "total_claims", len(claims))
// 	return claims, nil
// }

func (a *api) calculateExpectedAmount(claim claim.NewClaim) (claim.NewClaim, error) {

	if claim.RevenueCode == "0126" || claim.RevenueCode == "0128" || claim.RevenueCode == "0136" || claim.RevenueCode == "0138" || claim.RevenueCode == "126" || claim.RevenueCode == "128" || claim.RevenueCode == "136" || claim.RevenueCode == "138" {
		return a.inpatientRate(claim)

	}
	// if claim.CptCode == "99408" {
	// 	return decimal.Zero, nil
	// }
	expectedAmount, err := a.outPatientRate(claim)
	if err != nil {
		return claim, fmt.Errorf("failed to calculate expected amount: %w", err)
	}
	claim.ExpectedAmount = expectedAmount
	return claim, nil
}

func (a *api) inpatientRate(claim claim.NewClaim) (claim.NewClaim, error) {
	var primary, secondary, tertiary decimal.Decimal
	var err error

	// Load rates only for missing payers
	if claim.PrimaryPayer != nil && claim.PrimaryPayer.Id != "" {
		primary, err = a.getPatientRate("primary", claim)
		if err != nil {
			a.Logger.Error("failed to get primary payer rate", "claim", claim.ClaimId, "error", err)
		}
	}

	if claim.SecondaryPayer != nil && claim.SecondaryPayer.Id != "" {
		secondary, err = a.getPatientRate("secondary", claim)
		if err != nil {
			a.Logger.Error("failed to get secondary payer rate", "claim", claim.ClaimId, "error", err)
		}
	}

	if claim.TertiaryPayer != nil && claim.TertiaryPayer.Id != "" {
		tertiary, err = a.getPatientRate("tertiary", claim)
		if err != nil {
			a.Logger.Error("failed to get tertiary payer rate", "claim", claim.ClaimId, "error", err)
		}
	}

	// highest allowed amount across all payers
	highest := primary
	if secondary.GreaterThan(highest) {
		highest = secondary
	}
	if tertiary.GreaterThan(highest) {
		highest = tertiary
	}

	// Waterfall payment
	remaining := highest

	// Primary pays first
	payPrimary := decimal.Min(primary, remaining)
	if claim.PrimaryPayer != nil {
		claim.PrimaryPayer.ExpectedAmount = payPrimary
	}
	remaining = remaining.Sub(payPrimary)

	// Secondary pays next
	paySecondary := decimal.Min(secondary, remaining)
	if claim.SecondaryPayer != nil {
		claim.SecondaryPayer.ExpectedAmount = paySecondary
	}
	remaining = remaining.Sub(paySecondary)

	// Tertiary pays last
	payTertiary := decimal.Min(tertiary, remaining)
	if claim.TertiaryPayer != nil {
		claim.TertiaryPayer.ExpectedAmount = payTertiary
	}
	claim.ExpectedAmount = highest
	return claim, nil
}

func (a *api) getPatientRate(payer string, claim claim.NewClaim) (decimal.Decimal, error) {
	var id string
	var name string
	switch payer {
	case "primary":
		name = claim.PrimaryPayer.Name
		id = claim.PrimaryPayer.Id
	case "secondary":
		name = claim.SecondaryPayer.Name
		id = claim.SecondaryPayer.Id
	case "tertiary":
		name = claim.TertiaryPayer.Name
		id = claim.TertiaryPayer.Id
	}
	if id == "" {
		return decimal.Zero, fmt.Errorf("no payer id for claim %s", claim.ClaimId)
	}
	lookup, err := getPayerRatesLookup()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to load payer rates: %w", err)
	}

	rateType := getRateTypeFromRevenueCode(claim.RevenueCode)
	rate, err := lookup.getPayerRate(id, claim.ClaimFromDate, rateType)
	if err != nil {
		return decimal.NewFromFloat(couldntfindRate), fmt.Errorf("failed to get payer rate: %w name %s id %s", err, name, id)
	}

	if rate == nil {
		return decimal.NewFromFloat(couldntfindRate), nil
	}

	// Multiply by units
	units, err := strconv.Atoi(claim.Units)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert units to int: %w", err)
	}
	//
	return rate.Mul(decimal.NewFromInt(int64(units))), nil
}

func (a *api) getOutPatientRateFromCacheOrDB(cptcode, modifiers string) (ratesdomain.OutPatientRate, error) {
	cacheKey := fmt.Sprintf("%s-%s", cptcode, modifiers)
	rate, err := a.Cache.Get(cacheKey)
	if err == nil {
		return rate, nil
	}

	rate, err = a.Store.GetOutPatientRate(context.Background(), cptcode, modifiers)
	if err != nil {
		return ratesdomain.OutPatientRate{}, fmt.Errorf("failed to get out patient rate: %w cpt code %s modifiers %s", err, cptcode, modifiers)
	}
	a.Cache.Set(cacheKey, rate)
	return rate, nil
}
func (a *api) outPatientRate(claim claim.NewClaim) (decimal.Decimal, error) {
	// Filter modifiers to only allowed ones
	if claim.CptCode == "INT" || claim.RevenueCode == "INT" {
		return claim.Amount, nil
	}
	var allowedModifiers string
	for _, m := range claim.Modifiers {

		if isAllowedModifier(m) && (claim.CptCode == "90853" || claim.CptCode == "90849") {
			allowedModifiers = m
		}
	}
	/// impment sometye of caching here
	// cacheKey := fmt.Sprintf("%s-%s-%s", claim.CptCode, allowedModifiers)
	rate, err := a.getOutPatientRateFromCacheOrDB(claim.CptCode, allowedModifiers)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get out patient rate: %w cpt code %s modifiers %s", err, claim.CptCode, allowedModifiers)
	}

	offsiteRate := decimal.Zero
	onsiteRate := decimal.Zero
	for _, ratePeriod := range rate.RatePeriods {
		startDate, err := parseRatePeriodDate(ratePeriod.StartDate)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to parse start date %q: %w", ratePeriod.StartDate, err)
		}

		if ratePeriod.EndDate == "" {
			if !claim.ClaimFromDate.Before(startDate) {
				onsiteRate = decimal.NewFromFloat(ratePeriod.OnSiteRate)
				offsiteRate = decimal.NewFromFloat(ratePeriod.OffSiteRate)
				break
			}
			continue
		}

		endDate, err := parseRatePeriodDate(ratePeriod.EndDate)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to parse end date %q: %w", ratePeriod.EndDate, err)
		}

		// Check if claim date is within the period (start_date <= claim_date <= end_date)
		if !claim.ClaimFromDate.Before(startDate) && !claim.ClaimFromDate.After(endDate) {
			offsiteRate = decimal.NewFromFloat(ratePeriod.OffSiteRate)
			onsiteRate = decimal.NewFromFloat(ratePeriod.OnSiteRate)
			break
		}
	}

	if offsiteRate.IsZero() && onsiteRate.IsZero() {
		return decimal.Zero, nil
	}

	if claim.ValueAmount1 == 1080 {
		// fmt.Println("1080")
		units, err := strconv.Atoi(claim.Units)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to convert units to int: %w", err)
		}
		return offsiteRate.Mul(decimal.NewFromInt(int64(units))), nil
	}

	units, err := strconv.Atoi(claim.Units)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert units to int: %w", err)
	}
	return onsiteRate.Mul(decimal.NewFromInt(int64(units))), nil

}

var count int = 0
