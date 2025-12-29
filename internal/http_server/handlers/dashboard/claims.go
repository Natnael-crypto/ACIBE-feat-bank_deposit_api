package dashboard

import (
	"net/http"
	"strconv"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *DashboardHandler) GetClaimsData(c *gin.Context){

	// Parse query parameters
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

	category := c.Query("category")
	claimType := c.Query("type")
	
	// Parse search parameter (optional)
	var search *string
	if searchStr := c.Query("search"); searchStr != "" {
		search = &searchStr
	}
	
	// Parse excludePeer parameter (defaults to false)
	// For outpatient type, always exclude peer (peer is only included in total, not in OUT)
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
	
	// If peerOnly is true, override claimType to "peer" and ensure excludePeer is false
	// This ensures we only get peer claims when peer_only is requested
	if peerOnly {
		claimType = "peer"
		excludePeer = false
	} else if (claimType == "outpatient" || claimType == "OUT") && category != "totalBilled" {
		// If type is outpatient, always exclude peer (peer should only be in total, not in OUT)
		// But not for totalBilled category (user explicitly controls it)
		excludePeer = true
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

	h.logger.Info("Fetching paginated claims", "year", year, "month", month, "category", category, "type", claimType, "search", search, "excludePeer", excludePeer, "peerOnly", peerOnly, "page", page, "limit", limit, "claimsExpectedFilter", claimsExpectedFilter, "paymentStatusFilter", paymentStatusFilter)

	// Get paginated claims from service
	data, err := h.service.GetClaims(c.Request.Context(), year, month, category, claimType, excludePeer, search, peerOnly, page, limit, claimsExpectedFilter, paymentStatusFilter)
	if err != nil {
		h.logger.Error("Failed to fetch paginated claims", "error", err)
		response.SendInternalError(c, "Failed to fetch paginated claims")
		return
	}

	h.logger.Info("Paginated claims retrieved successfully", "totalCount", data.Pagination.TotalRecords, "page", page)

	// Return response - service already returns the correct structure
	response.SendSuccess(c, "Paginated claims retrieved successfully", data)

}

// GetClaimCharges godoc
// @Summary      Get Claim Charges
// @Description  Retrieve all charges (claim_id, cpt_code pairs) for a specific claim
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        claim_id  query     string  true   "Claim ID"
// @Param        year      query     int     false  "Year (optional, 0 = all years)"
// @Param        month     query     int     false  "Month (optional, 0 = all months)"
// @Success      200  {array}  dashboard.ClaimCharge
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/claims/charges [get]
func (h *DashboardHandler) GetClaimCharges(c *gin.Context) {
	claimID := c.Query("claim_id")
	if claimID == "" {
		response.SendError(c, "claim_id parameter is required", http.StatusBadRequest)
		return
	}

	// Check if this is from bank deposit context - if so, skip date filtering
	skipDateFilter := c.Query("bank_deposit") == "true" || c.Query("skip_date_filter") == "true"

	var year *int
	var month *int
	
	if skipDateFilter {
		// Pass 0 to mean "all years/months" (no date filtering)
		zeroYear := 0
		zeroMonth := 0
		year = &zeroYear
		month = &zeroMonth
	} else {
		if yearStr := c.Query("year"); yearStr != "" {
			if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear >= 0 {
				year = &parsedYear
			}
		}

		if monthStr := c.Query("month"); monthStr != "" {
			if parsedMonth, err := strconv.Atoi(monthStr); err == nil && parsedMonth >= 0 && parsedMonth <= 12 {
				month = &parsedMonth
			}
		}
	}

	h.logger.Info("Fetching claim charges", "claimID", claimID, "year", year, "month", month, "skipDateFilter", skipDateFilter)

	charges, err := h.service.GetClaimCharges(c.Request.Context(), claimID, year, month)
	if err != nil {
		h.logger.Error("Failed to fetch claim charges", "claimID", claimID, "error", err)
		response.SendInternalError(c, "Failed to fetch claim charges")
		return
	}

	response.SendSuccess(c, "Claim charges retrieved successfully", charges)
}

// GetClaimById godoc
// @Summary      Get Claim By ID
// @Description  Retrieve all unique claim(s) for a specific claim_id
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        claim_id   query     string  true   "Claim ID"
// @Success      200  {array}  dashboard.Claim
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/claims/by-id [get]
func (h *DashboardHandler) GetClaimById(c *gin.Context) {
	claimID := c.Query("claim_id")
	if claimID == "" {
		response.SendError(c, "claim_id parameter is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("Fetching claim by ID", "claimID", claimID)

	claims, err := h.service.GetClaimById(c.Request.Context(), claimID)
	if err != nil {
		h.logger.Error("Failed to fetch claim by ID", "claimID", claimID, "error", err)
		response.SendInternalError(c, "Failed to fetch claim by ID")
		return
	}

	response.SendSuccess(c, "Claim retrieved successfully", claims)
}
