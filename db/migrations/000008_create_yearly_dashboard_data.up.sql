-- Create materialized view for yearly dashboard data
-- This aggregates the already computed monthly data into yearly totals
--
-- Business Logic:
-- Sums all monthly metrics from dashboard_annual_data grouped by year and category

DROP MATERIALIZED VIEW IF EXISTS collaboratemd.dashboard_yearly_data CASCADE;

CREATE MATERIALIZED VIEW collaboratemd.dashboard_yearly_data AS
SELECT 
    year,
    claim_category,
    SUM(unique_claims)::bigint as unique_claims,
    SUM(total_units)::numeric as total_units,
    SUM(total_amount)::numeric as total_amount,
    SUM(total_expected_amount)::numeric as total_expected_amount,
    SUM(unique_claims_with_payments)::bigint as unique_claims_with_payments,
    SUM(total_units_with_payments)::numeric as total_units_with_payments,
    SUM(total_payments_amount)::numeric as total_payments_amount,
    SUM(total_expected_amount_with_payments)::numeric as total_expected_amount_with_payments,
    SUM(unique_claims_with_outstanding)::bigint as unique_claims_with_outstanding,
    SUM(total_units_with_outstanding)::numeric as total_units_with_outstanding,
    SUM(total_outstanding_amount)::numeric as total_outstanding_amount
FROM collaboratemd.dashboard_annual_data
GROUP BY year, claim_category
ORDER BY year DESC, claim_category;

-- Create unique index for CONCURRENT refresh support
CREATE UNIQUE INDEX IF NOT EXISTS idx_dashboard_yearly_data_unique 
ON collaboratemd.dashboard_yearly_data (year, claim_category);

-- Create index for fast year-based lookups
CREATE INDEX IF NOT EXISTS idx_dashboard_yearly_data_year 
ON collaboratemd.dashboard_yearly_data (year DESC);

-- Analyze the materialized view to update statistics
ANALYZE collaboratemd.dashboard_yearly_data;

-- Create function to refresh the materialized view
CREATE OR REPLACE FUNCTION collaboratemd.refresh_dashboard_yearly_data()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY collaboratemd.dashboard_yearly_data;
    ANALYZE collaboratemd.dashboard_yearly_data;
END;
$$;

COMMENT ON MATERIALIZED VIEW collaboratemd.dashboard_yearly_data IS 
'Pre-computed yearly dashboard data. Derived from dashboard_annual_data.';
