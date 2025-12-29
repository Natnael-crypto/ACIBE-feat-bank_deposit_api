package dashboard

import (
	"net/http"
	"strconv"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetInsuranceChecks godoc
// @Summary      Get Insurance Checks
// @Description  Retrieve paginated insurance checks filtered by year, month, and search
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        year   query     int     true   "Year (required)"
// @Param        month  query     int     false  "Month (0 = all months, 1-12 = specific month)"
// @Param        search query     string  false  "Search term (optional)"
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        limit  query     int     false  "Items per page (default: 15)"
// @Success      200  {object}  dashboard.InsuranceChecksResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/insurance-checks [get]
func (h *DashboardHandler) GetInsuranceChecks(c *gin.Context) {
	// Parse year parameter (required)
	year := 0
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear > 0 {
			year = parsedYear
		}
	}
	if year == 0 {
		response.SendError(c, "year parameter is required and must be greater than 0", http.StatusBadRequest)
		return
	}

	// Parse month parameter (optional, 0 = all months)
	month := 0
	if monthStr := c.Query("month"); monthStr != "" {
		if parsedMonth, err := strconv.Atoi(monthStr); err == nil && parsedMonth >= 0 && parsedMonth <= 12 {
			month = parsedMonth
		}
	}

	// Parse search parameter (optional)
	search := c.Query("search")

	// Parse pagination parameters
	cursor := c.Query("cursor") // Cursor for infinite scroll
	limit := 15
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit == 0 {
				// Limit 0 means return all data - convert to max int32
				limit = 2147483647
				cursor = "" // When returning all, reset cursor
			} else if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	h.logger.Info("Fetching insurance checks", "year", year, "month", month, "search", search, "cursor", cursor, "limit", limit)

	// Get insurance checks from service
	data, err := h.service.GetInsuranceChecks(c.Request.Context(), year, month, search, limit, cursor)
	if err != nil {
		h.logger.Error("Failed to fetch insurance checks", "error", err)
		response.SendInternalError(c, "Failed to fetch insurance checks")
		return
	}

	h.logger.Info("Insurance checks retrieved successfully", "totalCount", data.Pagination.TotalRecords, "hasMore", data.Pagination.HasMore)

	// Return response
	response.SendSuccess(c, "Insurance checks retrieved successfully", data)
}
