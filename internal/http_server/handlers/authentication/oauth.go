package authenticationapi

import (
	"net/http"

	"github.com/aci/backend/internal/core/services/auth"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type OAuthLoginRequest struct {
	Provider       string `json:"provider" binding:"required,oneof=google microsoft"`
	AccessToken    string `json:"access_token" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Name           string `json:"name"`
	ProviderUserID string `json:"provider_user_id" binding:"required"`
}

type OAuthHandler struct {
	oauthService   auth.OAuthService
	userRepo       repository.UserRepository
	departmentRepo repository.DepartmentRepository
}

func NewOAuthHandler(oauthService auth.OAuthService, userRepo repository.UserRepository, departmentRepo repository.DepartmentRepository) *OAuthHandler {
	return &OAuthHandler{
		oauthService:   oauthService,
		userRepo:       userRepo,
		departmentRepo: departmentRepo,
	}
}

// OAuthLogin godoc
// @Summary      OAuth Login
// @Description  Authenticate user using OAuth provider (Google/Microsoft)
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        oauth  body      OAuthLoginRequest  true  "OAuth login request"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/oauth [post]
func (h *OAuthHandler) OAuthLogin(c *gin.Context) {
	var req OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request format")
		return
	}

	tokenPair, userInfo, err := h.oauthService.OAuthLogin(
		c.Request.Context(),
		req.Provider,
		req.AccessToken,
		req.Email,
		req.Name,
		req.ProviderUserID,
	)
	if err != nil {
		response.SendError(c, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get full user from database to include departments
	// We need to get the userRepo from oauthService, but it's not exposed
	// So we'll need to add userRepo to the handler or fetch user by ID
	// For now, let's add userRepo to the handler
	user, err := h.userRepo.GetByID(c.Request.Context(), userInfo.ID)
	if err != nil {
		response.SendError(c, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := userhandler.ToUserResponse(user, h.departmentRepo)

	response.SendSuccess(c, "OAuth login successful", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"user":          userResponse,
	})
}

