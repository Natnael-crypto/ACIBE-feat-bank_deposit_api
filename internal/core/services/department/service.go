package department

import (
	"context"
	"encoding/json"

	dbgen "github.com/aci/backend/gen/db"
	domain "github.com/aci/backend/internal/core/domain/common"
	"github.com/aci/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	GetAllDepartments(ctx context.Context) ([]dbgen.DepartmentDepartment, error)
	GetDepartmentsWithPagination(ctx context.Context, page, limit int, search string) (*PaginatedDepartmentsResponse, error)
	GetDepartmentByID(ctx context.Context, id uuid.UUID) (*dbgen.DepartmentDepartment, error)
	CreateDepartment(ctx context.Context, req *CreateDepartmentRequest) (*dbgen.DepartmentDepartment, error)
	UpdateDepartment(ctx context.Context, id uuid.UUID, req *UpdateDepartmentRequest) (*dbgen.DepartmentDepartment, error)
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo repository.DepartmentRepository
}

func NewService(repo repository.DepartmentRepository) Service {
	return &service{repo: repo}
}

type PaginatedDepartmentsResponse struct {
	Data     []dbgen.DepartmentDepartment `json:"data"`
	Metadata domain.Metadata              `json:"metadata"`
}

type CreateDepartmentRequest struct {
	Name        string
	Description string
	Resources   []string
}

type UpdateDepartmentRequest struct {
	Name        *string
	Description *string
	Resources   *[]string
}

func (s *service) GetAllDepartments(ctx context.Context) ([]dbgen.DepartmentDepartment, error) {
	// Get all without pagination - use a large limit, no search
	rows, err := s.repo.GetAllPaginated(ctx, nil, 10000, 0)
	if err != nil {
		return nil, err
	}

	var departments []dbgen.DepartmentDepartment
	for _, row := range rows {
		departments = append(departments, dbgen.DepartmentDepartment{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Resources:   row.Resources,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return departments, nil
}

func (s *service) GetDepartmentsWithPagination(ctx context.Context, page, limit int, search string) (*PaginatedDepartmentsResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	
	// Use unified GetAllPaginated with optional search
	var searchPtr *string
	if search != "" {
		searchPtr = &search
	}
	
	rows, err := s.repo.GetAllPaginated(ctx, searchPtr, int32(limit), int32(offset))
	if err != nil {
		return nil, err
	}

	// Initialize empty slice if nil
	if rows == nil {
		rows = []dbgen.GetAllDepartmentsRow{}
	}

	var total int
	if len(rows) > 0 {
		total = int(rows[0].Count)
	}

	var departments []dbgen.DepartmentDepartment
	for _, row := range rows {
		departments = append(departments, dbgen.DepartmentDepartment{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Resources:   row.Resources,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	// Ensure departments is never nil
	if departments == nil {
		departments = []dbgen.DepartmentDepartment{}
	}

	metadata := domain.CalculateMetadata(total, offset, limit)

	return &PaginatedDepartmentsResponse{
		Data:     departments,
		Metadata: metadata,
	}, nil
}

func (s *service) GetDepartmentByID(ctx context.Context, id uuid.UUID) (*dbgen.DepartmentDepartment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) CreateDepartment(ctx context.Context, req *CreateDepartmentRequest) (*dbgen.DepartmentDepartment, error) {
	resourcesBytes, err := stringArrayToJSONB(req.Resources)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, dbgen.CreateDepartmentParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Resources:   resourcesBytes,
	})
}

func (s *service) UpdateDepartment(ctx context.Context, id uuid.UUID, req *UpdateDepartmentRequest) (*dbgen.DepartmentDepartment, error) {
	params := dbgen.UpdateDepartmentParams{
		ID: id,
	}

	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Resources != nil {
		resourcesBytes, err := stringArrayToJSONB(*req.Resources)
		if err != nil {
			return nil, err
		}
		params.Resources = resourcesBytes
	}

	return s.repo.Update(ctx, id, params)
}

func (s *service) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func stringArrayToJSONB(arr []string) ([]byte, error) {
	if arr == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(arr)
}

