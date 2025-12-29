package dashboard

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// sendExcelResponse sends an Excel file as HTTP response
func (h *DashboardHandler) sendExcelResponse(c *gin.Context, excelData []byte, filename string) {
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelData)
}

// parseYearMonthParams parses optional year and month query parameters
func parseYearMonthParams(c *gin.Context) (*int, *int) {
	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear > 1900 && parsedYear < 2100 {
			year = &parsedYear
		}
	}

	var month *int
	if monthStr := c.Query("month"); monthStr != "" {
		if parsedMonth, err := strconv.Atoi(monthStr); err == nil && parsedMonth >= 0 && parsedMonth <= 12 {
			month = &parsedMonth
		}
	}

	return year, month
}

// ExportClaimsToExcel exports claims to Excel
func (h *DashboardHandler) ExportClaimsToExcel(c *gin.Context) {
	// Parse query parameters
	year, month := parseYearMonthParams(c)

	category := c.DefaultQuery("category", "")
	claimType := c.DefaultQuery("type", "")
	excludePeer := c.Query("excludePeer") == "true"
	peerOnly := c.Query("peerOnly") == "true"
	var search *string
	if searchStr := c.Query("search"); searchStr != "" {
		search = &searchStr
	}

	// Get Excel file
	excelData, filename, err := h.service.ExportClaimsToExcel(c.Request.Context(), year, month, category, claimType, excludePeer, search, peerOnly)
	if err != nil {
		h.logger.Error("Failed to export claims to Excel", "error", err)
		response.SendInternalError(c, "Failed to export claims to Excel")
		return
	}

	h.sendExcelResponse(c, excelData, filename)
}

// ExportPaymentsToExcel exports payments to Excel
func (h *DashboardHandler) ExportPaymentsToExcel(c *gin.Context) {
	// Parse query parameters
	year, month := parseYearMonthParams(c)

	excludePeer := c.Query("excludePeer") == "true"
	peerOnly := c.Query("peerOnly") == "true"
	claimType := c.DefaultQuery("type", "")
	var search *string
	if searchStr := c.Query("search"); searchStr != "" {
		search = &searchStr
	}

	// Get Excel file
	excelData, filename, err := h.service.ExportPaymentsToExcel(c.Request.Context(), year, month, excludePeer, peerOnly, search, claimType)
	if err != nil {
		h.logger.Error("Failed to export payments to Excel", "error", err)
		response.SendInternalError(c, "Failed to export payments to Excel")
		return
	}

	h.sendExcelResponse(c, excelData, filename)
}

// ExportPaymentsByClaimAndChargeToExcel exports payments for a specific claim and charge to Excel
func (h *DashboardHandler) ExportPaymentsByClaimAndChargeToExcel(c *gin.Context) {
	claimID := c.Query("claimId")
	chargeID := c.Query("chargeId")
	if claimID == "" || chargeID == "" {
		response.SendValidationError(c, "claimId and chargeId are required")
		return
	}

	year, month := parseYearMonthParams(c)

	// Get Excel file
	excelData, filename, err := h.service.ExportPaymentsByClaimAndChargeToExcel(c.Request.Context(), claimID, chargeID, year, month)
	if err != nil {
		h.logger.Error("Failed to export payments to Excel", "error", err)
		response.SendInternalError(c, "Failed to export payments to Excel")
		return
	}

	h.sendExcelResponse(c, excelData, filename)
}

// ExportInsuranceChecksToExcel exports insurance checks to Excel
func (h *DashboardHandler) ExportInsuranceChecksToExcel(c *gin.Context) {
	// Parse query parameters
	yearStr := c.Query("year")
	if yearStr == "" {
		response.SendValidationError(c, "year is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		response.SendValidationError(c, "invalid year")
		return
	}

	monthStr := c.Query("month")
	if monthStr == "" {
		response.SendValidationError(c, "month is required")
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 0 || month > 12 {
		response.SendValidationError(c, "invalid month")
		return
	}

	search := c.DefaultQuery("search", "")

	// Get Excel file
	excelData, filename, err := h.service.ExportInsuranceChecksToExcel(c.Request.Context(), year, month, search)
	if err != nil {
		h.logger.Error("Failed to export insurance checks to Excel", "error", err)
		response.SendInternalError(c, "Failed to export insurance checks to Excel")
		return
	}

	h.sendExcelResponse(c, excelData, filename)
}

// ExportChargesToExcel exports charges to Excel
func (h *DashboardHandler) ExportChargesToExcel(c *gin.Context) {
	claimID := c.Query("claimId")
	if claimID == "" {
		response.SendValidationError(c, "claimId is required")
		return
	}

	year, month := parseYearMonthParams(c)

	// Get Excel file
	excelData, filename, err := h.service.ExportChargesToExcel(c.Request.Context(), claimID, year, month)
	if err != nil {
		h.logger.Error("Failed to export charges to Excel", "error", err)
		response.SendInternalError(c, "Failed to export charges to Excel")
		return
	}

	h.sendExcelResponse(c, excelData, filename)
}

