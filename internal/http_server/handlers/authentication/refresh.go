package authenticationapi

import (
	"net/http"

	"github.com/aci/backend/internal/core/services/user"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshHandler struct {
	userService    user.Service
	departmentRepo repository.DepartmentRepository
}

func NewRefreshHandler(userService user.Service, departmentRepo repository.DepartmentRepository) *RefreshHandler {
	return &RefreshHandler{
		userService:    userService,
		departmentRepo: departmentRepo,
	}
}

// RefreshToken godoc
// @Summary      Refresh Token
// @Description  Refresh access token using refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        refresh  body      RefreshTokenRequest  true  "Refresh token request"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *RefreshHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request format")
		return
	}

	authResponse, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.SendError(c, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := userhandler.ToUserResponse(authResponse.User, h.departmentRepo)
	
	// Return transformed response
	response.SendSuccess(c, "Token refreshed successfully", gin.H{
		"user":          userResponse,
		"access_token":  authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"expires_in":    authResponse.ExpiresIn,
	})
}

