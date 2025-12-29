package authenticationapi

import (
	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type LogoutRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type LogoutHandler struct {
	authService auth.Service
}

func NewLogoutHandler(authService auth.Service) *LogoutHandler {
	return &LogoutHandler{
		authService: authService,
	}
}

// Logout godoc
// @Summary      Logout
// @Description  Logout user and invalidate session
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        logout  body      LogoutRequest  true  "Logout request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *LogoutHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request data")
		return
	}

	// Validate that access token is provided
	if req.AccessToken == "" {
		response.SendValidationError(c, "Access token is required")
		return
	}

	// Validate the token to ensure it's valid before logout
	// This also extracts user info for potential session tracking
	_, err := h.authService.ValidateToken(c.Request.Context(), req.AccessToken)
	if err != nil {
		// Token is invalid/expired, but logout should still succeed (idempotent)
		response.SendSuccess(c, "Logout successful", nil)
		return
	}

	// For now, we'll just return success since JWT tokens are stateless
	// In a more advanced implementation, you might want to blacklist tokens
	// or use the account.user_session table to track and revoke sessions
	// The token will naturally expire after its TTL (15 minutes for access token)
	// and the frontend should clear tokens from storage

	response.SendSuccess(c, "Logout successful", nil)
}

