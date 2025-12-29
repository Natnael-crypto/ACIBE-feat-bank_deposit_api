package middlewares

import (
	"encoding/json"
	"net/http"

	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireRoles enforces that the authenticated user has one of the allowed roles
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			response.SendError(c, "User not authenticated", http.StatusUnauthorized)
			c.Abort()
			return
		}
		role, _ := roleVal.(string)
		if _, ok := allowed[role]; !ok {
			response.SendError(c, "Forbidden: insufficient role", http.StatusForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAdmin ensures the current authenticated user is admin by querying DB
func RequireAdmin(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("userID")
		if !ok {
			response.SendError(c, "User not authenticated", http.StatusUnauthorized)
			c.Abort()
			return
		}

		var id uuid.UUID
		if u, isUUID := userIDVal.(uuid.UUID); isUUID {
			id = u
		} else if userIDStr, ok := userIDVal.(string); ok {
			parsed, err := uuid.Parse(userIDStr)
			if err != nil {
				response.SendError(c, "Invalid user ID in token", http.StatusUnauthorized)
				c.Abort()
				return
			}
			id = parsed
		} else {
			response.SendError(c, "Invalid user ID format", http.StatusUnauthorized)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			response.SendError(c, "User not found", http.StatusUnauthorized)
			c.Abort()
			return
		}
		if user.Role != "admin" {
			response.SendError(c, "Forbidden: admin only", http.StatusForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireNotEmployee blocks employees but allows admin and other roles
func RequireNotEmployee(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("userID")
		if !ok {
			response.SendError(c, "User not authenticated", http.StatusUnauthorized)
			c.Abort()
			return
		}

		var id uuid.UUID
		if u, isUUID := userIDVal.(uuid.UUID); isUUID {
			id = u
		} else if userIDStr, ok := userIDVal.(string); ok {
			parsed, err := uuid.Parse(userIDStr)
			if err != nil {
				response.SendError(c, "Invalid user ID in token", http.StatusUnauthorized)
				c.Abort()
				return
			}
			id = parsed
		} else {
			response.SendError(c, "Invalid user ID format", http.StatusUnauthorized)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			response.SendError(c, "User not found", http.StatusUnauthorized)
			c.Abort()
			return
		}
		if user.Role == "employee" {
			response.SendError(c, "Forbidden: Employees cannot perform this action", http.StatusForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireResource checks if the user has access to a specific resource via their departments
// Admins always have access to all resources
func RequireResource(userRepo repository.UserRepository, deptRepo repository.DepartmentRepository, requiredResource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("userID")
		if !ok {
			response.SendError(c, "User not authenticated", http.StatusUnauthorized)
			c.Abort()
			return
		}

		var id uuid.UUID
		if u, isUUID := userIDVal.(uuid.UUID); isUUID {
			id = u
		} else if userIDStr, ok := userIDVal.(string); ok {
			parsed, err := uuid.Parse(userIDStr)
			if err != nil {
				response.SendError(c, "Invalid user ID in token", http.StatusUnauthorized)
				c.Abort()
				return
			}
			id = parsed
		} else {
			response.SendError(c, "Invalid user ID format", http.StatusUnauthorized)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			response.SendError(c, "User not found", http.StatusUnauthorized)
			c.Abort()
			return
		}

		// Admins have access to all resources
		if user.Role == "admin" {
			c.Next()
			return
		}

		// Get user's departments
		departments, err := deptRepo.GetUserDepartments(c.Request.Context(), id)
		if err != nil {
			response.SendError(c, "Failed to check resource access", http.StatusInternalServerError)
			c.Abort()
			return
		}

		// Check if any department has the required resource
		hasAccess := false
		for _, dept := range departments {
			// Parse resources from JSONB
			var resources []string
			if len(dept.Resources) > 0 {
				if err := json.Unmarshal(dept.Resources, &resources); err == nil {
					for _, resource := range resources {
						if resource == requiredResource {
							hasAccess = true
							break
						}
					}
				}
			}
			if hasAccess {
				break
			}
		}

		if !hasAccess {
			response.SendError(c, "Forbidden: You do not have access to this resource", http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireActiveUser ensures the current authenticated user has active status
func RequireActiveUser(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("userID")
		if !ok {
			response.SendError(c, "User not authenticated", http.StatusUnauthorized)
			c.Abort()
			return
		}

		var id uuid.UUID
		if u, isUUID := userIDVal.(uuid.UUID); isUUID {
			id = u
		} else if userIDStr, ok := userIDVal.(string); ok {
			parsed, err := uuid.Parse(userIDStr)
			if err != nil {
				response.SendError(c, "Invalid user ID in token", http.StatusUnauthorized)
				c.Abort()
				return
			}
			id = parsed
		} else {
			response.SendError(c, "Invalid user ID format", http.StatusUnauthorized)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			response.SendError(c, "User not found", http.StatusUnauthorized)
			c.Abort()
			return
		}

		if user.AccountStatus != "active" {
			response.SendError(c, "Your account has been deactivated. Please contact an administrator", http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
