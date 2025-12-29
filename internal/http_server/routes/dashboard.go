package routes

import (
	"log/slog"

	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/internal/core/services/dashboard"
	bankdeposithandler "github.com/aci/backend/internal/http_server/handlers/bankdeposit"
	dashboardhandler "github.com/aci/backend/internal/http_server/handlers/dashboard"
	"github.com/aci/backend/internal/http_server/middlewares"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// SetupDashboardRoutes configures all dashboard routes
func SetupDashboardRoutes(
	router *gin.Engine,
	authService auth.Service,
	dashboardService dashboard.Service,
	userRepo repository.UserRepository,
	logger *slog.Logger,
) {
	api := router.Group("/api")
	protected := api.Group("")
	protected.Use(middlewares.RequireAuthN(authService))
	protected.Use(middlewares.RequireActiveUser(userRepo))

	{
		dashboardGroup := protected.Group("/dashboard")
		{
		dashboardHandler := dashboardhandler.NewDashboardHandler(dashboardService, logger)
		bankDepositHandler := bankdeposithandler.NewBankDepositHandler(dashboardService, logger)

			// Get dashboard data - available to all authenticated users
			dashboardGroup.GET("", dashboardHandler.GetDashboardData)

			// Get yearly dashboard data
			dashboardGroup.GET("/yearly", dashboardHandler.GetYearlyDashboardData)
			
			// Get available years - available to all authenticated users
			dashboardGroup.GET("/years", dashboardHandler.GetAvailableYears)

			// Get claims data via year and month with pagination
            dashboardGroup.GET("/claims", dashboardHandler.GetClaimsData)

			// Get claim data by id
			dashboardGroup.GET("/claims/by-id", dashboardHandler.GetClaimById)
			
			// Get claim charges data via claim id and charge id
			dashboardGroup.GET("/claims/charges", dashboardHandler.GetClaimCharges)

			// Get payment checks by claim and charge (alias for /payment-checks)
			dashboardGroup.GET("/claims/payments", dashboardHandler.GetPaymentChecksByClaimAndCharge)

			//Get payments data
			dashboardGroup.GET("/payments", dashboardHandler.GetPaymentsData)

			//Get payments by check number
			dashboardGroup.GET("/payments/by-check-number", dashboardHandler.GetPaymentsByCheckNumber)
			dashboardGroup.GET("/payment-checks", dashboardHandler.GetPaymentChecksByClaimAndCharge)

			//Get bank deposit data
			dashboardGroup.GET("/bank-deposit", bankDepositHandler.GetBankDepositData)
			dashboardGroup.GET("/bank-deposit/claims", bankDepositHandler.GetBankDepositClaims)
			dashboardGroup.GET("/bank-deposit/applied-payments", bankDepositHandler.GetBankDepositAppliedPayments)

			//Get insurance checks
			dashboardGroup.GET("/insurance-checks", dashboardHandler.GetInsuranceChecks)

			// Export endpoints
			dashboardGroup.GET("/export/claims", dashboardHandler.ExportClaimsToExcel)
			dashboardGroup.GET("/export/payments", dashboardHandler.ExportPaymentsToExcel)
			dashboardGroup.GET("/export/payments/by-claim-charge", dashboardHandler.ExportPaymentsByClaimAndChargeToExcel)
			dashboardGroup.GET("/export/insurance-checks", dashboardHandler.ExportInsuranceChecksToExcel)
			dashboardGroup.GET("/export/charges", dashboardHandler.ExportChargesToExcel)
		}
	}
}

