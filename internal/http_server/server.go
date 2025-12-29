package httpserver

import (
	"log/slog"
	"strconv"

	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/core/services/auth"
	dashboardservice "github.com/aci/backend/internal/core/services/dashboard"
	"github.com/aci/backend/internal/core/services/department"
	rateservice "github.com/aci/backend/internal/core/services/rates"
	userservice "github.com/aci/backend/internal/core/services/user"
	"github.com/aci/backend/internal/http_server/handlers/rates"
	"github.com/aci/backend/internal/http_server/routes"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

type App struct {
	gin    *gin.Engine
	port   int
	logger *slog.Logger
}

type Dependencies struct {
	AuthService        auth.Service
	UserService        userservice.Service
	DeptService        department.Service
	DashboardService   dashboardservice.Service
	RateService        *rateservice.Service
	UserRepo           repository.UserRepository
	DeptRepo           repository.DepartmentRepository
	Config             config.Config
}

func NewApp(
	port string,
	logger *slog.Logger,
	deps Dependencies,
) (*App, error) {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	a := &App{
		gin:    gin.Default(),
		port:   portInt,
		logger: logger,
	}

	// Add CORS middleware
	a.gin.Use(corsMiddleware())

	// Register all routes
	routes.SetupAuthRoutes(a.gin, deps.AuthService, deps.UserService, deps.UserRepo, deps.DeptRepo, deps.Config)
	routes.SetupUserRoutes(a.gin, deps.AuthService, deps.UserService, deps.UserRepo, deps.DeptRepo)
	routes.SetupDepartmentRoutes(a.gin, deps.AuthService, deps.DeptService, deps.UserRepo, deps.DeptRepo)
	routes.SetupDashboardRoutes(a.gin, deps.AuthService, deps.DashboardService, deps.UserRepo, logger)
	routes.SetupRateRoutes(a.gin, deps.AuthService, deps.RateService, deps.UserRepo, logger)

	// Register existing rates routes
	rates.NewAuthenticationHandlers(a.gin, logger)

	// Register Swagger and pprof
	a.swaggerRoute()

	return a, nil
}
func (a *App) Run() error {
	addr := ":" + strconv.Itoa(a.port)
	return a.gin.Run(addr)
}

// TODO: Add a proper CORS policy
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// allowedOrigins := map[string]bool{
		// 	"http://localhost:3000": true,
		// }

		origin := c.Request.Header.Get("Origin")
		// if allowedOrigins[origin] {
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		// }

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
