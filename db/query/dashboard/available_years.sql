-- name: GetAvailableYears :many
-- Get available years from the available_years table (no need to query all claims)
-- This table is automatically maintained by triggers when claims are inserted/updated
SELECT year
FROM collaboratemd.available_years
ORDER BY year DESC;

