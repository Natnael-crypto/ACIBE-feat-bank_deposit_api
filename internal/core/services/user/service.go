package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/config"
	domain "github.com/aci/backend/internal/core/domain/common"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/auth"
	"github.com/aci/backend/pkg/email"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	GetAllUsersPaginated(ctx context.Context, page, limit int, sortField, sortOrder, search string) (*PaginatedUsersResponse, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*dbgen.AccountUser, error)
	SearchUsers(ctx context.Context, query string, page, limit int) (*PaginatedUsersResponse, error)
	CreateUser(ctx context.Context, req *CreateUserRequest) (*dbgen.AccountUser, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*dbgen.AccountUser, error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error
	ChangePassword(ctx context.Context, id uuid.UUID, req *ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error
	ValidateResetToken(ctx context.Context, token string) error
	ResetPasswordWithToken(ctx context.Context, req *ResetPasswordWithTokenRequest) (*AuthResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
}

type service struct {
	userRepo  repository.UserRepository
	tokenRepo repository.PasswordResetTokenRepository
	emailSvc  email.EmailService
	cfg       config.Config
}

func NewService(userRepo repository.UserRepository, tokenRepo repository.PasswordResetTokenRepository, emailSvc email.EmailService, cfg config.Config) Service {
	return &service{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		emailSvc:  emailSvc,
		cfg:       cfg,
	}
}

type PaginatedUsersResponse struct {
	Data     []dbgen.AccountUser `json:"data"`
	Metadata domain.Metadata     `json:"metadata"`
}

type CreateUserRequest struct {
	FirstName     string
	LastName      string
	Email         string
	Password      string
	Role          string
	DepartmentIDs []uuid.UUID
}

type UpdateUserRequest struct {
	FirstName     *string
	LastName      *string
	Email         *string
	Role          *string
	DepartmentIDs *[]uuid.UUID
}

type ChangePasswordRequest struct {
	CurrentPassword *string
	NewPassword     string
}

type ForgotPasswordRequest struct {
	Email string
}

type ResetPasswordWithTokenRequest struct {
	Token       string
	NewPassword string
}

type LoginRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	User         *dbgen.AccountUser `json:"user"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	ExpiresIn    int64              `json:"expires_in"`
}

func (s *service) GetAllUsersPaginated(ctx context.Context, page, limit int, sortField, sortOrder, search string) (*PaginatedUsersResponse, error) {
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	offset := (page - 1) * limit
	var rows interface{}
	var total int
	var err error

	// If search query is provided, use search; otherwise use regular pagination
	if search != "" {
		searchRows, searchErr := s.userRepo.SearchUsers(ctx, search, int32(limit), int32(offset))
		err = searchErr
		rows = searchRows
		if err == nil && len(searchRows) > 0 {
			total = int(searchRows[0].Count)
		}
	} else {
		allRows, allErr := s.userRepo.GetAllPaginated(ctx, sortOrder, int32(limit), int32(offset))
		err = allErr
		rows = allRows
		if err == nil && len(allRows) > 0 {
			total = int(allRows[0].Count)
		}
	}

	if err != nil {
		return nil, err
	}

	// Handle nil rows
	if rows == nil {
		metadata := domain.CalculateMetadata(0, offset, limit)
		return &PaginatedUsersResponse{
			Data:     []dbgen.AccountUser{},
			Metadata: metadata,
		}, nil
	}

	var users []dbgen.AccountUser
	switch v := rows.(type) {
	case []dbgen.GetAllUsersRow:
		for _, row := range v {
			users = append(users, dbgen.AccountUser{
				ID:            row.ID,
				Fname:         row.Fname,
				Lname:         row.Lname,
				Email:         row.Email,
				UserPassword:  row.UserPassword,
				AccountStatus: row.AccountStatus,
				EmailVerified: row.EmailVerified,
				Role:          row.Role,
				ProfilePictureUrl: row.ProfilePictureUrl,
				VerifiedAt:    row.VerifiedAt,
				CreatedAt:     row.CreatedAt,
				UpdatedAt:     row.UpdatedAt,
			})
			if row.Count > 0 {
				total = int(row.Count)
			}
		}
	case []dbgen.SearchUsersRow:
		for _, row := range v {
			users = append(users, dbgen.AccountUser{
				ID:            row.ID,
				Fname:         row.Fname,
				Lname:         row.Lname,
				Email:         row.Email,
				UserPassword:  row.UserPassword,
				AccountStatus: row.AccountStatus,
				EmailVerified: row.EmailVerified,
				Role:          row.Role,
				ProfilePictureUrl: row.ProfilePictureUrl,
				VerifiedAt:    row.VerifiedAt,
				CreatedAt:     row.CreatedAt,
				UpdatedAt:     row.UpdatedAt,
			})
			if row.Count > 0 {
				total = int(row.Count)
			}
		}
	}

	// Ensure users is never nil
	if users == nil {
		users = []dbgen.AccountUser{}
	}

	metadata := domain.CalculateMetadata(total, offset, limit)

	return &PaginatedUsersResponse{
		Data:     users,
		Metadata: metadata,
	}, nil
}

func (s *service) GetUserByID(ctx context.Context, id uuid.UUID) (*dbgen.AccountUser, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *service) SearchUsers(ctx context.Context, query string, page, limit int) (*PaginatedUsersResponse, error) {
	if query == "" {
		return nil, repository.ErrUserNotFound
	}

	offset := (page - 1) * limit
	rows, err := s.userRepo.SearchUsers(ctx, query, int32(limit), int32(offset))
	if err != nil {
		return nil, err
	}

	var total int
	var users []dbgen.AccountUser
	for _, row := range rows {
		users = append(users, dbgen.AccountUser{
			ID:            row.ID,
			Fname:         row.Fname,
			Lname:         row.Lname,
			Email:         row.Email,
			UserPassword:  row.UserPassword,
			AccountStatus: row.AccountStatus,
			EmailVerified: row.EmailVerified,
			Role:          row.Role,
			ProfilePictureUrl: row.ProfilePictureUrl,
			VerifiedAt:    row.VerifiedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
		if row.Count > 0 {
			total = int(row.Count)
		}
	}

	// Ensure users is never nil
	if users == nil {
		users = []dbgen.AccountUser{}
	}

	metadata := domain.CalculateMetadata(total, offset, limit)

	return &PaginatedUsersResponse{
		Data:     users,
		Metadata: metadata,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, req *CreateUserRequest) (*dbgen.AccountUser, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, repository.ErrUserExists
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, dbgen.CreateUserParams{
		Fname:         req.FirstName,
		Lname:         req.LastName,
		Email:         req.Email,
		UserPassword:  []byte(hashedPassword),
		AccountStatus: "active",
		EmailVerified: false,
		Role:          req.Role,
		GoogleID:      pgtype.Text{Valid: false},
		MicrosoftID:   pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, err
	}

	// If department IDs provided, set associations
	if len(req.DepartmentIDs) > 0 {
		if err := s.userRepo.ReplaceDepartments(ctx, user.ID, req.DepartmentIDs); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *service) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*dbgen.AccountUser, error) {
	// Get existing user
	existingUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	params := dbgen.UpdateUserParams{
		ID: id,
	}

	if req.FirstName != nil {
		params.Fname = pgtype.Text{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		params.Lname = pgtype.Text{String: *req.LastName, Valid: true}
	}
	if req.Email != nil {
		// Check if new email already exists
		if *req.Email != existingUser.Email {
			_, err := s.userRepo.GetByEmail(ctx, *req.Email)
			if err == nil {
				return nil, repository.ErrUserExists
			}
		}
		params.Email = pgtype.Text{String: *req.Email, Valid: true}
	}
	if req.Role != nil {
		params.Role = pgtype.Text{String: *req.Role, Valid: true}
	}

	updatedUser, err := s.userRepo.Update(ctx, id, params)
	if err != nil {
		return nil, err
	}

	// Replace departments if provided (even if empty array - means clear all departments)
	if req.DepartmentIDs != nil {
		// ReplaceDepartments will delete all existing and add new ones
		// Empty array means delete all, non-empty means replace with new list
		if err := s.userRepo.ReplaceDepartments(ctx, id, *req.DepartmentIDs); err != nil {
			// If department update fails, we still return the updated user
			// but log the error - this allows partial updates
			// However, for data consistency, we should return the error
			return nil, err
		}
	}

	return updatedUser, nil
}

func (s *service) UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	params := dbgen.UpdateUserParams{
		ID:            id,
		AccountStatus: pgtype.Text{String: status, Valid: true},
	}
	_, err := s.userRepo.Update(ctx, id, params)
	return err
}

func (s *service) ChangePassword(ctx context.Context, id uuid.UUID, req *ChangePasswordRequest) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify current password only if provided
	if req.CurrentPassword != nil && *req.CurrentPassword != "" {
		if err := auth.VerifyPassword(string(user.UserPassword), *req.CurrentPassword); err != nil {
			return err
		}
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// Update password
	_, err = s.userRepo.UpdatePassword(ctx, id, []byte(hashedPassword))
	return err
}

func (s *service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	log.Printf("[FORGOT_PASSWORD] Processing request for email: %s", req.Email)

	// Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("[FORGOT_PASSWORD] User not found for email: %s", req.Email)
		return errors.New("user_not_found_silent")
	}

	// Check if user is active
	if user.AccountStatus != "active" {
		log.Printf("[FORGOT_PASSWORD] User found but inactive for email: %s", req.Email)
		return errors.New("user_not_found_silent")
	}

	log.Printf("[FORGOT_PASSWORD] User found: %s", user.Email)

	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("[FORGOT_PASSWORD] [ERROR] Failed to generate token: %v", err)
		return errors.New("failed to generate reset token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Delete any existing tokens for this user
	_ = s.tokenRepo.DeleteByUserID(ctx, user.ID)

	// Create new reset token (valid for 1 hour)
	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.tokenRepo.Create(ctx, user.ID, token, expiresAt); err != nil {
		log.Printf("[FORGOT_PASSWORD] [ERROR] Failed to create reset token: %v", err)
		return errors.New("failed to create reset token")
	}

	// Build reset link
	resetLink := s.cfg.FrontendURL + "/reset-password?token=" + token

	// Send email with reset link
	userName := user.Fname + " " + user.Lname

	if s.emailSvc == nil {
		log.Printf("[FORGOT_PASSWORD] [ERROR] Email service is nil!")
		return fmt.Errorf("email service not initialized")
	}

	if err := s.emailSvc.SendPasswordResetEmail(user.Email, resetLink, userName); err != nil {
		log.Printf("[FORGOT_PASSWORD] [ERROR] Failed to send email: %v", err)
		return fmt.Errorf("email sending failed: %w", err)
	}

	log.Printf("[FORGOT_PASSWORD] Password reset email sent successfully to %s", user.Email)
	return nil
}

func (s *service) ValidateResetToken(ctx context.Context, token string) error {
	_, err := s.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	return nil
}

func (s *service) ResetPasswordWithToken(ctx context.Context, req *ResetPasswordWithTokenRequest) (*AuthResponse, error) {
	// Find the token
	resetToken, err := s.tokenRepo.GetByToken(ctx, req.Token)
	if err != nil {
		return nil, errors.New("invalid or expired reset token")
	}

	// Get the user
	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Hash the new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Update the password
	if _, err := s.userRepo.UpdatePassword(ctx, user.ID, []byte(hashedPassword)); err != nil {
		return nil, errors.New("failed to update password")
	}

	// Mark token as used
	_ = s.tokenRepo.MarkAsUsed(ctx, req.Token)

	// Delete any other tokens for this user
	_ = s.tokenRepo.DeleteByUserID(ctx, user.ID)

	// Generate tokens for auto-login
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Email, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	response := &AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}

	return response, nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := auth.VerifyPassword(string(user.UserPassword), req.Password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if user.AccountStatus != "active" {
		return nil, errors.New("user account is inactive")
	}

	// Generate tokens
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Email, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	response := &AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}

	return response, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token and generate new tokens
	tokenPair, err := auth.RefreshToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	// Get user info from token claims
	claims, err := auth.ValidateToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user status is active
	if user.AccountStatus != "active" {
		return nil, errors.New("your account has been deactivated. Please contact an administrator")
	}

	response := &AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}

	return response, nil
}

