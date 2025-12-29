-- Create materialized view for dashboard data
-- This pre-computes all dashboard metrics for ALL years, enabling sub-100ms queries
-- The view aggregates data by year, month, and claim category (IN/PEER/OUT)
--
-- Business Logic:
-- 1. Monthly grouping based on new_claim.claim_from_date
-- 2. Unique claim = claim_id (even if multiple rows with different charge_id or CPT)
-- 3. Units = SUM(units) for all charge_id rows for that claim_id
-- 4. Total Claim Amount = SUM(amount) across all charge lines
-- 5. Total Expected Amount = SUM(expected_amount) from new_claim grouped by (claim_id + charge_id)
-- 6. Categorization: IN > PEER > OUT (priority-based)
-- 7. Payments: Uses ALL payments for each claim_id (not matched by charge_id + cpt_code + modifiers)
--    This matches GetClaimsCount logic and ensures Dashboard API count matches Claims API count
-- 8. Outstanding calculation: (sum of expected_amount for month) - (sum of ALL payments for claim)
-- 9. Each claim counted only once in primary category (prevents double counting for outstanding)

-- Drop the materialized view if it exists to allow recreation with new columns
DROP MATERIALIZED VIEW IF EXISTS collaboratemd.dashboard_annual_data CASCADE;

CREATE MATERIALIZED VIEW collaboratemd.dashboard_annual_data AS
WITH 
-- Step 1: Get ALL claim charge lines with category and priority
all_claim_lines AS (
    SELECT 
        nc.claim_id,
        nc.charge_id,
        nc.cpt_code,
        nc.modifiers,
        EXTRACT(YEAR FROM nc.claim_from_date)::int as year,
        EXTRACT(MONTH FROM nc.claim_from_date)::int as month_number,
        CASE 
            WHEN nc.revenue_code IN ('0128', '0126') 
              OR nc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 'IN'
            WHEN nc.cpt_code = 'H0038' THEN 'PEER'
            ELSE 'OUT'
        END AS claim_category,
        CASE 
            WHEN nc.revenue_code IN ('0128', '0126') 
              OR nc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 1
            WHEN nc.cpt_code = 'H0038' THEN 2
            ELSE 3
        END AS row_priority,
        COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0)::numeric as units,
        COALESCE(nc.amount, 0)::numeric as amount,
        COALESCE(nc.expected_amount, 0)::numeric as expected_amount
    FROM collaboratemd.new_claim nc
    WHERE nc.claim_from_date IS NOT NULL
),
-- Step 2: Aggregate claims by claim_id, year, month, category
claims_by_category AS (
    SELECT 
        claim_id,
        year,
        month_number,
        claim_category,
        MIN(row_priority) as row_priority,
        SUM(units)::numeric as total_units,
        SUM(amount)::numeric as total_amount,
        SUM(expected_amount)::numeric as total_expected_amount
    FROM all_claim_lines
    GROUP BY claim_id, year, month_number, claim_category
),
-- Step 2.5: Calculate total units for each claim across all categories
claim_total_sum_all_units AS (
    SELECT
        claim_id,
        year,
        month_number,
        SUM(total_units)::numeric as overall_claim_total_units
    FROM claims_by_category
    GROUP BY claim_id, year, month_number
),
-- Step 3: Get primary category for each claim (optimized with DISTINCT ON)
claim_primary AS (
    SELECT DISTINCT ON (claim_id, year, month_number)
        claim_id,
        year,
        month_number,
        CASE 
            WHEN row_priority = 1 THEN 'IN'
            WHEN row_priority = 2 THEN 'PEER'
            ELSE 'OUT'
        END as primary_category
    FROM claims_by_category
        ORDER BY claim_id, year, month_number, row_priority ASC
),
-- Step 4: Combine claims with primary category
unique_claims AS (
    SELECT 
        cbc.claim_id,
        cbc.year,
        cbc.month_number,
        cbc.claim_category,
        cp.primary_category,
        cbc.total_units,
        cbc.total_amount,
        cbc.total_expected_amount
    FROM claims_by_category cbc
    INNER JOIN claim_primary cp
        ON cbc.claim_id = cp.claim_id
        AND cbc.year = cp.year
        AND cbc.month_number = cp.month_number
),
-- Step 5: Calculate claim-level totals (sum of all categories for each claim)
claim_totals AS (
    SELECT 
        claim_id,
        year,
        month_number,
        SUM(total_expected_amount)::numeric as claim_total_expected_amount
    FROM unique_claims
    GROUP BY claim_id, year, month_number
),
-- Step 6: Get ALL payments for each claim (updated to match GetClaimsCount logic)
-- Since claim_id is unique per month, we use ALL payments for the claim
payments_by_claim_all AS (
    SELECT 
        claim_id,
        SUM(COALESCE(p.insurance_paid_a, 0))::numeric as total_payment_amount
    FROM collaboratemd.new_payment p
    GROUP BY claim_id
),
-- Step 7: Combine claims with payments and calculate all metrics
claims_with_all_data AS (
    SELECT 
        uc.claim_id,
        uc.year,
        uc.month_number,
        uc.claim_category,
        uc.primary_category,
        uc.total_units,
        uc.total_amount,
        uc.total_expected_amount,
        ctsu.overall_claim_total_units, -- NEW FIELD
        -- Calculate payment amount per category (for display purposes)
        COALESCE((SELECT SUM(p.insurance_paid_a)::numeric
                  FROM collaboratemd.new_payment p
                  WHERE p.claim_id = uc.claim_id), 0)::numeric as payment_amount,
        -- Check if claim has any payment record (even if amount is 0.0)
        CASE 
            WHEN EXISTS (SELECT 1 FROM collaboratemd.new_payment p WHERE p.claim_id = uc.claim_id) THEN 1
            ELSE 0
        END as has_payment_record,
        -- Outstanding: claim-level expected (all categories) minus ALL payments for the claim
        -- This matches GetClaimsCount logic - uses ALL payments, not matched by charge
        (ct.claim_total_expected_amount - COALESCE(pca.total_payment_amount, 0))::numeric as outstanding_amount
    FROM unique_claims uc
    INNER JOIN claim_totals ct 
        ON uc.claim_id = ct.claim_id
        AND uc.year = ct.year
        AND uc.month_number = ct.month_number
    LEFT JOIN payments_by_claim_all pca ON uc.claim_id = pca.claim_id
    LEFT JOIN claim_total_sum_all_units ctsu
        ON uc.claim_id = ctsu.claim_id
        AND uc.year = ctsu.year
        AND uc.month_number = ctsu.month_number
)
-- Step 8: Final aggregation by year, month and category
SELECT 
    year,
    month_number,
    CASE month_number
        WHEN 1 THEN 'January'
        WHEN 2 THEN 'February'
        WHEN 3 THEN 'March'
        WHEN 4 THEN 'April'
        WHEN 5 THEN 'May'
        WHEN 6 THEN 'June'
        WHEN 7 THEN 'July'
        WHEN 8 THEN 'August'
        WHEN 9 THEN 'September'
        WHEN 10 THEN 'October'
        WHEN 11 THEN 'November'
        WHEN 12 THEN 'December'
    END as month_name,
    claim_category,
    -- Total Claims: count unique claims in primary category
    COUNT(DISTINCT CASE 
        WHEN primary_category = claim_category THEN claim_id 
        ELSE NULL 
    END)::bigint as unique_claims,
    -- Total Claims: sum units, amount, expected
    SUM(CASE 
        WHEN primary_category = claim_category THEN overall_claim_total_units
        ELSE 0 
    END)::numeric as total_units,
    SUM(total_amount)::numeric as total_amount,
    SUM(total_expected_amount)::numeric as total_expected_amount,
    -- Payments: count unique claims with payments in primary category (even if payment is $0.00)
    COUNT(DISTINCT CASE 
        WHEN primary_category = claim_category AND has_payment_record = 1 THEN claim_id 
        ELSE NULL 
    END)::bigint as unique_claims_with_payments,
    -- Payments: sum units for claims with payments (even if payment is $0.00)
    SUM(CASE WHEN primary_category = claim_category AND has_payment_record = 1 THEN overall_claim_total_units ELSE 0 END)::numeric as total_units_with_payments,
    -- Payments: sum payment amounts (only count in primary category to avoid double counting)
    -- This ensures the sum matches bank deposit API which counts each claim only once
    SUM(CASE 
        WHEN primary_category = claim_category THEN payment_amount
        ELSE 0 
    END)::numeric as total_payments_amount,
    -- Expected amount for claims with payments (even if payment amount is 0.0)
    -- Sum expected_amount for claims that have at least one payment record
    SUM(CASE 
        WHEN has_payment_record = 1 THEN total_expected_amount
        ELSE 0 
    END)::numeric as total_expected_amount_with_payments,
    -- Outstanding: count unique claims with outstanding (only count in primary category to avoid double counting)
    -- This ensures the sum across categories matches the claims API count
    COUNT(DISTINCT CASE 
        WHEN primary_category = claim_category AND outstanding_amount != 0 THEN claim_id 
        ELSE NULL 
    END)::bigint as unique_claims_with_outstanding,
    -- Outstanding: sum units for outstanding claims (includes negative and zero values)
    SUM(CASE WHEN primary_category = claim_category AND outstanding_amount != 0 THEN overall_claim_total_units ELSE 0 END)::numeric as total_units_with_outstanding,
    -- Outstanding: calculated amount (includes negative values)
    -- Note: outstanding_amount is already calculated per row using ALL payments for the claim
    -- This matches GetClaimsCount logic (uses all payments, not matched by charge)
    SUM(outstanding_amount)::numeric as total_outstanding_amount
FROM claims_with_all_data
GROUP BY year, month_number, claim_category
ORDER BY year DESC, month_number, claim_category;

-- Create unique index for CONCURRENT refresh support
-- This is required for REFRESH MATERIALIZED VIEW CONCURRENTLY
CREATE UNIQUE INDEX IF NOT EXISTS idx_dashboard_annual_data_unique 
ON collaboratemd.dashboard_annual_data (year, month_number, claim_category);

-- Create index for fast year-based lookups (primary use case)
CREATE INDEX IF NOT EXISTS idx_dashboard_annual_data_year 
ON collaboratemd.dashboard_annual_data (year DESC, month_number, claim_category);

-- Create index for available years query
CREATE INDEX IF NOT EXISTS idx_dashboard_annual_data_year_only 
ON collaboratemd.dashboard_annual_data (year DESC);

-- Analyze the materialized view to update statistics
ANALYZE collaboratemd.dashboard_annual_data;

-- Create function to refresh the materialized view
-- This function can be called manually or scheduled via cron
CREATE OR REPLACE FUNCTION collaboratemd.refresh_dashboard_annual_data()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    -- Use CONCURRENT refresh to allow queries during refresh
    -- Note: CONCURRENT requires the unique index above
    REFRESH MATERIALIZED VIEW CONCURRENTLY collaboratemd.dashboard_annual_data;
    
    -- Update statistics after refresh
    ANALYZE collaboratemd.dashboard_annual_data;
END;
$$;

-- Create function to get available years from materialized view
-- This is faster than querying the available_years table
CREATE OR REPLACE FUNCTION collaboratemd.get_dashboard_available_years()
RETURNS TABLE(year INTEGER)
LANGUAGE sql
STABLE
AS $$
    SELECT DISTINCT year
    FROM collaboratemd.dashboard_annual_data
    ORDER BY year DESC;
$$;

COMMENT ON MATERIALIZED VIEW collaboratemd.dashboard_annual_data IS 
'Pre-computed dashboard data for all years. Refreshed nightly via refresh_dashboard_annual_data() function. 
Enables sub-100ms queries for dashboard endpoints.';

COMMENT ON FUNCTION collaboratemd.refresh_dashboard_annual_data() IS 
'Refreshes the dashboard_annual_data materialized view using CONCURRENT mode. 
Can be called manually or scheduled via cron for nightly updates.';

COMMENT ON FUNCTION collaboratemd.get_dashboard_available_years() IS 
'Returns distinct years available in the dashboard materialized view. 
Faster than querying available_years table.';

