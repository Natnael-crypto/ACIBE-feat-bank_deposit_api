package authenticationapi

import (
	"net/http"

	"github.com/aci/backend/internal/core/services/auth"
	userhandler "github.com/aci/backend/internal/http_server/handlers/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MeHandler struct {
	authService   auth.Service
	userRepo      repository.UserRepository
	departmentRepo repository.DepartmentRepository
}

func NewMeHandler(authService auth.Service, userRepo repository.UserRepository, departmentRepo repository.DepartmentRepository) *MeHandler {
	return &MeHandler{
		authService:   authService,
		userRepo:      userRepo,
		departmentRepo: departmentRepo,
	}
}

// GetCurrentUser godoc
// @Summary      Get Current User
// @Description  Get the currently authenticated user's information
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *MeHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendError(c, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get full user info to check OAuth status
	userIDUUID, ok := userID.(uuid.UUID)
	if !ok {
		// Try to parse if it's a string
		if userIDStr, ok := userID.(string); ok {
			parsedID, err := uuid.Parse(userIDStr)
			if err != nil {
				response.SendError(c, "Invalid user ID", http.StatusBadRequest)
				return
			}
			userIDUUID = parsedID
		} else {
			response.SendError(c, "Invalid user ID format", http.StatusBadRequest)
			return
		}
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userIDUUID)
	if err != nil {
		response.SendError(c, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Determine authentication method
	authMethods := []string{"password"} // Default: password authentication

	var googleID, microsoftID *string
	if user.GoogleID.Valid {
		googleID = &user.GoogleID.String
		if *googleID != "" {
			authMethods = append(authMethods, "google")
		}
	}
	if user.MicrosoftID.Valid {
		microsoftID = &user.MicrosoftID.String
		if *microsoftID != "" {
			authMethods = append(authMethods, "microsoft")
		}
	}

	// Transform user to frontend format (with departments)
	userResponse := userhandler.ToUserResponse(user, h.departmentRepo)
	
	response.SendSuccess(c, "User info retrieved", gin.H{
		"id":            userResponse.ID,
		"email":         userResponse.Email,
		"role":          userResponse.Role,
		"first_name":    userResponse.FirstName,
		"last_name":     userResponse.LastName,
		"status":         userResponse.Status,
		"created_at":    userResponse.CreatedAt,
		"updated_at":    userResponse.UpdatedAt,
		"departments":    userResponse.Departments,
		"auth_methods":  authMethods,
		"has_google":    googleID != nil && *googleID != "",
		"has_microsoft": microsoftID != nil && *microsoftID != "",
	})
}

