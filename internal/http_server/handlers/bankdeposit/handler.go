package bankdeposit

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"log/slog"

	dashboardservice "github.com/aci/backend/internal/core/services/dashboard"
	"github.com/aci/backend/internal/types/dashboard"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type BankDepositHandler struct {
	service dashboardservice.Service
	logger  *slog.Logger
}

// Reference to dashboard types for swagger documentation
var _ = dashboard.BankDepositMonthData{}

func NewBankDepositHandler(service dashboardservice.Service, logger *slog.Logger) *BankDepositHandler {
	return &BankDepositHandler{
		service: service,
		logger:  logger,
	}
}

// GetBankDepositData godoc
// @Summary      Get Bank Deposit Data
// @Description  Retrieve bank deposit data aggregated by month for a specific year
// @Tags         Bank Deposit
// @Accept       json
// @Produce      json
// @Param        year   query     int     false  "Year (defaults to current year)"
// @Success      200  {array}  dashboard.BankDepositMonthData
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/bank-deposit [get]
func (h *BankDepositHandler) GetBankDepositData(c *gin.Context) {
	// Get year from query parameter, default to current year
	year := time.Now().Year()
	if yearStr := c.Query("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil && parsedYear > 1900 && parsedYear < 2100 {
			year = parsedYear
		} else {
			h.logger.Warn("Invalid year parameter", "year", yearStr, "error", err)
		}
	}

	h.logger.Info("Fetching bank deposit data", "year", year)

	// Create a context with longer timeout for bank deposit query (5 minutes)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Get bank deposit data from service
	data, err := h.service.GetBankDepositData(ctx, year)
	if err != nil {
		h.logger.Error("Failed to fetch bank deposit data", "year", year, "error", err)
		response.SendInternalError(c, "Failed to fetch bank deposit data")
		return
	}

	h.logger.Info("Bank deposit data retrieved successfully", "year", year, "months", len(data))

	// Return response matching frontend format
	response.SendSuccess(c, "Bank deposit data retrieved successfully", data)
}

// GetBankDepositClaims godoc
// @Summary      Get Bank Deposit Claims
// @Description  Retrieve paginated claims for bank deposit data filtered by year and month
// @Tags         Bank Deposit
// @Accept       json
// @Produce      json
// @Param        year   query     int     true   "Year (required)"
// @Param        month  query     int     false  "Month (0 = all months, 1-12 = specific month)"
// @Param        search query     string  false  "Search term (optional)"
// @Param        cursor query     string  false  "Cursor for pagination (optional)"
// @Param        limit  query     int     false  "Items per page (default: 15)"
// @Success      200  {object}  dashboard.ClaimsResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/bank-deposit/claims [get]
func (h *BankDepositHandler) GetBankDepositClaims(c *gin.Context) {
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
	cursor := c.Query("cursor")
	limit := 15
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit == 0 {
				limit = 2147483647
				cursor = ""
			} else if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	h.logger.Info("Fetching bank deposit claims", "year", year, "month", month, "search", search, "cursor", cursor, "limit", limit)

	// Get bank deposit claims from service
	data, err := h.service.GetBankDepositClaims(c.Request.Context(), year, month, search, limit, cursor)
	if err != nil {
		h.logger.Error("Failed to fetch bank deposit claims", "error", err)
		response.SendInternalError(c, "Failed to fetch bank deposit claims")
		return
	}

	h.logger.Info("Bank deposit claims retrieved successfully", "totalCount", data.Pagination.TotalRecords, "hasMore", data.Pagination.HasMore)

	// Return response
	response.SendSuccess(c, "Bank deposit claims retrieved successfully", data)
}

// GetBankDepositAppliedPayments godoc
// @Summary      Get Bank Deposit Applied Payments
// @Description  Retrieve paginated payments for bank deposit applied amount filtered by year and month
// @Tags         Bank Deposit
// @Accept       json
// @Produce      json
// @Param        year   query     int     true   "Year (required)"
// @Param        month  query     int     false  "Month (0 = all months, 1-12 = specific month)"
// @Param        search query     string  false  "Search term (optional)"
// @Param        cursor query     string  false  "Cursor for pagination (optional)"
// @Param        limit  query     int     false  "Items per page (default: 15)"
// @Success      200  {object}  dashboard.PaymentsResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /dashboard/bank-deposit/applied-payments [get]
func (h *BankDepositHandler) GetBankDepositAppliedPayments(c *gin.Context) {
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
	cursor := c.Query("cursor")
	limit := 15
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit == 0 {
				limit = 2147483647
				cursor = ""
			} else if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	h.logger.Info("Fetching bank deposit applied payments", "year", year, "month", month, "search", search, "cursor", cursor, "limit", limit)

	// Get bank deposit applied payments from service
	data, err := h.service.GetBankDepositAppliedPayments(c.Request.Context(), year, month, search, limit, cursor)
	if err != nil {
		h.logger.Error("Failed to fetch bank deposit applied payments", "error", err)
		response.SendInternalError(c, "Failed to fetch bank deposit applied payments")
		return
	}

	h.logger.Info("Bank deposit applied payments retrieved successfully", 
		"totalCount", data.Pagination.TotalRecords, 
		"hasMore", data.Pagination.HasMore,
		"itemsReturned", len(data.Data),
		"page", data.Pagination.Page,
		"limit", data.Pagination.Limit,
		"nextCursor", data.Pagination.NextCursor)

	// Log sample of response data (first 3 items for debugging)
	if len(data.Data) > 0 {
		sampleSize := 3
		if len(data.Data) < sampleSize {
			sampleSize = len(data.Data)
		}
		for i := 0; i < sampleSize; i++ {
			payment := data.Data[i]
			// Dereference pointers for logging
			totalCheckNumber := "nil"
			if payment.TotalCheckNumber != nil {
				totalCheckNumber = strconv.Itoa(*payment.TotalCheckNumber)
			}
			trn := "nil"
			if payment.Trn != nil {
				trn = *payment.Trn
			}
			h.logger.Info("Sample payment data", 
				"index", i,
				"claimId", payment.ClaimID,
				"chargeId", payment.ChargeID,
				"totalPaidAmount", payment.TotalPaidAmount,
				"totalExpectedAmount", payment.TotalExpectedAmount,
				"totalCheckNumber", totalCheckNumber,
				"trn", trn)
		}
	}

	// Return response
	response.SendSuccess(c, "Bank deposit applied payments retrieved successfully", data)
}

