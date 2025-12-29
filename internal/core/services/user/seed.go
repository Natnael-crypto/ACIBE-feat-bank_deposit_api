package user

import (
	"context"
	"log/slog"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/auth"
	"github.com/jackc/pgx/v5/pgtype"
)

// SeedAdminUser creates the admin user if it doesn't exist
func SeedAdminUser(ctx context.Context, userRepo repository.UserRepository, cfg config.Config) error {
	// Check if admin user already exists
	_, err := userRepo.GetByEmail(ctx, cfg.Admin.Username)
	if err == nil {
		// Admin user already exists
		slog.Info("Admin user already exists", "email", cfg.Admin.Username)
		return nil
	}

	// Admin user doesn't exist, create it
	slog.Info("Creating admin user", "email", cfg.Admin.Username)

	// Hash the password
	hashedPassword, err := auth.HashPassword(cfg.Admin.Password)
	if err != nil {
		return err
	}

	// Parse email to get first and last name
	firstName := "Admin"
	lastName := "User"

	// Create admin user directly using repository
	_, err = userRepo.Create(ctx, dbgen.CreateUserParams{
		Fname:             firstName,
		Lname:             lastName,
		Email:             cfg.Admin.Username,
		UserPassword:      []byte(hashedPassword),
		AccountStatus:     "active",
		EmailVerified:     true,
		Role:              "admin",
		VerifiedAt:        pgtype.Timestamptz{Valid: false},
		ProfilePictureUrl: pgtype.Text{Valid: false},
		GoogleID:          pgtype.Text{Valid: false},
		MicrosoftID:       pgtype.Text{Valid: false},
	})

	if err != nil {
		slog.Error("Failed to create admin user", "error", err)
		return err
	}

	slog.Info("Admin user created successfully", "email", cfg.Admin.Username)
	return nil
}

