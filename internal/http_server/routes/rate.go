package routes

import (
	"log/slog"

	"github.com/aci/backend/internal/core/services/auth"
	rateservice "github.com/aci/backend/internal/core/services/rates"
	ratehandler "github.com/aci/backend/internal/http_server/handlers/rates"
	"github.com/aci/backend/internal/http_server/middlewares"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

func SetupRateRoutes(
	router *gin.Engine,
	authService auth.Service,
	rateService *rateservice.Service,
	userRepo repository.UserRepository,
	logger *slog.Logger,
) {
	api := router.Group("/api")
	protected := api.Group("/rates")
	protected.Use(middlewares.RequireAuthN(authService))
	protected.Use(middlewares.RequireActiveUser(userRepo))

	{
		rateHandler := ratehandler.NewRateHandler(rateService, logger)

		// Get all outpatient rates
		protected.GET("/outpatient/all", rateHandler.GetAllOutpatientRates)

		// Get outpatient rate by code and modifiers (using /by-code to avoid conflict with existing /outpatient route)
		protected.GET("/outpatient/by-code", rateHandler.GetOutpatientRate)

		// Create outpatient rate (using /create to avoid conflict with existing /outpatient POST route)
		protected.POST("/outpatient/create", rateHandler.CreateOutpatientRate)
	}
}