.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest || { \
			echo "Warning: Failed to install swag. Skipping Swagger generation."; \
			exit 0; \
		}; \
	fi
	@export PATH=$$PATH:$$(go env GOPATH)/bin && swag init -g cmd/main.go -o docs --parseDependency --parseInternal || { \
		echo "Warning: Swagger generation failed. Continuing anyway..."; \
		exit 0; \
	}
	@echo "Swagger documentation generated successfully!"

.PHONY: air
air:
	@echo "Running air"
	@air -c .air.toml

.PHONY: build
build: swagger
	@echo "Building Go application..."
	@go build -o bin/aci ./cmd/main.go
	@echo "Build completed successfully! Binary: bin/aci"

.PHONY: seed
seed:
	@echo "Seeding admin user..."
	@go run ./cmd/seed
	@echo "Seeding completed!"

.PHONY: run
run:
	@echo "Running Go application"
	@go run ./cmd/main.go
	
.PHONY: migrate-create
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: Migration name is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir db/migrations -seq $(NAME)
	@echo "Created migration files for: $(NAME)"

.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations..."
	@go run ./cmd/migrate
	@echo "Migrations completed successfully!"

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back database migrations..."
	@sudo -u postgres psql -d aci_db -f db/migrations/000001_account.down.sql
	@echo "Migrations rolled back successfully!"

.PHONY: sqlc-generate
sqlc-generate:
	@echo "Generating Go code from SQL queries..."
	@sqlc generate
	@echo "Code generation completed successfully!"

.PHONY: sync
sync:
	@echo "Running all sync jobs (claims, payments, insurance)..."
	@go run ./cmd/sync -all

.PHONY: sync-claims
sync-claims:
	@echo "Syncing new claims..."
	@go run ./cmd/sync -claims

.PHONY: sync-payments
sync-payments:
	@echo "Syncing new payments..."
	@go run ./cmd/sync -payments

.PHONY: sync-insurance
sync-insurance:
	@echo "Syncing insurance checks..."
	@go run ./cmd/sync -insurance