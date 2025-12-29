package repository

import (
	"context"
	"errors"
	"strings"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*dbgen.AccountUser, error)
	GetByEmail(ctx context.Context, email string) (*dbgen.AccountUser, error)
	GetByGoogleID(ctx context.Context, googleID string) (*dbgen.AccountUser, error)
	GetByMicrosoftID(ctx context.Context, microsoftID string) (*dbgen.AccountUser, error)
	Create(ctx context.Context, params dbgen.CreateUserParams) (*dbgen.AccountUser, error)
	Update(ctx context.Context, id uuid.UUID, params dbgen.UpdateUserParams) (*dbgen.AccountUser, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, password []byte) (*dbgen.AccountUser, error)
	UpdateOAuthIDs(ctx context.Context, id uuid.UUID, googleID, microsoftID *string) (*dbgen.AccountUser, error)
	GetAllPaginated(ctx context.Context, sortOrder string, limit, offset int32) ([]dbgen.GetAllUsersRow, error)
	SearchUsers(ctx context.Context, query string, limit, offset int32) ([]dbgen.SearchUsersRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ReplaceDepartments(ctx context.Context, userID uuid.UUID, departmentIDs []uuid.UUID) error
}

type userRepository struct {
	store *Store
}

func NewUserRepository(store *Store) UserRepository {
	return &userRepository{store: store}
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*dbgen.AccountUser, error) {
	row, err := r.store.queries.GetUserById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return rowToAccountUser(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*dbgen.AccountUser, error) {
	row, err := r.store.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return rowToAccountUserFromEmailRow(row), nil
}

func (r *userRepository) GetByGoogleID(ctx context.Context, googleID string) (*dbgen.AccountUser, error) {
	row, err := r.store.queries.GetUserByGoogleID(ctx, pgtype.Text{String: googleID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return rowToAccountUserFromGoogleIDRow(row), nil
}

func (r *userRepository) GetByMicrosoftID(ctx context.Context, microsoftID string) (*dbgen.AccountUser, error) {
	// Microsoft IDs can be stored in different formats, try exact match first
	row, err := r.store.queries.GetUserByMicrosoftID(ctx, pgtype.Text{String: microsoftID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Try with last 16 characters if the ID is longer
			if len(microsoftID) > 16 {
				last16 := microsoftID[len(microsoftID)-16:]
				row, err = r.store.queries.GetUserByMicrosoftID(ctx, pgtype.Text{String: last16, Valid: true})
				if err != nil {
					return nil, ErrUserNotFound
				}
			} else {
				return nil, ErrUserNotFound
			}
		} else {
			return nil, err
		}
	}
	return rowToAccountUserFromMicrosoftIDRow(row), nil
}

func (r *userRepository) Create(ctx context.Context, params dbgen.CreateUserParams) (*dbgen.AccountUser, error) {
	row, err := r.store.queries.CreateUser(ctx, params)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return rowToAccountUserFromCreateRow(row), nil
}

func (r *userRepository) Update(ctx context.Context, id uuid.UUID, params dbgen.UpdateUserParams) (*dbgen.AccountUser, error) {
	params.ID = id
	row, err := r.store.queries.UpdateUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return rowToAccountUserFromUpdateRow(row), nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id uuid.UUID, password []byte) (*dbgen.AccountUser, error) {
	row, err := r.store.queries.UpdatePassword(ctx, dbgen.UpdatePasswordParams{
		ID:           id,
		UserPassword: password,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return rowToAccountUserFromUpdatePasswordRow(row), nil
}

func (r *userRepository) UpdateOAuthIDs(ctx context.Context, id uuid.UUID, googleID, microsoftID *string) (*dbgen.AccountUser, error) {
	googleIDText := pgtype.Text{Valid: googleID != nil, String: ""}
	if googleID != nil {
		googleIDText.String = *googleID
	}
	microsoftIDText := pgtype.Text{Valid: microsoftID != nil, String: ""}
	if microsoftID != nil {
		microsoftIDText.String = *microsoftID
	}

	row, err := r.store.queries.UpdateOAuthIDs(ctx, dbgen.UpdateOAuthIDsParams{
		ID:          id,
		GoogleID:    googleIDText,
		MicrosoftID: microsoftIDText,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return rowToAccountUserFromUpdateOAuthIDsRow(row), nil
}

func (r *userRepository) GetAllPaginated(ctx context.Context, sortOrder string, limit, offset int32) ([]dbgen.GetAllUsersRow, error) {
	if sortOrder == "" {
		sortOrder = "desc"
	}
	users, err := r.store.queries.GetAllUsers(ctx, dbgen.GetAllUsersParams{
		Column1: sortOrder,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]dbgen.SearchUsersRow, error) {
	users, err := r.store.queries.SearchUsers(ctx, dbgen.SearchUsersParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First verify user exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete related tokens
	_ = r.store.queries.MarkUserTokensAsRevoked(ctx, id)

	// Note: We don't have a DeleteUser query yet, so we'll need to add it
	// For now, we can mark the user as inactive or add a delete query
	// Let's add a soft delete by updating status
	_, err = r.Update(ctx, id, dbgen.UpdateUserParams{
		AccountStatus: pgtype.Text{String: "deleted", Valid: true},
	})
	return err
}

func (r *userRepository) ReplaceDepartments(ctx context.Context, userID uuid.UUID, departmentIDs []uuid.UUID) error {
	// Delete existing associations
	if err := r.store.queries.ReplaceUserDepartments(ctx, userID); err != nil {
		return err
	}

	// Add new associations if any
	if len(departmentIDs) > 0 {
		// Add departments one by one
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

// Helper functions to convert Row types to AccountUser
func rowToAccountUser(row dbgen.GetUserByIdRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromEmailRow(row dbgen.GetUserByEmailRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromGoogleIDRow(row dbgen.GetUserByGoogleIDRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromMicrosoftIDRow(row dbgen.GetUserByMicrosoftIDRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromCreateRow(row dbgen.CreateUserRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromUpdateRow(row dbgen.UpdateUserRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromUpdatePasswordRow(row dbgen.UpdatePasswordRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func rowToAccountUserFromUpdateOAuthIDsRow(row dbgen.UpdateOAuthIDsRow) *dbgen.AccountUser {
	return &dbgen.AccountUser{
		ID:                row.ID,
		Fname:             row.Fname,
		Lname:             row.Lname,
		Email:             row.Email,
		UserPassword:      row.UserPassword,
		AccountStatus:     row.AccountStatus,
		EmailVerified:     row.EmailVerified,
		Role:              row.Role,
		ProfilePictureUrl: row.ProfilePictureUrl,
		VerifiedAt:        row.VerifiedAt,
		GoogleID:          row.GoogleID,
		MicrosoftID:       row.MicrosoftID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

