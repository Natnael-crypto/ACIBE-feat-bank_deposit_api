package routes

import (
	"github.com/aci/backend/internal/core/services/auth"
	"github.com/aci/backend/internal/core/services/department"
	departmenthandler "github.com/aci/backend/internal/http_server/handlers/department"
	"github.com/aci/backend/internal/http_server/middlewares"
	"github.com/aci/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// SetupDepartmentRoutes configures all department management routes
func SetupDepartmentRoutes(
	router *gin.Engine,
	authService auth.Service,
	deptService department.Service,
	userRepo repository.UserRepository,
	deptRepo repository.DepartmentRepository,
) {
	api := router.Group("/api")
	protected := api.Group("")
	protected.Use(middlewares.RequireAuthN(authService))
	protected.Use(middlewares.RequireActiveUser(userRepo))

	{
		deptGroup := protected.Group("/departments")
		{
			deptHandler := departmenthandler.NewDepartmentHandler(deptService)

			// Read operations: available to all authenticated users
			deptGroup.GET("", deptHandler.GetDepartmentsWithPagination)
			deptGroup.GET("/all", deptHandler.GetAllDepartments)
			deptGroup.GET("/:id", deptHandler.GetDepartment)
			
			// Create, Update, Delete: require /departments resource access (admins always have access)
			deptGroup.POST("", middlewares.RequireResource(userRepo, deptRepo, "/departments"), deptHandler.CreateDepartment)
			deptGroup.PUT("/:id", middlewares.RequireResource(userRepo, deptRepo, "/departments"), deptHandler.UpdateDepartment)
			deptGroup.DELETE("/:id", middlewares.RequireResource(userRepo, deptRepo, "/departments"), deptHandler.DeleteDepartment)
		}
	}
}

