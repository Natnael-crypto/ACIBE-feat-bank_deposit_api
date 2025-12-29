package dashboard

import (
	"net/http"
	"strconv"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *DashboardHandler) GetPaymentsData(c *gin.Context) {
	// Parse query parameters
	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear >= 0 {
			year = &parsedYear
		}
	}

	var month *int
	if monthStr := c.Query("month"); monthStr != "" {
		if parsedMonth, err := strconv.Atoi(monthStr); err == nil && parsedMonth >= 0 && parsedMonth <= 12 {
			month = &parsedMonth
		}
	}

	// Parse excludePeer parameter (defaults to false)
	excludePeer := false
	if excludePeerStr := c.Query("exclude_peer"); excludePeerStr != "" {
		if parsedExcludePeer, err := strconv.ParseBool(excludePeerStr); err == nil {
			excludePeer = parsedExcludePeer
		}
	}

	// Parse peerOnly parameter (defaults to false)
	peerOnly := false
	if peerOnlyStr := c.Query("peer_only"); peerOnlyStr != "" {
		if parsedPeerOnly, err := strconv.ParseBool(peerOnlyStr); err == nil {
			peerOnly = parsedPeerOnly
		}
	}

	// Parse search parameter (optional)
	var search *string
	if searchStr := c.Query("search"); searchStr != "" {
		search = &searchStr
	}

	// Parse claim_type parameter (optional)
	// Frontend sends 'type', but we also support 'claim_type' for backward compatibility
	claimType := c.Query("type")
	if claimType == "" {
		claimType = c.Query("claim_type")
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 15
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit == 0 {
				// Limit 0 means return all data - convert to max int32
				limit = 2147483647
				page = 1 // When returning all, always use page 1
			} else if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	// Parse filter parameters
	var claimsExpectedFilter *string
	if filterStr := c.Query("claims_expected_filter"); filterStr != "" {
		claimsExpectedFilter = &filterStr
	}

	var paymentStatusFilter *string
	if filterStr := c.Query("payment_status_filter"); filterStr != "" {
		paymentStatusFilter = &filterStr
	}

	h.logger.Info("Fetching paginated payments", "year", year, "month", month, "excludePeer", excludePeer, "peerOnly", peerOnly, "search", search, "claimType", claimType, "page", page, "limit", limit, "claimsExpectedFilter", claimsExpectedFilter, "paymentStatusFilter", paymentStatusFilter)

	// Get paginated payments from service
	data, err := h.service.GetPayments(c.Request.Context(), year, month, excludePeer, peerOnly, search, claimType, page, limit, claimsExpectedFilter, paymentStatusFilter)
	if err != nil {
		h.logger.Error("Failed to fetch paginated payments", "error", err)
		response.SendInternalError(c, "Failed to fetch paginated payments")
		return
	}

	h.logger.Info("Paginated payments retrieved successfully", "totalCount", data.Pagination.TotalRecords, "page", page)

	// Return response
	response.SendSuccess(c, "Paginated payments retrieved successfully", data)
}

// GetPaymentsByCheckNumber godoc
// @Summary      Get Payments By Check Number
// @Description  Retrieve paginated payments for a specific check number (TRN), optionally filtered by year
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        check_number           query     string  false  "Check number (TRN)"
// @Param        year                   query     int     false  "Year (optional, 0 = all years)"
// @Param        page                   query     int     false  "Page number (default: 1)"
// @Param        limit                  query     int     false  "Items per page (default: 15)"
// @Param        claims_expected_filter  query     string  false  "Claims vs Expected filter (all, claimsGreater, expectedGreater)"
// @Param        payment_status_filter  query     string  false  "Payment status filter (all, overpaid, underpaid, fullyPaid)"
// @Param        search                 query     string  false  "Search term (optional)"
// @Success      200  {object}  dashboard.PaymentsResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/payments/by-check-number [get]
func (h *DashboardHandler) GetPaymentsByCheckNumber(c *gin.Context) {
	// Parse check_number parameter (required)
	checkNumber := c.Query("check_number")
	if checkNumber == "" {
		response.SendError(c, "check_number parameter is required", http.StatusBadRequest)
		return
	}

	// Parse year parameter (optional, 0 = all years, default to 0 if not provided)
	year := 0
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear > 0 {
			year = parsedYear
		}
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 15
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit == 0 {
				// Limit 0 means return all data - convert to max int32
				limit = 2147483647
				page = 1 // When returning all, always use page 1
			} else if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	// Parse filter parameters
	var claimsExpectedFilter *string
	if filter := c.Query("claims_expected_filter"); filter != "" && filter != "all" {
		claimsExpectedFilter = &filter
	}

	var paymentStatusFilter *string
	if filter := c.Query("payment_status_filter"); filter != "" && filter != "all" {
		paymentStatusFilter = &filter
	}

	// Parse search parameter
	var search *string
	if searchStr := c.Query("search"); searchStr != "" {
		search = &searchStr
	}

	h.logger.Info("Fetching payments by check number", "checkNumber", checkNumber, "year", year, "page", page, "limit", limit, "claimsExpectedFilter", claimsExpectedFilter, "paymentStatusFilter", paymentStatusFilter, "search", search)

	// Get paginated payments from service
	data, err := h.service.GetPaymentsByCheckNumber(c.Request.Context(), checkNumber, year, page, limit, claimsExpectedFilter, paymentStatusFilter, search)
	if err != nil {
		h.logger.Error("Failed to fetch payments by check number", "error", err)
		response.SendInternalError(c, "Failed to fetch payments by check number")
		return
	}

	h.logger.Info("Payments by check number retrieved successfully", "totalCount", data.Pagination.TotalRecords, "page", page)

	// Return response
	response.SendSuccess(c, "Payments by check number retrieved successfully", data)
}
