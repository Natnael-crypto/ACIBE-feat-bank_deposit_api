package dashboard

import (
	"net/http"
	"strconv"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetPaymentChecksByClaimAndCharge godoc
// @Summary      Get Payment Checks by Claim and Charge
// @Description  Retrieve all individual payment records (not grouped) for a specific claim_id and charge_id
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        claim_id   query     string  true  "Claim ID"
// @Param        charge_id  query     string  true  "Charge ID"
// @Param        year       query     int     false "Year (optional)"
// @Param        month      query     int     false "Month (optional, 1-12)"
// @Param        page       query     int     false "Page number (default: 1)"
// @Param        limit      query     int     false "Items per page (default: 25)"
// @Success      200  {object}  dashboard.PaymentChecksResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/payment-checks [get]
func (h *DashboardHandler) GetPaymentChecksByClaimAndCharge(c *gin.Context) {
	claimID := c.Query("claim_id")
	chargeID := c.Query("charge_id")

	if claimID == "" {
		response.SendError(c, "claim_id is required", http.StatusBadRequest)
		return
	}

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

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 25
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	h.logger.Info("Fetching payment checks", "claimID", claimID, "chargeID", chargeID, "year", year, "month", month, "page", page, "limit", limit)

	// Get payment checks from service
	data, err := h.service.GetPaymentChecksByClaimAndCharge(c.Request.Context(), claimID, chargeID, year, month, page, limit)
	if err != nil {
		h.logger.Error("Failed to fetch payment checks", "error", err)
		response.SendInternalError(c, "Failed to fetch payment checks")
		return
	}

	h.logger.Info("Payment checks retrieved successfully", "totalCount", data.Pagination.TotalRecords)

	// Return response
	response.SendSuccess(c, "Payment checks retrieved successfully", data)
}
