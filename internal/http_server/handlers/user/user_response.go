package user

import (
	"context"
	"encoding/json"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/repository"
	"github.com/google/uuid"
)

// DepartmentInfo represents department data in user response
type DepartmentInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Resources   []string  `json:"resources"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// UserResponse represents the user data format expected by the frontend
type UserResponse struct {
	ID          uuid.UUID       `json:"id"`
	FirstName   string          `json:"first_name"`
	LastName    string          `json:"last_name"`
	Email       string          `json:"email"`
	Status      bool            `json:"status"`
	Role        string          `json:"role"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Departments []DepartmentInfo `json:"departments,omitempty"`
}

// ToUserResponse converts dbgen.AccountUser to UserResponse format
// If deptRepo is provided, it will fetch and include departments
func ToUserResponse(user *dbgen.AccountUser, deptRepo repository.DepartmentRepository) *UserResponse {
	status := user.AccountStatus == "active"
	
	createdAt := ""
	if user.CreatedAt.Valid {
		createdAt = user.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	
	updatedAt := ""
	if user.UpdatedAt.Valid {
		updatedAt = user.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	// Fetch departments if repository is provided
	var departments []DepartmentInfo
	if deptRepo != nil {
		ctx := context.Background()
		depts, err := deptRepo.GetUserDepartments(ctx, user.ID)
		if err == nil {
			departments = make([]DepartmentInfo, len(depts))
			for i, dept := range depts {
				// Parse resources from JSONB
				var resources []string
				if len(dept.Resources) > 0 {
					if err := json.Unmarshal(dept.Resources, &resources); err != nil {
						resources = []string{}
					}
				} else {
					resources = []string{}
				}

				// Format description
				description := ""
				if dept.Description.Valid {
					description = dept.Description.String
				}

				// Format timestamps
				deptCreatedAt := ""
				if dept.CreatedAt.Valid {
					deptCreatedAt = dept.CreatedAt.Time.Format(time.RFC3339)
				}

				deptUpdatedAt := ""
				if dept.UpdatedAt.Valid {
					deptUpdatedAt = dept.UpdatedAt.Time.Format(time.RFC3339)
				}

				departments[i] = DepartmentInfo{
					ID:          dept.ID,
					Name:        dept.Name,
					Description: description,
					Resources:   resources,
					CreatedAt:   deptCreatedAt,
					UpdatedAt:   deptUpdatedAt,
				}
			}
		}
	}

	return &UserResponse{
		ID:          user.ID,
		FirstName:   user.Fname,
		LastName:    user.Lname,
		Email:       user.Email,
		Status:      status,
		Role:        user.Role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Departments: departments,
	}
}

// ToUserResponseList converts a slice of dbgen.AccountUser to UserResponse
// If deptRepo is provided, it will fetch and include departments for each user
func ToUserResponseList(users []dbgen.AccountUser, deptRepo repository.DepartmentRepository) []*UserResponse {
	result := make([]*UserResponse, len(users))
	for i := range users {
		result[i] = ToUserResponse(&users[i], deptRepo)
	}
	return result
}

