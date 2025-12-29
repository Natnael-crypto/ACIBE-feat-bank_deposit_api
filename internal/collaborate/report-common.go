package collaborate

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func (a *api) startReport(ctx context.Context, customerID, reportID, filterID string) (string, error) {
	if a.Username == "" || a.Password == "" {
		return "", fmt.Errorf("username and password must be set for API authentication")
	}

	url := fmt.Sprintf("%s/customer/%s/reports/%s/filter/%s/run", baseURL, customerID, reportID, filterID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	credentials := a.Username + ":" + a.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	req.Header.Set("Authorization", "Basic "+encoded)

	req.Header.Set("Accept", "application/xml")

	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header not set")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Log response status for debugging
	a.Logger.Info("API response received", "status_code", resp.StatusCode, "status", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.Logger.Error("API request failed", "status_code", resp.StatusCode, "response_body", string(body))
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response GenericResponseXML
	if err := xml.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse XML response: %w", err)
	}

	if response.Status != "SUCCESS" {
		return "", fmt.Errorf("report start failed: %s", response.StatusMessage)
	}

	if response.Identifier == "" {
		return "", fmt.Errorf("identifier not found in response")
	}

	return response.Identifier, nil
}

func (a *api) pollReportResults(ctx context.Context, customerID, identifier string) ([]byte, error) {
	time.Sleep(10 * time.Second)
	if a.Username == "" || a.Password == "" {
		return nil, fmt.Errorf("username and password must be set for API authentication")
	}

	url := fmt.Sprintf("%s/customer/%s/reports/results/%s", baseURL, customerID, identifier)

	maxAttempts := 60 // Poll for up to 5 minutes (60 * 5 seconds)
	pollInterval := 10 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		if err != nil {
			return nil, err
		}

		credentials := a.Username + ":" + a.Password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", "Basic "+encoded)

		req.Header.Set("Accept", "application/xml")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var response ReportResponse
		if err := xml.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse XML response: %w", err)
		}

		if response.Data != "" {

			zipData, err := base64.StdEncoding.DecodeString(response.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 data: %w", err)
			}
			return zipData, nil
		}

		if response.Status != "SUCCESS" {

			if strings.Contains(strings.ToLower(response.StatusMessage), "still running") ||
				strings.Contains(strings.ToLower(response.StatusMessage), "try again later") {

				if attempt < maxAttempts-1 {
					a.Logger.Info("Report still running, waiting...", "attempt", attempt+1, "message", response.StatusMessage)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(pollInterval):

					}
					continue
				}
			}

			return nil, fmt.Errorf("report status error: %s", response.StatusMessage)
		}

		if attempt < maxAttempts-1 {
			a.Logger.Info("Report not ready, waiting...", "attempt", attempt+1, "message", response.StatusMessage)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollInterval):

			}
		}
	}

	return nil, fmt.Errorf("report did not complete within timeout period")
}

func (a *api) saveZipFile(reportType string, identifier string, zipData []byte) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	paymentzipsDir := filepath.Join(cwd, fmt.Sprintf("%s_zips", reportType))
	if err := os.MkdirAll(paymentzipsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create paymentzips directory: %w", err)
	}

	filename := fmt.Sprintf("%s_report_%s.zip", reportType, identifier)
	zipPath := filepath.Join(paymentzipsDir, filename)

	if err := os.WriteFile(zipPath, zipData, 0644); err != nil {
		return "", fmt.Errorf("failed to write zip file: %w", err)
	}

	a.Logger.Info("Zip file saved", "path", zipPath)
	return zipPath, nil
}

func (a *api) parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}

	t, err := time.Parse("01/02/2006", dateStr)
	if err != nil {
		a.Logger.Warn("Failed to parse date", "date", dateStr, "error", err)
		return time.Time{}
	}
	return t
}
func (a *api) parseInt(intStr string) int32 {
	decimalint := a.parseMoney(intStr)
	return int32(decimalint.IntPart())
}
func (a *api) parseMoney(moneyStr string) decimal.Decimal {
	// it might have - sign in front of the money which means it is a negative number
	var negative bool
	if strings.HasPrefix(moneyStr, "-") {
		negative = true
		moneyStr = strings.TrimPrefix(moneyStr, "-")
	}
	moneyStr = strings.TrimSpace(moneyStr)
	if moneyStr == "" {
		return decimal.Zero
	}

	moneyStr = strings.ReplaceAll(moneyStr, "$", "")
	moneyStr = strings.ReplaceAll(moneyStr, ",", "")

	if negative {
		moneyStr = "-" + moneyStr
	}

	d, _ := decimal.NewFromString(moneyStr)
	return d
}

func (a *api) parseModifiers(mod1, mod2, mod3, mod4 string) []string {
	modifiers := []string{}
	for _, mod := range []string{mod1, mod2, mod3, mod4} {
		mod = strings.TrimSpace(mod)
		if mod != "" {
			modifiers = append(modifiers, mod)
		}
	}
	return modifiers
}

// for debugging purposes
func (a *api) findCSVFileInZipReader(zipReader *zip.Reader) (*zip.File, error) {
	var csvFiles []string
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		lowerName := strings.ToLower(f.Name)

		if strings.HasSuffix(lowerName, ".csv") {
			csvFiles = append(csvFiles, f.Name)
			return f, nil
		}
	}
	if len(csvFiles) > 0 {
		a.Logger.Warn("CSV file not found", "available_csv_files", csvFiles)
		return nil, fmt.Errorf("CSV file not found in zip. Available CSV files: %v", csvFiles)
	}
	return nil, fmt.Errorf("no CSV files found in zip")
}
