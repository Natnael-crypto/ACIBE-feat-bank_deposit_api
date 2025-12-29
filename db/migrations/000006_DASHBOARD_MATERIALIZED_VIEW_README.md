# Dashboard Materialized View - Setup Guide

## Overview

This migration creates a materialized view `dashboard_annual_data` that pre-computes all dashboard metrics for ALL years, enabling sub-100ms query response times (down from 3+ seconds).

## What's Included

1. **Materialized View**: `collaboratemd.dashboard_annual_data`
   - Pre-aggregated data for all years
   - Indexed for fast lookups
   - Supports CONCURRENT refresh

2. **Refresh Function**: `collaboratemd.refresh_dashboard_annual_data()`
   - Uses CONCURRENT mode (allows queries during refresh)
   - Updates statistics after refresh

3. **Helper Function**: `collaboratemd.get_dashboard_available_years()`
   - Returns distinct years from materialized view
   - Faster than querying available_years table

## Deployment Steps

### 1. Run the Migration

```bash
# Apply the migration
migrate -path ./db/migrations -database "your-db-url" up

# Or using your migration tool
```

### 2. Initial Data Population

The materialized view will be populated automatically when the migration runs. However, for large datasets, you may want to refresh it manually:

```sql
-- Refresh the materialized view (this may take several minutes for large datasets)
SELECT collaboratemd.refresh_dashboard_annual_data();
```

### 3. Regenerate Go Code

After running the migration, regenerate the Go code using sqlc:

```bash
cd ACIBE
sqlc generate
```

This will update the generated query code in `gen/db/` to use the new materialized view query.

### 4. Set Up Nightly Refresh (Recommended)

The materialized view should be refreshed nightly to include new data. You can set this up using:

#### Option A: PostgreSQL pg_cron Extension (Recommended)

```sql
-- Enable pg_cron extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Schedule nightly refresh at 2 AM
SELECT cron.schedule(
    'refresh-dashboard-annual-data',
    '0 2 * * *',  -- Daily at 2:00 AM
    $$SELECT collaboratemd.refresh_dashboard_annual_data();$$
);
```

#### Option B: System Cron Job

Add to your crontab:

```bash
# Edit crontab
crontab -e

# Add this line (runs daily at 2 AM)
0 2 * * * psql -d your_database -c "SELECT collaboratemd.refresh_dashboard_annual_data();"
```

#### Option C: Application-Level Scheduler

If you're using a Go scheduler (like `robfig/cron`), add this to your application:

```go
// In your main.go or scheduler setup
c := cron.New()
c.AddFunc("0 2 * * *", func() {
    // Execute refresh function
    _, err := db.Exec(context.Background(), 
        "SELECT collaboratemd.refresh_dashboard_annual_data();")
    if err != nil {
        log.Printf("Failed to refresh dashboard materialized view: %v", err)
    }
})
c.Start()
```

## Usage

### Query Dashboard Data

The application code automatically uses the materialized view. No changes needed to your Go code - it will use the optimized query after running `sqlc generate`.

### Manual Refresh

If you need to refresh the view manually (e.g., after bulk data imports):

```sql
-- Concurrent refresh (recommended - allows queries during refresh)
SELECT collaboratemd.refresh_dashboard_annual_data();

-- Or direct refresh (blocks queries during refresh, but faster)
REFRESH MATERIALIZED VIEW collaboratemd.dashboard_annual_data;
```

### Get Available Years

You can optionally use the helper function to get available years:

```sql
SELECT * FROM collaboratemd.get_dashboard_available_years();
```

Or continue using the existing `available_years` table (both work).

## Performance

- **Before**: 3+ seconds per query
- **After**: <100ms per query (typically 10-50ms)

The materialized view eliminates:
- Multiple table scans
- Complex CTEs
- Expensive joins
- Aggregation calculations

## Monitoring

### Check Materialized View Size

```sql
SELECT 
    schemaname,
    matviewname,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||matviewname)) as size
FROM pg_matviews
WHERE matviewname = 'dashboard_annual_data';
```

### Check Last Refresh Time

```sql
-- Note: PostgreSQL doesn't track refresh time automatically
-- You may want to add a refresh_log table if needed
```

### Verify Data Freshness

```sql
-- Compare counts between materialized view and source tables
SELECT 
    (SELECT COUNT(DISTINCT year) FROM collaboratemd.dashboard_annual_data) as mv_years,
    (SELECT COUNT(*) FROM collaboratemd.available_years) as table_years;
```

## Rollback

If you need to rollback this migration:

```bash
migrate -path ./db/migrations -database "your-db-url" down
```

This will:
- Drop the refresh function
- Drop the helper function
- Drop the materialized view and all indexes

## Troubleshooting

### Issue: "REFRESH MATERIALIZED VIEW CONCURRENTLY requires a unique index"

**Solution**: The migration creates the required unique index automatically. If you see this error, check that the migration ran successfully.

### Issue: Refresh takes too long

**Solution**: 
- Use CONCURRENT refresh (already configured)
- Consider refreshing during off-peak hours
- For very large datasets, you may need to increase `work_mem` temporarily

### Issue: Data appears stale

**Solution**: 
- Verify the refresh job is running: `SELECT * FROM cron.job WHERE jobname = 'refresh-dashboard-annual-data';`
- Manually trigger a refresh: `SELECT collaboratemd.refresh_dashboard_annual_data();`
- Check for errors in your cron logs

## Maintenance

### Regular Maintenance Tasks

1. **Monitor refresh times**: Track how long refreshes take
2. **Check disk space**: Materialized views consume storage
3. **Verify data accuracy**: Periodically compare with source data
4. **Update statistics**: PostgreSQL auto-analyzes, but you can force it:
   ```sql
   ANALYZE collaboratemd.dashboard_annual_data;
   ```

### When to Refresh

- **Nightly**: Recommended for production
- **After bulk imports**: Refresh immediately after large data imports
- **On-demand**: If you notice stale data

## Notes

- The materialized view includes data for ALL years, not just the current year
- Historical data is static (only current year changes daily)
- The view uses the same business logic as the original query
- CONCURRENT refresh allows queries to continue during refresh (slower but non-blocking)
- Regular refresh is faster but blocks queries (use during maintenance windows)

