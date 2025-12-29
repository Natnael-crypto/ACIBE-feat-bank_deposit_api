package department

import (
	"net/http"
	"strconv"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/core/services/department"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DepartmentHandler struct {
	service department.Service
}

func NewDepartmentHandler(service department.Service) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

type CreateDepartmentRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Resources   []string `json:"resources"`
}

type UpdateDepartmentRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Resources   *[]string `json:"resources"`
}

// GetAllDepartments godoc
// @Summary      Get All Departments
// @Description  Retrieve all departments without pagination
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Success      200  {array}   department.DepartmentResponse
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments/all [get]
func (h *DepartmentHandler) GetAllDepartments(c *gin.Context) {
	departments, err := h.service.GetAllDepartments(c.Request.Context())
	if err != nil {
		response.SendInternalError(c, "Failed to fetch departments")
		return
	}

	// Transform departments to frontend format
	deptResponses := ToDepartmentResponseList(departments)
	response.SendSuccess(c, "Departments retrieved successfully", deptResponses)
}

// GetDepartmentsWithPagination godoc
// @Summary      Get Departments with Pagination
// @Description  Retrieve a paginated list of departments with optional search
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "Page number" default(1)
// @Param        limit   query     int     false  "Items per page" default(10)
// @Param        search  query     string  false  "Search query"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments [get]
func (h *DepartmentHandler) GetDepartmentsWithPagination(c *gin.Context) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			response.SendInternalError(c, "Internal server error occurred")
			return
		}
	}()

	page := 1
	limit := 10
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	search := c.Query("search")

	departments, err := h.service.GetDepartmentsWithPagination(c.Request.Context(), page, limit, search)
	if err != nil {
		response.SendInternalError(c, "Failed to fetch departments")
		return
	}

	// Handle nil response (shouldn't happen, but be safe)
	if departments == nil {
		response.SendSuccess(c, "Departments retrieved successfully", gin.H{
			"data": []interface{}{},
			"pagination": gin.H{
				"currentPage":  1,
				"pageSize":    limit,
				"firstPage":   1,
				"lastPage":    1,
				"totalRecords": 0,
			},
		})
		return
	}

	// Ensure Data is not nil (service should handle this, but be safe)
	if departments.Data == nil {
		departments.Data = []dbgen.DepartmentDepartment{}
	}

	// Transform departments to frontend format - handle potential panics
	var deptResponses []*DepartmentResponse
	if len(departments.Data) > 0 {
		deptResponses = ToDepartmentResponseList(departments.Data)
	} else {
		deptResponses = []*DepartmentResponse{}
	}

	// Return transformed response matching frontend expectations
	response.SendSuccess(c, "Departments retrieved successfully", gin.H{
		"data": deptResponses,
		"pagination": gin.H{
			"currentPage":  departments.Metadata.CurrentPage,
			"pageSize":    departments.Metadata.PageSize,
			"firstPage":   departments.Metadata.FirstPage,
			"lastPage":    departments.Metadata.LastPage,
			"totalRecords": departments.Metadata.TotalRecords,
		},
	})
}

// GetDepartment godoc
// @Summary      Get Department by ID
// @Description  Retrieve a specific department by its ID
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Department ID"
// @Success      200  {object}  department.DepartmentResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments/{id} [get]
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid department ID format")
		return
	}
	dept, err := h.service.GetDepartmentByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrDepartmentNotFound {
			response.SendNotFound(c, "Department not found")
			return
		}
		response.SendInternalError(c, "Failed to fetch department")
		return
	}

	// Transform department to frontend format
	deptResponse := ToDepartmentResponse(dept)
	response.SendSuccess(c, "Department retrieved successfully", deptResponse)
}

// CreateDepartment godoc
// @Summary      Create Department
// @Description  Create a new department
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        department  body      CreateDepartmentRequest  true  "Department to create"
// @Success      201  {object}  department.DepartmentResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments [post]
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request data")
		return
	}
	dept, err := h.service.CreateDepartment(c.Request.Context(), &department.CreateDepartmentRequest{
		Name:        req.Name,
		Description: req.Description,
		Resources:   req.Resources,
	})
	if err != nil {
		response.SendInternalError(c, "Failed to create department")
		return
	}

	// Transform department to frontend format
	deptResponse := ToDepartmentResponse(dept)
	response.SendSuccess(c, "Department created successfully", deptResponse, http.StatusCreated)
}

// UpdateDepartment godoc
// @Summary      Update Department
// @Description  Update an existing department by ID
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        id          path      string                 true  "Department ID"
// @Param        department  body      UpdateDepartmentRequest  true  "Department update data"
// @Success      200  {object}  department.DepartmentResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid department ID format")
		return
	}
	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "Invalid request data")
		return
	}
	dept, err := h.service.UpdateDepartment(c.Request.Context(), id, &department.UpdateDepartmentRequest{
		Name:        req.Name,
		Description: req.Description,
		Resources:   req.Resources,
	})
	if err != nil {
		if err == repository.ErrDepartmentNotFound {
			response.SendNotFound(c, "Department not found")
			return
		}
		response.SendInternalError(c, "Failed to update department")
		return
	}

	// Transform department to frontend format
	deptResponse := ToDepartmentResponse(dept)
	response.SendSuccess(c, "Department updated successfully", deptResponse)
}

// DeleteDepartment godoc
// @Summary      Delete Department
// @Description  Delete a department by ID
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Department ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "Invalid department ID format")
		return
	}
	if err := h.service.DeleteDepartment(c.Request.Context(), id); err != nil {
		if err == repository.ErrDepartmentNotFound {
			response.SendNotFound(c, "Department not found")
			return
		}
		response.SendInternalError(c, "Failed to delete department")
		return
	}
	response.SendSuccess(c, "Department deleted successfully", nil)
}

