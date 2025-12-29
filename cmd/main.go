// @title           ACI Backend API
// @version         1.0
// @description     This is the ACI Backend API server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	_ "net/http/pprof"

	_ "github.com/aci/backend/docs"
	"github.com/aci/backend/internal/collaborate"
	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/internal/core/services/claim"
	dashboardservice "github.com/aci/backend/internal/core/services/dashboard"
	"github.com/aci/backend/internal/core/services/department"
	"github.com/aci/backend/internal/core/services/insurance"
	"github.com/aci/backend/internal/core/services/payment"
	rateservice "github.com/aci/backend/internal/core/services/rates"
	userservice "github.com/aci/backend/internal/core/services/user"
	cronjobs "github.com/aci/backend/internal/cron"
	httpserver "github.com/aci/backend/internal/http_server"
	"github.com/aci/backend/internal/repository"
	dashboardrepo "github.com/aci/backend/internal/repository/dashboard"
	"github.com/aci/backend/pkg/email"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	conn, close, err := repository.OpenDB(cfg.DbUrl)
	if err != nil {
		panic(err)
	}
	defer close()
	logger := slog.Default()
	store := repository.NewStore(conn)
	collabApi := collaborate.NewAPI(cfg.Collaborate.Username, cfg.Collaborate.Password, store)
	// Initialize repositories
	userRepo := repository.NewUserRepository(store)
	tokenRepo := repository.NewPasswordResetTokenRepository(store)
	deptRepo := repository.NewDepartmentRepository(store)
	dashboardRepo := dashboardrepo.NewRepository(store)
	
	// Initialize email service - use env vars directly like ACIBE_backend
	// This ensures it uses the same .env values that work in ACIBE_backend
	emailService := email.NewEmailServiceFromEnv()
	slog.Info("Email service initialized from environment", "provider", os.Getenv("EMAIL_PROVIDER"), "from", os.Getenv("EMAIL_FROM"))

	// Initialize services
	authService := auth.NewService(userRepo, cfg)
	userService := userservice.NewService(userRepo, tokenRepo, emailService, cfg)
	deptService := department.NewService(deptRepo)
	claimSvc := claim.NewService(store, collabApi, logger)
	paymentSvc := payment.NewService(store, collabApi, logger)
	insuranceSvc := insurance.NewService(store, collabApi, logger)
	// Option 1: Run payment sync job (uncomment to use instead of server)
	// claimSvc := claim.NewService(store, collabApi, slog.Default())
	// claimSvc.SyncClaims(context.Background(), time.Now())
	// return
	// paymentSvc := payment.NewService(store, collabApi, slog.Default())
	// paymentSvc.SyncPayments(context.Background(), time.Now())
	// return

	// Option 2: Start HTTP Server with Gin (default - currently active)
	dashboardService := dashboardservice.NewService(dashboardRepo, logger)
	rateService := rateservice.NewService(store)

	// Create HTTP server with all dependencies
	app, err := httpserver.NewApp(
		strconv.Itoa(cfg.Port),
		logger,
		httpserver.Dependencies{
			AuthService:        authService,
			UserService:        userService,
			DeptService:        deptService,
			DashboardService:   dashboardService,
			RateService:        rateService,
			UserRepo:           userRepo,
			DeptRepo:           deptRepo,
			Config:             cfg,
		},
	)
	if err != nil {
		logger.Error("Failed to create HTTP server", "error", err)
		os.Exit(1)
	}

	// Start cron jobs for scheduled tasks (e.g., dashboard materialized view refresh)
	cronJobs := cronjobs.NewCronJobs(claimSvc, paymentSvc, insuranceSvc, store)
	cronJobs.Start()
	defer cronJobs.Stop()

	// Start server
	logger.Info("Starting HTTP server", "port", cfg.Port)
	if err := app.Run(); err != nil {
		logger.Error("failed to run server", "error", err)
		os.Exit(1)
	}

	// Variables kept for easy switching between server and sync job modes
	_ = store
	_ = collabApi
	_ = context.Background()
	_ = time.Now()
	_ = payment.Service{}
}
