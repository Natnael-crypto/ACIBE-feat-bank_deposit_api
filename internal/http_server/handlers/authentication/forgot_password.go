package authenticationapi

import (
	"net/http"
	"strings"

	"github.com/aci/backend/internal/core/services/user"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordHandler struct {
	userService user.Service
}

func NewForgotPasswordHandler(userService user.Service) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		userService: userService,
	}
}

// ForgotPassword godoc
// @Summary      Forgot Password
// @Description  Request a password reset link via email
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      ForgotPasswordRequest  true  "Forgot password request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/forgot-password [post]
func (h *ForgotPasswordHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request format")
		return
	}

	err := h.userService.ForgotPassword(c.Request.Context(), &user.ForgotPasswordRequest{
		Email: req.Email,
	})
	if err != nil {
		// User not found - return success message without revealing user existence
		if err.Error() == "user_not_found_silent" {
			response.SendSuccess(c, "If an account exists with this email, a password reset link has been sent.", nil)
			return
		}

		// Email sending error
		if strings.Contains(err.Error(), "email sending failed:") {
			response.SendError(c, "Failed to send email. Please try again.", http.StatusInternalServerError, err.Error())
			return
		}

		// Other errors
		response.SendError(c, "An error occurred. Please try again.", http.StatusInternalServerError, err.Error())
		return
	}

	// Email sent successfully
	response.SendSuccess(c, "Reset link sent to your email.", nil)
}

