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

	claim "github.com/aci/backend/internal/core/services/claim"
	"github.com/shopspring/decimal"
)

const layout = "20060102"

var allowedModifiers = map[string]struct{}{
	"AH": {},
	"AJ": {},
	"HO": {},
	"AF": {},
	"AG": {},
	"SA": {},
}

func isAllowedModifier(modifier string) bool {
	_, ok := allowedModifiers[strings.ToUpper(strings.TrimSpace(modifier))]
	return ok
}
func (a *api) GetClaimsApi(ctx context.Context, date time.Time) ([]claim.Claim, error) {
	return nil, nil
}
func (a *api) GetClaimsFileDebug(ctx context.Context, date time.Time) ([]claim.Claim, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// fmt.Println("Current working directory:", cwd)

	// Process all zip files in the claimzips folder
	claimzipsDir := filepath.Join(cwd, "claimzips")
	entries, err := os.ReadDir(claimzipsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read claimzips directory: %w", err)
	}

	var claims []claim.Claim

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
			parsedClaims, err := a.parseClaimEdi(text)
			if err != nil {
				a.Logger.Error("failed to parse 837 file", "zip", entry.Name(), "file", f.Name, "error", err)
				continue
			}

			claims = append(claims, parsedClaims)
		}

		r.Close()
	}

	// if len(claims) > 0 {
	// 	fmt.Printf("Processed %d claims from claimzips folder\n", len(claims))
	// 	fmt.Printf("%+v", claims[0])
	// }
	return claims, nil
}

func (a *api) parseClaimEdi(raw string) (claim.Claim, error) {
	if strings.Contains(raw, "Please contact customer") {
		return claim.Claim{}, fmt.Errorf("invalid EDI data: contains error message")
	}

	url := "http://localhost:3000/parse"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(raw))
	if err != nil {
		return claim.Claim{}, err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return claim.Claim{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return claim.Claim{}, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return claim.Claim{}, err
	}
	var parseEdiRes EDIResponse
	if err := json.Unmarshal(body, &parseEdiRes); err != nil {
		return claim.Claim{}, err
	}
	claimparsed := a.mapSingleClaimFromSegments(parseEdiRes.Segments)
	claimparsed.RawText = raw
	return claimparsed, nil
}
func (a *api) mapSingleClaimFromSegments(segments []RawSegment837) claim.Claim {
	var claimres claim.Claim

	state := &claimProcessingState{
		currentLineNumber: "",
		primaryInsurance:  false,
	}

	for _, seg := range segments {
		segName := getClaimString(seg, "name")

		switch segName {
		case "BHT":
			a.handleBHTSegment(&claimres, seg)
		case "HI":
			a.handleHISegment(&claimres, seg)
		case "ST":
			// Reserved for future use - claim type detection
		case "CLM":
			a.handleCLMSegment(&claimres, seg)
		case "DTP":
			a.handleDTPSegment(&claimres, seg)
		case "NM1":
			a.handleClaimNM1Segment(&claimres, state, seg)
		case "LX":
			a.handleLXSegment(state, seg)
		case "SV2":
			a.handleSV2Segment(&claimres, state, seg)
		case "SV1":
			a.handleSV1Segment(&claimres, state, seg)
		case "SBR":
			a.handleSBRSegment(state, seg)
		case "REF":
			a.handleREFSegmentClaim(&claimres, seg)
		}
	}

	return claimres
}

type claimProcessingState struct {
	currentLineNumber string
	primaryInsurance  bool
}

func (a *api) handleREFSegmentClaim(c *claim.Claim, seg RawSegment837) {
	if getClaimString(seg, "1") == "F8" {
		c.OriginalClaimNumber = getClaimString(seg, "2")
	}
}
func (a *api) handleBHTSegment(c *claim.Claim, seg RawSegment837) {
	dateStr := getClaimString(seg, "4")
	c.ClaimSubittionDate = parseDate(dateStr)
}

func (a *api) handleHISegment(c *claim.Claim, seg RawSegment837) {
	if getClaimString(seg, "1") != "BE" {
		return
	}

	if getClaimString(seg, "1-1") == "24" || getClaimString(seg, "2-2") == "24" {
		pos := getClaimString(seg, "1-4")
		if pos == "" {
			pos = getClaimString(seg, "2-4")
		}

		switch pos {
		case "1080.0":
			c.TypeOfService = string(claim.PlaceOfServiceOffSite)
		case "1540.0":
			c.TypeOfService = string(claim.PlaceOfServiceOnsite)
		}
	}
}

func (a *api) handleCLMSegment(c *claim.Claim, seg RawSegment837) {
	c.ClaimId = getClaimString(seg, "1")

	if amountStr := getClaimString(seg, "2"); amountStr != "" {
		c.TotalCharge = a.parseDecimal(amountStr)
	}

	c.ClaimFrequency = getClaimString(seg, "5-2")
	pos := getClaimString(seg, "5")

	if pos == "11" {
		c.TypeOfService = string(claim.PlaceOfServiceOnsite)
	} else if pos != "" {
		c.TypeOfService = string(claim.PlaceOfServiceOffSite)
	}

}

func (a *api) handleDTPSegment(c *claim.Claim, seg RawSegment837) {
	dtType := getClaimString(seg, "1")
	dateStr := getClaimString(seg, "3")

	if dateStr == "" {
		return
	}

	switch dtType {
	case "434":
		dates := strings.Split(dateStr, "-")
		c.ServiceDateFrom = parseDate(dates[0])
		if len(dates) > 1 {
			c.ServiceDateTo = parseDate(dates[1])
		} else {
			c.ServiceDateTo = c.ServiceDateFrom
		}
	case "431":
		c.ClaimSubittionDate = parseDate(dateStr)
	case "472":
		c.ServiceDateFrom = parseDate(dateStr)
	default:
		// fmt.Println(seg)
	}
}

func (a *api) handleClaimNM1Segment(c *claim.Claim, state *claimProcessingState, seg RawSegment837) {
	idType := getClaimString(seg, "1")

	switch idType {
	case "IL":
		c.PatientId = getClaimString(seg, "9")
		firstName := getClaimString(seg, "4")
		lastName := getClaimString(seg, "3")
		if firstName != "" || lastName != "" {
			c.PatientName = strings.TrimSpace(firstName + " " + lastName)
		}
	case "PR":
		if state.primaryInsurance {
			c.PrimaryInsuranceName = getClaimString(seg, "3")
			c.PrimaryInsuranceId = getClaimString(seg, "9")
		} else {
			c.SecondaryInsuranceName = getClaimString(seg, "3")
			c.SecondaryInsuranceId = getClaimString(seg, "9")
		}
	}
}

func (a *api) handleLXSegment(state *claimProcessingState, seg RawSegment837) {
	state.currentLineNumber = getClaimString(seg, "1")
}

func (a *api) handleSV2Segment(c *claim.Claim, state *claimProcessingState, seg RawSegment837) {
	sl := claim.ServiceLine{
		LineNumber: state.currentLineNumber,
		CPTCode:    getClaimString(seg, "2-1"),
		Modifiers:  extractClaimModifiers(seg, "2"),
		Code:       getClaimString(seg, "4"),
	}

	if unitsStr := getClaimString(seg, "5"); unitsStr != "" {
		sl.Units, _ = strconv.Atoi(unitsStr)
	}

	if amountStr := getClaimString(seg, "3"); amountStr != "" {
		sl.Amount = a.parseDecimal(amountStr)
	}

	// Update TypeOfService if Code is DA
	if sl.Code == "DA" {
		if tos := getClaimString(seg, "1"); tos != "" {
			c.TypeOfService = tos
		}
	}

	c.ServiceLines = append(c.ServiceLines, sl)
	state.currentLineNumber = ""
}

func (a *api) handleSV1Segment(c *claim.Claim, state *claimProcessingState, seg RawSegment837) {
	sl := claim.ServiceLine{
		LineNumber: state.currentLineNumber,
		CPTCode:    getClaimString(seg, "1-1"),
		Modifiers:  extractClaimModifiers(seg, "1"),
		Code:       getClaimString(seg, "3"),
	}

	if unitsStr := getClaimString(seg, "4"); unitsStr != "" {
		sl.Units, _ = strconv.Atoi(unitsStr)
	}

	if amountStr := getClaimString(seg, "2"); amountStr != "" {
		sl.Amount = a.parseDecimal(amountStr)
	}

	c.ServiceLines = append(c.ServiceLines, sl)
	state.currentLineNumber = ""
}

func (a *api) handleSBRSegment(state *claimProcessingState, seg RawSegment837) {
	typestr := getClaimString(seg, "1")
	state.primaryInsurance = (typestr == "P")
}

func extractClaimModifiers(seg RawSegment837, prefix string) []string {
	var modifiers []string
	for i := 2; i <= 5; i++ {
		key := fmt.Sprintf("%s-%d", prefix, i)
		if modifierStr := getClaimString(seg, key); modifierStr != "" && isAllowedModifier(modifierStr) {
			modifiers = append(modifiers, modifierStr)
		}
	}
	return modifiers
}

func getClaimString(segment RawSegment837, key string) string {
	if val, ok := segment[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("20060102", s)
	return t
}

func (a *api) parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		a.Logger.Error("error parsing decimal", "value", s, "error", err)
		return decimal.Zero
	}

	return d
}
