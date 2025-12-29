package department

import (
	"encoding/json"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/google/uuid"
)

// DepartmentResponse represents the department data format expected by the frontend
type DepartmentResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Resources   []string  `json:"resources"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// ToDepartmentResponse converts dbgen.DepartmentDepartment to DepartmentResponse format
func ToDepartmentResponse(dept *dbgen.DepartmentDepartment) *DepartmentResponse {
	// Handle nil department
	if dept == nil {
		return &DepartmentResponse{
			ID:          uuid.Nil,
			Name:        "",
			Description: "",
			Resources:   []string{},
			CreatedAt:   "",
			UpdatedAt:   "",
		}
	}

	// Parse resources from JSONB ([]byte) to []string
	var resources []string
	if len(dept.Resources) > 0 {
		if err := json.Unmarshal(dept.Resources, &resources); err != nil {
			// If parsing fails, return empty array
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
	createdAt := ""
	if dept.CreatedAt.Valid {
		createdAt = dept.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	updatedAt := ""
	if dept.UpdatedAt.Valid {
		updatedAt = dept.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &DepartmentResponse{
		ID:          dept.ID,
		Name:        dept.Name,
		Description: description,
		Resources:   resources,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ToDepartmentResponseList converts a slice of dbgen.DepartmentDepartment to DepartmentResponse
func ToDepartmentResponseList(departments []dbgen.DepartmentDepartment) []*DepartmentResponse {
	if departments == nil {
		return []*DepartmentResponse{}
	}
	result := make([]*DepartmentResponse, len(departments))
	for i := range departments {
		result[i] = ToDepartmentResponse(&departments[i])
	}
	return result
}

