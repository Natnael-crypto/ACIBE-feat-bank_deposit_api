package repository

import (
	"context"
	"encoding/json"
	"errors"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrDepartmentNotFound = errors.New("department not found")
	ErrDepartmentExists   = errors.New("department already exists")
)

type DepartmentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*dbgen.DepartmentDepartment, error)
	GetAllPaginated(ctx context.Context, search *string, limit, offset int32) ([]dbgen.GetAllDepartmentsRow, error)
	Create(ctx context.Context, params dbgen.CreateDepartmentParams) (*dbgen.DepartmentDepartment, error)
	Update(ctx context.Context, id uuid.UUID, params dbgen.UpdateDepartmentParams) (*dbgen.DepartmentDepartment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetUserDepartments(ctx context.Context, userID uuid.UUID) ([]dbgen.DepartmentDepartment, error)
	ReplaceUserDepartments(ctx context.Context, userID uuid.UUID, departmentIDs []uuid.UUID) error
}

type departmentRepository struct {
	store *Store
}

func NewDepartmentRepository(store *Store) DepartmentRepository {
	return &departmentRepository{store: store}
}

func (r *departmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*dbgen.DepartmentDepartment, error) {
	dept, err := r.store.queries.GetDepartmentById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) GetAllPaginated(ctx context.Context, search *string, limit, offset int32) ([]dbgen.GetAllDepartmentsRow, error) {
	params := dbgen.GetAllDepartmentsParams{
		Limit:   limit,
		Offset:  offset,
		Column3: "", // Default to empty string (NULL handling is done in SQL)
	}
	// Add search parameter if provided
	if search != nil && *search != "" {
		params.Column3 = *search
	}
	
	result, err := r.store.queries.GetAllDepartments(ctx, params)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

func (r *departmentRepository) Create(ctx context.Context, params dbgen.CreateDepartmentParams) (*dbgen.DepartmentDepartment, error) {
	dept, err := r.store.queries.CreateDepartment(ctx, params)
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) Update(ctx context.Context, id uuid.UUID, params dbgen.UpdateDepartmentParams) (*dbgen.DepartmentDepartment, error) {
	params.ID = id
	dept, err := r.store.queries.UpdateDepartment(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.queries.DeleteDepartment(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *departmentRepository) GetUserDepartments(ctx context.Context, userID uuid.UUID) ([]dbgen.DepartmentDepartment, error) {
	return r.store.queries.GetUserDepartments(ctx, userID)
}

func (r *departmentRepository) ReplaceUserDepartments(ctx context.Context, userID uuid.UUID, departmentIDs []uuid.UUID) error {
	// Delete existing associations
	if err := r.store.queries.ReplaceUserDepartments(ctx, userID); err != nil {
		return err
	}

	// Add new associations if any
	if len(departmentIDs) > 0 {
		// Add departments one by one (sqlc doesn't handle array parameters well)
		for _, deptID := range departmentIDs {
			if err := r.store.queries.AddUserToDepartment(ctx, dbgen.AddUserToDepartmentParams{
				UserID:       userID,
				DepartmentID: deptID,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper function to convert string array to []byte (JSONB)
func StringArrayToJSONB(arr []string) ([]byte, error) {
	if arr == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(arr)
}

// Helper function to convert []byte (JSONB) to string array
func JSONBToStringArray(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

