package authenticationapi

import (
	"net/http"

	"github.com/aci/backend/internal/core/services/user"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type ResetPasswordHandler struct {
	userService    user.Service
	departmentRepo repository.DepartmentRepository
}

func NewResetPasswordHandler(userService user.Service, departmentRepo repository.DepartmentRepository) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		userService:    userService,
		departmentRepo: departmentRepo,
	}
}

// ResetPassword godoc
// @Summary      Reset Password
// @Description  Reset password using reset token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      ResetPasswordRequest  true  "Reset password request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/reset-password [post]
func (h *ResetPasswordHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request format")
		return
	}

	authResponse, err := h.userService.ResetPasswordWithToken(c.Request.Context(), &user.ResetPasswordWithTokenRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		if err.Error() == "invalid or expired reset token" || err.Error() == "user not found" {
			response.SendError(c, "Invalid or expired reset token", http.StatusUnauthorized, err.Error())
			return
		}
		response.SendInternalError(c, "Failed to reset password")
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := userhandler.ToUserResponse(authResponse.User, h.departmentRepo)
	
	// Return transformed response like login endpoint - allows frontend to auto-login
	response.SendSuccess(c, "Password reset successfully. You have been logged in.", gin.H{
		"user":          userResponse,
		"access_token":  authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"expires_in":    authResponse.ExpiresIn,
	})
}

// ValidateResetToken godoc
// @Summary      Validate Reset Token
// @Description  Validate a password reset token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        token  query     string  true  "Reset token"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/validate-reset-token [get]
func (h *ResetPasswordHandler) ValidateResetToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.SendError(c, "Token parameter is required", http.StatusBadRequest, "missing_token")
		return
	}

	err := h.userService.ValidateResetToken(c.Request.Context(), token)
	if err != nil {
		response.SendError(c, "Invalid or expired reset token. Please request a new password reset.", http.StatusUnauthorized, err.Error())
		return
	}

	response.SendSuccess(c, "Token is valid", nil)
}

