package middlewares

import (
	"net/http"
	"strings"

	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequireAuthN validates JWT tokens and extracts user info to context
func RequireAuthN(authService auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			response.SendError(c, "Authorization header required", http.StatusUnauthorized)
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authorizationHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.SendError(c, "Invalid authorization header format", http.StatusUnauthorized)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			response.SendError(c, "Invalid or expired token", http.StatusUnauthorized)
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
