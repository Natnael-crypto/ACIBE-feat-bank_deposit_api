package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/repository"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to database using the same method as the app
	conn, close, err := repository.OpenDB(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer close()

	slog.Info("Connected to database successfully")

	// List of migration files in order
	migrations := []string{
		"000001_account.up.sql",
		"000002_collaborate_schema.up.sql",
		"000003_add_oauth_to_user.up.sql",
		"000004_departments.up.sql",
		"000005_create_available_years.up.sql",
		"000006_create_dashboard_materialized_view.up.sql",
		"000007_add_performance_indexes.up.sql",
		"000008_create_yearly_dashboard_data.up.sql",
	}

	migrationDir := "db/migrations"

	for _, migration := range migrations {
		migrationPath := filepath.Join(migrationDir, migration)
		
		// Read migration file
		sqlBytes, err := os.ReadFile(migrationPath)
		if err != nil {
			slog.Error("Failed to read migration file", "file", migrationPath, "error", err)
			os.Exit(1)
		}

		slog.Info("Running migration", "file", migration)

		// Execute migration
		ctx := context.Background()
		if _, err := conn.Exec(ctx, string(sqlBytes)); err != nil {
			slog.Error("Failed to execute migration", "file", migration, "error", err)
			// Check if error is because schema/table already exists
			if contains(err.Error(), "already exists") {
				slog.Warn("Migration already applied, skipping", "file", migration)
				continue
			}
			os.Exit(1)
		}

		slog.Info("Migration completed", "file", migration)
	}

	slog.Info("All migrations completed successfully!")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

