package authenticationapi

import (
	"net/http"

	"github.com/aci/backend/internal/core/services/user"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginHandler struct {
	userService    user.Service
	departmentRepo repository.DepartmentRepository
}

func NewLoginHandler(userService user.Service, departmentRepo repository.DepartmentRepository) *LoginHandler {
	return &LoginHandler{
		userService:    userService,
		departmentRepo: departmentRepo,
	}
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user with email and password
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "Login credentials"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *LoginHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request format")
		return
	}

	authResponse, err := h.userService.Login(c.Request.Context(), &user.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.SendError(c, err.Error(), http.StatusUnauthorized)
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := userhandler.ToUserResponse(authResponse.User, h.departmentRepo)
	
	// Return transformed response
	response.SendSuccess(c, "Login successful", gin.H{
		"user":          userResponse,
		"access_token":  authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
		"expires_in":    authResponse.ExpiresIn,
	})
}

