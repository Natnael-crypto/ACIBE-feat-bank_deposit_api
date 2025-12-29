package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/core/services/user"
	"github.com/aci/backend/internal/repository"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	conn, close, err := repository.OpenDB(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer close()

	store := repository.NewStore(conn)
	userRepo := repository.NewUserRepository(store)

	ctx := context.Background()

	// Seed admin user using the service function
	if err := user.SeedAdminUser(ctx, userRepo, cfg); err != nil {
		slog.Error("Failed to seed admin user", "error", err)
		os.Exit(1)
	}

	slog.Info("Seeding completed successfully")
}

