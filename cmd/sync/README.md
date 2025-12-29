# Manual Sync Command

This command allows you to manually run the cron jobs that sync data from the Collaborate API to your database tables.

## What it does

- **Claims Sync**: Adds new claims to the `newclaims` table
- **Payments Sync**: Adds new payments to the new payment table  
- **Insurance Checks Sync**: Adds insurance checks to the insurance check table

## Usage

### Using Go directly:

```bash
# Run all sync jobs
go run ./cmd/sync -all

# Run only claims sync
go run ./cmd/sync -claims

# Run only payments sync
go run ./cmd/sync -payments

# Run only insurance checks sync
go run ./cmd/sync -insurance

# Run multiple specific jobs
go run ./cmd/sync -claims -payments
```

### Using Make commands:

```bash
# Run all sync jobs
make sync

# Run only claims sync
make sync-claims

# Run only payments sync
make sync-payments

# Run only insurance checks sync
make sync-insurance
```

## Configuration

The command uses the same configuration as the main application (from `config.yaml` or environment variables). Make sure your database and Collaborate API credentials are properly configured before running.

## Examples

```bash
# Sync everything
make sync

# Just update claims
make sync-claims

# Update payments and insurance checks (but not claims)
go run ./cmd/sync -payments -insurance
```
