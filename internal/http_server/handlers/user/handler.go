package user

import (
	"net/http"
	"strconv"
	"strings"

	dbgen "github.com/aci/backend/gen/db"
	domain "github.com/aci/backend/internal/core/domain/common"
	userservice "github.com/aci/backend/internal/core/services/user"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService     userservice.Service
	departmentRepo  repository.DepartmentRepository
	userRepo        repository.UserRepository
}

func NewUserHandler(userService userservice.Service, departmentRepo repository.DepartmentRepository, userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userService:    userService,
		departmentRepo: departmentRepo,
		userRepo:       userRepo,
	}
}

// checkIfEmployeeModifyingAdmin checks if an employee is trying to modify an admin user
// Returns true if the operation should be blocked, false if allowed
func (h *UserHandler) checkIfEmployeeModifyingAdmin(c *gin.Context, targetUserID uuid.UUID) bool {
	// Get current user's role from context
	currentRoleVal, exists := c.Get("role")
	if !exists {
		return false // If no role in context, let other middleware handle it
	}
	
	currentRole, ok := currentRoleVal.(string)
	if !ok || currentRole == "admin" {
		return false // Admins can modify anyone
	}
	
	// If current user is employee, check if target user is admin
	if currentRole == "employee" {
		targetUser, err := h.userRepo.GetByID(c.Request.Context(), targetUserID)
		if err != nil {
			return false // If user not found, let the handler return appropriate error
		}
		
		if targetUser.Role == "admin" {
			return true // Employee trying to modify admin - block it
		}
	}
	
	return false // Allow the operation
}

type CreateUserRequest struct {
	FirstName     string   `json:"first_name" binding:"required"`
	LastName      string   `json:"last_name" binding:"required"`
	Email         string   `json:"email" binding:"required,email"`
	Password      string   `json:"password" binding:"required,min=6"`
	Role          string   `json:"role" binding:"required,oneof=admin employee"`
	DepartmentIDs []string `json:"department_ids"` // Accept as strings, convert to UUIDs
}

type UpdateUserRequest struct {
	FirstName     *string  `json:"first_name"`
	LastName      *string  `json:"last_name"`
	Email         *string  `json:"email" binding:"omitempty,email"`
	Role          *string  `json:"role" binding:"omitempty,oneof=admin employee"`
	DepartmentIDs *[]string `json:"department_ids"` // Accept as strings, convert to UUIDs
}

type UpdateStatusRequest struct {
	Status interface{} `json:"status"` // Accept any type, we'll convert to bool
}

type ChangePasswordRequest struct {
	CurrentPassword *string `json:"current_password,omitempty"`
	NewPassword     string  `json:"new_password" binding:"required,min=6"`
}

// GetUsers godoc
// @Summary      Get Users
// @Description  Retrieve a paginated list of users with optional search and sorting
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        page        query     int     false  "Page number" default(1)
// @Param        limit       query     int     false  "Items per page" default(10)
// @Param        sort_field  query     string  false  "Field to sort by" default(created_at)
// @Param        sort_order  query     string  false  "Sort order (asc/desc)" default(desc)
// @Param        search      query     string  false  "Search query"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			response.SendInternalError(c, "Internal server error occurred")
			return
		}
	}()

	// Parse query parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	sortField := c.DefaultQuery("sort_field", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	search := c.Query("search") // Optional search parameter

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	users, err := h.userService.GetAllUsersPaginated(c.Request.Context(), page, limit, sortField, sortOrder, search)
	if err != nil {
		// If search returns no results, return empty response instead of error
		if err == repository.ErrUserNotFound && search != "" {
			offset := (page - 1) * limit
			metadata := domain.CalculateMetadata(0, offset, limit)
			response.SendSuccess(c, "No users found matching your search", gin.H{
				"data": []*UserResponse{},
				"pagination": gin.H{
					"currentPage":  metadata.CurrentPage,
					"pageSize":    metadata.PageSize,
					"firstPage":   metadata.FirstPage,
					"lastPage":    metadata.LastPage,
					"totalRecords": metadata.TotalRecords,
				},
			})
			return
		}
		response.SendInternalError(c, "Failed to fetch users")
		return
	}

	// Handle nil response
	if users == nil {
		offset := (page - 1) * limit
		metadata := domain.CalculateMetadata(0, offset, limit)
		response.SendSuccess(c, "Users retrieved successfully", gin.H{
			"data": []*UserResponse{},
			"pagination": gin.H{
				"currentPage":  metadata.CurrentPage,
				"pageSize":    metadata.PageSize,
				"firstPage":   metadata.FirstPage,
				"lastPage":    metadata.LastPage,
				"totalRecords": metadata.TotalRecords,
			},
		})
		return
	}

	// Ensure Data is not nil
	if users.Data == nil {
		users.Data = []dbgen.AccountUser{}
	}

	// Transform users to frontend format (with departments)
	userResponses := ToUserResponseList(users.Data, h.departmentRepo)

	// Return transformed response matching frontend expectations
	response.SendSuccess(c, "Users retrieved successfully", gin.H{
		"data": userResponses,
		"pagination": gin.H{
			"currentPage":  users.Metadata.CurrentPage,
			"pageSize":    users.Metadata.PageSize,
			"firstPage":   users.Metadata.FirstPage,
			"lastPage":    users.Metadata.LastPage,
			"totalRecords": users.Metadata.TotalRecords,
		},
	})
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	query := c.Query("search")
	if query == "" {
		response.SendValidationError(c, "Search query is required")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	users, err := h.userService.SearchUsers(c.Request.Context(), query, page, limit)
	if err != nil {
		if err == repository.ErrUserNotFound {
			// Calculate metadata for empty results
			offset := (page - 1) * limit
			metadata := domain.CalculateMetadata(0, offset, limit)
			// Return empty results in frontend format
			response.SendSuccess(c, "No users found matching your search", gin.H{
				"data": []interface{}{},
				"pagination": gin.H{
					"currentPage":  metadata.CurrentPage,
					"pageSize":    metadata.PageSize,
					"firstPage":   metadata.FirstPage,
					"lastPage":    metadata.LastPage,
					"totalRecords": metadata.TotalRecords,
				},
			})
			return
		}
		response.SendInternalError(c, "Failed to search users")
		return
	}

	// Transform users to frontend format (with departments)
	userResponses := ToUserResponseList(users.Data, h.departmentRepo)

	// Return transformed response matching frontend expectations
	response.SendSuccess(c, "Search results retrieved successfully", gin.H{
		"data": userResponses,
		"pagination": gin.H{
			"currentPage":  users.Metadata.CurrentPage,
			"pageSize":    users.Metadata.PageSize,
			"firstPage":   users.Metadata.FirstPage,
			"lastPage":    users.Metadata.LastPage,
			"totalRecords": users.Metadata.TotalRecords,
		},
	})
}

// GetUser godoc
// @Summary      Get User by ID
// @Description  Retrieve a specific user by their ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  user.UserResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid user ID format")
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			response.SendNotFound(c, "User not found")
			return
		}
		response.SendInternalError(c, "Failed to fetch user")
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := ToUserResponse(user, h.departmentRepo)
	response.SendSuccess(c, "User retrieved successfully", userResponse)
}

// CreateUser godoc
// @Summary      Create User
// @Description  Create a new user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      CreateUserRequest  true  "User to create"
// @Success      201  {object}  user.UserResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, formatValidationError(err))
		return
	}

	// Convert department ID strings to UUIDs
	// Frontend sends department_ids as string array (e.g., ["uuid1", "uuid2"])
	var departmentUUIDs []uuid.UUID
	if len(req.DepartmentIDs) > 0 {
		departmentUUIDs = make([]uuid.UUID, 0, len(req.DepartmentIDs))
		for _, deptIDStr := range req.DepartmentIDs {
			if deptIDStr == "" {
				continue // Skip empty strings
			}
			deptID, err := uuid.Parse(deptIDStr)
			if err != nil {
				response.SendValidationError(c, "Invalid department ID format: "+deptIDStr)
				return
			}
			departmentUUIDs = append(departmentUUIDs, deptID)
		}
	}

	user, err := h.userService.CreateUser(c.Request.Context(), &userservice.CreateUserRequest{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Email:         req.Email,
		Password:      req.Password,
		Role:          req.Role,
		DepartmentIDs: departmentUUIDs,
	})
	if err != nil {
		if err == repository.ErrUserExists {
			response.SendConflict(c, "User with this email already exists")
			return
		}
		response.SendInternalError(c, "Failed to create user")
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := ToUserResponse(user, h.departmentRepo)
	response.SendSuccess(c, "User created successfully", userResponse, http.StatusCreated)
}

// UpdateUser godoc
// @Summary      Update User
// @Description  Update an existing user by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "User ID"
// @Param        user  body      UpdateUserRequest  true  "User update data"
// @Success      200  {object}  user.UserResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid user ID format")
		return
	}

	// Check if employee is trying to modify admin user
	if h.checkIfEmployeeModifyingAdmin(c, id) {
		response.SendError(c, "Forbidden: Employees cannot modify admin users", http.StatusForbidden)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, formatValidationError(err))
		return
	}

	// Convert department ID strings to UUIDs if provided
	// Note: If department_ids is provided (even as empty array []), we update departments
	// If department_ids is nil/not provided, we don't change departments
	var departmentUUIDs *[]uuid.UUID
	if req.DepartmentIDs != nil {
		converted := make([]uuid.UUID, 0, len(*req.DepartmentIDs))
		for _, deptIDStr := range *req.DepartmentIDs {
			if deptIDStr == "" {
				continue // Skip empty strings
			}
			deptID, err := uuid.Parse(deptIDStr)
			if err != nil {
				response.SendValidationError(c, "Invalid department ID format: "+deptIDStr)
				return
			}
			converted = append(converted, deptID)
		}
		// Always set departmentUUIDs if DepartmentIDs was provided (even if empty array)
		// Empty array means clear all departments
		departmentUUIDs = &converted
	}

	// Get current user's role from context
	currentRoleVal, exists := c.Get("role")
	currentRole := ""
	if exists {
		currentRole, _ = currentRoleVal.(string)
	}

	// Prevent employees from changing roles (especially setting to admin)
	if currentRole == "employee" && req.Role != nil {
		// Get target user to check current role
		targetUser, err := h.userRepo.GetByID(c.Request.Context(), id)
		if err == nil {
			// If trying to change role to admin, block it
			if *req.Role == "admin" {
				response.SendError(c, "Forbidden: Employees cannot change user role to admin", http.StatusForbidden)
				return
			}
			// If target user is admin and trying to change their role, block it
			if targetUser.Role == "admin" && *req.Role != "admin" {
				response.SendError(c, "Forbidden: Employees cannot change admin user's role", http.StatusForbidden)
				return
			}
		}
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, &userservice.UpdateUserRequest{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Email:         req.Email,
		Role:          req.Role,
		DepartmentIDs: departmentUUIDs,
	})
	if err != nil {
		if err == repository.ErrUserNotFound {
			response.SendNotFound(c, "User not found")
			return
		}
		if err == repository.ErrUserExists {
			response.SendConflict(c, "User with this email already exists")
			return
		}
		response.SendInternalError(c, "Failed to update user")
		return
	}

	// Transform user to frontend format (with departments)
	userResponse := ToUserResponse(user, h.departmentRepo)
	response.SendSuccess(c, "User updated successfully", userResponse)
}

// UpdateUserStatus godoc
// @Summary      Update User Status
// @Description  Update the status (active/inactive) of a user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id      path      string                true  "User ID"
// @Param        status  body      UpdateStatusRequest   true  "Status update"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users/{id}/status [put]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid user ID format")
		return
	}

	// Check if employee is trying to modify admin user
	if h.checkIfEmployeeModifyingAdmin(c, id) {
		response.SendError(c, "Forbidden: Employees cannot modify admin users", http.StatusForbidden)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	// Convert status to boolean, handling multiple input types
	var statusBool bool
	if req.Status == nil {
		response.SendValidationError(c, "Status field is required")
		return
	}

	switch v := req.Status.(type) {
	case bool:
		statusBool = v
	case string:
		statusBool = (v == "active" || v == "true" || v == "1")
	case float64:
		// JSON numbers are parsed as float64
		statusBool = (v != 0)
	default:
		response.SendValidationError(c, "Status must be a boolean value")
		return
	}

	// Convert boolean status to string format expected by service
	statusStr := "inactive"
	if statusBool {
		statusStr = "active"
	}

	if err := h.userService.UpdateUserStatus(c.Request.Context(), id, statusStr); err != nil {
		if err == repository.ErrUserNotFound {
			response.SendNotFound(c, "User not found")
			return
		}
		response.SendInternalError(c, "Failed to update user status")
		return
	}

	response.SendSuccess(c, "User status updated successfully", nil)
}

// ChangePassword godoc
// @Summary      Change User Password
// @Description  Change the password for a user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id         path      string                  true  "User ID"
// @Param        password   body      ChangePasswordRequest   true  "Password change data"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users/{id}/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid user ID format")
		return
	}

	// Check if employee is trying to modify admin user
	if h.checkIfEmployeeModifyingAdmin(c, id) {
		response.SendError(c, "Forbidden: Employees cannot modify admin users", http.StatusForbidden)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, formatValidationError(err))
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), id, &userservice.ChangePasswordRequest{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		if err == repository.ErrUserNotFound {
			response.SendNotFound(c, "User not found")
			return
		}
		response.SendValidationError(c, "Current password is incorrect")
		return
	}

	response.SendSuccess(c, "Password changed successfully", nil)
}

// DeleteUser godoc
// @Summary      Delete User
// @Description  Delete a user by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid user ID format")
		return
	}

	// Check if employee is trying to modify admin user
	if h.checkIfEmployeeModifyingAdmin(c, id) {
		response.SendError(c, "Forbidden: Employees cannot delete admin users", http.StatusForbidden)
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), id); err != nil {
		if err == repository.ErrUserNotFound {
			response.SendNotFound(c, "User not found")
			return
		}
		response.SendInternalError(c, "Failed to delete user")
		return
	}

	response.SendSuccess(c, "User deleted successfully", nil)
}

func formatValidationError(err error) string {
	errStr := err.Error()

	// Handle specific validation errors
	if strings.Contains(errStr, "Email") && strings.Contains(errStr, "email") {
		return "Please provide a valid email address (e.g., user@example.com)"
	}
	if strings.Contains(errStr, "FirstName") && strings.Contains(errStr, "required") {
		return "First name is required"
	}
	if strings.Contains(errStr, "LastName") && strings.Contains(errStr, "required") {
		return "Last name is required"
	}
	if strings.Contains(errStr, "Password") && strings.Contains(errStr, "min") {
		return "Password must be at least 6 characters long"
	}
	if strings.Contains(errStr, "Password") && strings.Contains(errStr, "required") {
		return "Password is required"
	}
	if strings.Contains(errStr, "Role") && strings.Contains(errStr, "oneof") {
		return "Role must be one of: admin, employee"
	}

	// Default fallback
	return "Please check your input and try again"
}

