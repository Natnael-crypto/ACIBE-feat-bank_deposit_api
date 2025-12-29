package routes

import (
	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/internal/core/services/user"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/http_server/middlewares"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// SetupUserRoutes configures all user management routes
func SetupUserRoutes(
	router *gin.Engine,
	authService auth.Service,
	userService user.Service,
	userRepo repository.UserRepository,
	deptRepo repository.DepartmentRepository,
) {
	api := router.Group("/api")
	protected := api.Group("")
	protected.Use(middlewares.RequireAuthN(authService))
	protected.Use(middlewares.RequireActiveUser(userRepo))

	{
		userGroup := protected.Group("/users")
		{
			userHandler := userhandler.NewUserHandler(userService, deptRepo, userRepo)

			userGroup.GET("", userHandler.GetUsers) // Supports optional ?search=query parameter
			userGroup.GET("/:id", userHandler.GetUser)
			userGroup.POST("", middlewares.RequireAdmin(userRepo), userHandler.CreateUser)
			userGroup.PUT("/:id", userHandler.UpdateUser)
			userGroup.PUT("/:id/status", middlewares.RequireAdmin(userRepo), userHandler.UpdateUserStatus)
			userGroup.PATCH("/:id/status", middlewares.RequireAdmin(userRepo), userHandler.UpdateUserStatus) // Support PATCH for frontend compatibility
			userGroup.PUT("/:id/password", userHandler.ChangePassword)
			userGroup.DELETE("/:id", middlewares.RequireAdmin(userRepo), userHandler.DeleteUser)
		}
	}
}

