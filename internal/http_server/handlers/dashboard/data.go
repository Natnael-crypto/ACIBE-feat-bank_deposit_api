package dashboard

import (
	"context"
	"strconv"
	"time"

	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	// Get year from query parameter, default to current year
	year := time.Now().Year()
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear > 1900 && parsedYear < 2100 {
			year = parsedYear
		} else {
			h.logger.Warn("Invalid year parameter", "year", yearStr, "error", err)
		}
	}

	h.logger.Info("Fetching dashboard data", "year", year)

	// Create a context with longer timeout for dashboard query (5 minutes)
	// This allows for complex aggregations on large datasets
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Get dashboard data from service
	data, err := h.service.GetDashboardData(ctx, year)
	if err != nil {
		h.logger.Error("Failed to fetch dashboard data", "year", year, "error", err)
		response.SendInternalError(c, "Failed to fetch dashboard data")
		return
	}

	h.logger.Info("Dashboard data retrieved successfully", "year", year, "months", len(*data))

	// Return response matching frontend format
	// Send the dashboard data directly, not wrapped in another "data" key
	response.SendSuccess(c, "Dashboard data retrieved successfully", data)
}

// GetYearlyDashboardData godoc
// @Summary      Get Yearly Dashboard Data
// @Description  Retrieve aggregated dashboard data across all years
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]dashboard.MonthData
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/yearly [get]
func (h *DashboardHandler) GetYearlyDashboardData(c *gin.Context) {
	h.logger.Info("Fetching yearly dashboard data")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	data, err := h.service.GetYearlyDashboardData(ctx)
	if err != nil {
		h.logger.Error("Failed to fetch yearly dashboard data", "error", err)
		response.SendInternalError(c, "Failed to fetch yearly dashboard data")
		return
	}

	h.logger.Info("Yearly dashboard data retrieved successfully", "years", len(data))

	response.SendSuccess(c, "Yearly dashboard data retrieved successfully", data)
}

func (h *DashboardHandler) GetAvailableYears(c *gin.Context) {
	h.logger.Info("Fetching available years")

	// Get available years from service
	years, err := h.service.GetAvailableYears(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to fetch available years", "error", err)
		response.SendInternalError(c, "Failed to fetch available years")
		return
	}

	h.logger.Info("Available years retrieved successfully", "count", len(years))

	// Return response
	responseData := map[string]interface{}{
		"years": years,
	}

	response.SendSuccess(c, "Available years retrieved successfully", responseData)
}
