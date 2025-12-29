# Database Workflow Guide

This guide explains how to work with the database in this project: creating schemas, writing queries, and generating type-safe Go code using sqlc.

## Table of Contents

1. [Overview](#overview)
2. [Step 1: Create Database Schema (Migrations)](#step-1-create-database-schema-migrations)
3. [Step 2: Create SQL Queries](#step-2-create-sql-queries)
4. [Step 3: Update sqlc.yaml Configuration](#step-3-update-sqlcyaml-configuration)
5. [Step 4: Generate Type-Safe Go Code](#step-4-generate-type-safe-go-code)
6. [Complete Workflow Example](#complete-workflow-example)
7. [Commands Reference](#commands-reference)

---

## Overview

This project uses:
- **Migrations** (`db/migrations/`) - Database schema definitions
- **Queries** (`db/query/`) - SQL queries for database operations
- **sqlc** - Generates type-safe Go code from SQL queries
- **Generated Code** (`gen/db/`) - Auto-generated Go functions (DO NOT EDIT)

### Workflow Summary

```
1. Create Migration (Schema) → 2. Create Query File → 3. Update sqlc.yaml → 4. Generate Code
```

---

## Step 1: Create Database Schema (Migrations)

Migrations define your database structure (tables, schemas, indexes, etc.).

### Create Migration Files

```bash
make migrate-create NAME=your_migration_name
```

This creates two files:
- `00000X_your_migration_name.up.sql` - Apply migration
- `00000X_your_migration_name.down.sql` - Rollback migration

### Example: Creating a Department Schema

**File: `db/migrations/000003_departments.up.sql`**

```sql
-- Create schema
CREATE SCHEMA IF NOT EXISTS department;

-- Create table
CREATE TABLE IF NOT EXISTS department.department (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    resources JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create junction table for many-to-many relationship
CREATE TABLE IF NOT EXISTS department.user_department (
    user_id UUID NOT NULL REFERENCES account.user(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES department.department(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, department_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_department_user_id ON department.user_department(user_id);
CREATE INDEX IF NOT EXISTS idx_user_department_department_id ON department.user_department(department_id);
```

**File: `db/migrations/000003_departments.down.sql`**

```sql
-- Rollback: Drop tables and schema
DROP TABLE IF EXISTS department.user_department;
DROP TABLE IF EXISTS department.department;
DROP SCHEMA IF EXISTS department;
```

### Apply Migrations

```bash
make migrate-up
```

---

## Step 2: Create SQL Queries

After creating the schema, create SQL query files for database operations.

### Create Query Folder Structure

Organize queries by domain/feature in `db/query/`:

```
db/query/
  ├── account/
  ├── department/     ← Create this folder
  └── collaborate/
```

### Create Query File

**File: `db/query/department/department.sql`**

```sql
-- name: CreateDepartment :one
INSERT INTO department.department (
    name, description, resources
) VALUES (
    $1, $2, $3
)
RETURNING id, name, description, resources, created_at, updated_at;

-- name: GetDepartmentById :one
SELECT
    id, name, description, resources, created_at, updated_at
FROM
    department.department
WHERE
    id = $1;

-- name: GetAllDepartments :many
SELECT
    id, name, description, resources, created_at, updated_at,
    COUNT(*) OVER() AS count
FROM
    department.department
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateDepartment :one
UPDATE department.department
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    resources = COALESCE(sqlc.narg('resources'), resources),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, description, resources, created_at, updated_at;

-- name: DeleteDepartment :exec
DELETE FROM department.department
WHERE id = $1;
```

### Query Annotations Explained

- `-- name: FunctionName :one` - Returns a single row
- `-- name: FunctionName :many` - Returns multiple rows
- `-- name: FunctionName :exec` - Executes without returning data
- `$1, $2, $3...` - Positional parameters
- `sqlc.narg('field')` - Optional/nullable parameter for updates

---

## Step 3: Update sqlc.yaml Configuration

After creating your query folder, add it to `sqlc.yaml` so sqlc knows to generate code for it.

**File: `sqlc.yaml`**

```yaml
version: "2"
sql:
  - schema: "./db/migrations/"
    queries:
      - "./db/query/account"
      - "./db/query/collaborate"
      - "./db/query/department"    # ← Add your new query folder here
    engine: "postgresql"
    gen:
      go:
        package: "dbgen"
        sql_package: "pgx/v5"
        out: "./gen/db"
        emit_exported_queries: true
        emit_json_tags: false
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "uuid"
            go_type: "github.com/google/uuid.NullUUID"
            nullable: true
```

### What to Add

Add the path to your query folder under the `queries:` section:
```yaml
queries:
  - "./db/query/department"  # Add this line
```

---

## Step 4: Generate Type-Safe Go Code

After updating `sqlc.yaml`, generate the Go code from your SQL queries.

### Generate Code

```bash
make sqlc-generate
```

Or directly:
```bash
sqlc generate
```

### What Gets Generated

sqlc creates files in `gen/db/`:

- **`db.go`** - Base query interface and transaction support
- **`models.go`** - Go structs matching your database tables
- **`department.sql.go`** - Go functions for each SQL query

### Example Generated Code

For the query:
```sql
-- name: GetDepartmentById :one
SELECT id, name, description FROM department.department WHERE id = $1;
```

sqlc generates:
```go
func (q *Queries) GetDepartmentById(ctx context.Context, id uuid.UUID) (DepartmentDepartment, error) {
    // Implementation...
}
```

### Using Generated Code

```go
import "your-project/gen/db"

// Initialize queries
queries := dbgen.New(db)

// Use generated function
dept, err := queries.GetDepartmentById(ctx, departmentID)
if err != nil {
    return err
}
```

---

## Complete Workflow Example

Let's create a complete example: Adding a new "Product" feature.

### Step 1: Create Migration

```bash
make migrate-create NAME=create_products
```

**Edit `db/migrations/000004_create_products.up.sql`:**
```sql
CREATE SCHEMA IF NOT EXISTS product;

CREATE TABLE IF NOT EXISTS product.product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Edit `db/migrations/000004_create_products.down.sql`:**
```sql
DROP TABLE IF EXISTS product.product;
DROP SCHEMA IF EXISTS product;
```

### Step 2: Create Query File

**Create folder and file: `db/query/product/product.sql`**

```sql
-- name: CreateProduct :one
INSERT INTO product.product (name, price)
VALUES ($1, $2)
RETURNING id, name, price, created_at;

-- name: GetProductById :one
SELECT id, name, price, created_at
FROM product.product
WHERE id = $1;
```

### Step 3: Update sqlc.yaml

Add to `queries:` section:
```yaml
queries:
  - "./db/query/account"
  - "./db/query/collaborate"
  - "./db/query/department"
  - "./db/query/product"    # ← Add this
```

### Step 4: Generate Code

```bash
make sqlc-generate
```

Done! You now have type-safe Go functions for your Product queries.

---

## Commands Reference

### Migration Commands

```bash
# Create new migration
make migrate-create NAME=migration_name

# Apply all migrations
make migrate-up

# Rollback migrations
make migrate-down
```

### Code Generation

```bash
# Generate type-safe Go code from SQL queries
make sqlc-generate

# Or directly
sqlc generate
```

### Manual SQL Execution

```bash
# Run migration file directly
sudo -u postgres psql -d aci_db -f db/migrations/000003_departments.up.sql
```

---

## Best Practices

### 1. Schema Organization
- Use separate schemas for different domains (e.g., `account`, `department`, `product`)
- Keep related tables in the same schema
- Use descriptive migration names

### 2. Query Organization
- Group queries by domain/feature in separate folders
- One SQL file per entity/feature
- Use consistent naming: `entity_name.sql`

### 3. Query Naming
- Use clear, descriptive function names
- Follow pattern: `ActionEntity` (e.g., `CreateDepartment`, `GetUserById`)
- Use appropriate return types (`:one`, `:many`, `:exec`)

### 4. Code Generation
- **Never edit files in `gen/db/`** - they are auto-generated
- Always regenerate after changing SQL queries
- Regenerate after schema changes
- Delete `gen/db/` folder if you need a clean rebuild

### 5. Workflow Order
Always follow this order:
1. ✅ Create/update migration
2. ✅ Apply migration to database
3. ✅ Create/update query file
4. ✅ Update `sqlc.yaml` (if new folder)
5. ✅ Generate code with `make sqlc-generate`

---

## Troubleshooting

### Generated code is outdated
```bash
# Delete and regenerate
rm -rf gen/db
make sqlc-generate
```

### sqlc can't find queries
- Check `sqlc.yaml` has the correct query folder path
- Ensure query files have proper annotations (`-- name: ...`)

### Migration conflicts
- Check migration numbering (000001, 000002, etc.)
- Ensure down migrations properly reverse up migrations

---

## File Structure

```
db/
├── migrations/              # Database schema definitions
│   ├── 000001_account.up.sql
│   ├── 000001_account.down.sql
│   ├── 000002_add_oauth.up.sql
│   └── ...
├── query/                   # SQL queries organized by domain
│   ├── account/
│   │   ├── account_user.sql
│   │   └── account_session.sql
│   ├── department/
│   │   └── department.sql
│   └── ...
└── DB_WORKFLOW.md          # This file

gen/db/                      # Auto-generated Go code (DO NOT EDIT)
├── db.go
├── models.go
├── account_user.sql.go
├── department.sql.go
└── ...

sqlc.yaml                    # sqlc configuration
```

---

## Summary

1. **Create Schema** → Write migration files in `db/migrations/`
2. **Create Queries** → Write SQL queries in `db/query/{domain}/`
3. **Update Config** → Add query folder path to `sqlc.yaml`
4. **Generate Code** → Run `make sqlc-generate`

That's it! You now have type-safe database access in Go. 🚀

