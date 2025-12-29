package routes

import (
	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/internal/core/services/user"
	authenticationapi "github.com/aci/backend/internal/http_server/handlers/authentication"
	"github.com/aci/backend/internal/http_server/middlewares"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)


// SetupAuthRoutes configures all authentication-related routes
func SetupAuthRoutes(
	router *gin.Engine,
	authService auth.Service,
	userService user.Service,
	userRepo repository.UserRepository,
	deptRepo repository.DepartmentRepository,
	cfg config.Config,
) {
	api := router.Group("/api")
	{
		// Public auth routes
		public := api.Group("")
		{
			authGroup := public.Group("/auth")
			{
				loginHandler := authenticationapi.NewLoginHandler(userService, deptRepo)
				oauthService := auth.NewOAuthService(userRepo, cfg)
				oauthHandler := authenticationapi.NewOAuthHandler(oauthService, userRepo, deptRepo)
				refreshHandler := authenticationapi.NewRefreshHandler(userService, deptRepo)
				forgotPasswordHandler := authenticationapi.NewForgotPasswordHandler(userService)
				resetPasswordHandler := authenticationapi.NewResetPasswordHandler(userService, deptRepo)

				authGroup.POST("/login", loginHandler.Login)
				authGroup.POST("/oauth", oauthHandler.OAuthLogin)
				authGroup.POST("/refresh", refreshHandler.RefreshToken)
				authGroup.POST("/forgot-password", forgotPasswordHandler.ForgotPassword)
				authGroup.GET("/validate-reset-token", resetPasswordHandler.ValidateResetToken)
				authGroup.POST("/reset-password", resetPasswordHandler.ResetPassword)
			}
		}

		// Protected auth routes
		protected := api.Group("")
		protected.Use(middlewares.RequireAuthN(authService))
		protected.Use(middlewares.RequireActiveUser(userRepo))
		{
			protectedAuth := protected.Group("/auth")
			{
				meHandler := authenticationapi.NewMeHandler(authService, userRepo, deptRepo)
				logoutHandler := authenticationapi.NewLogoutHandler(authService)
				protectedAuth.GET("/me", meHandler.GetCurrentUser)
				protectedAuth.POST("/logout", logoutHandler.Logout)
			}
		}
	}
}

