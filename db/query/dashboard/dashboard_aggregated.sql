-- name: GetDashboardAggregatedData :many
-- Get pre-aggregated dashboard data by month and claim type
-- OPTIMIZED: Simplified structure, removed expensive array_to_string in COUNT(DISTINCT)
-- Returns aggregated counts, totals, and units for each month and type (IN, OUT, PEER)
-- $1::int = year
WITH year_date_range AS (
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end
),
unique_claims_base AS (
    -- Get unique claims first using DISTINCT ON for better performance
    SELECT DISTINCT ON (nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), nc.modifiers)
        nc.claim_id,
        COALESCE(nc.charge_id, '') as charge_id,
        COALESCE(nc.cpt_code, '') as cpt_code,
        nc.modifiers,
        EXTRACT(MONTH FROM nc.claim_entered_date)::int as month_number,
        TO_CHAR(nc.claim_entered_date, 'Month') as month_name,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code,
        nc.amount,
        nc.expected_amount,
        COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0) as units_numeric
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE nc.claim_entered_date >= ydr.year_start
      AND nc.claim_entered_date < ydr.year_end
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), nc.modifiers, nc.claim_entered_date DESC
),
monthly_aggregates AS (
    SELECT 
        month_number,
        month_name,
        CASE 
            WHEN claim_cpt_code = 'H0038' THEN 'PEER'
            WHEN (revenue_code IN ('0126', '0128') OR UPPER(claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND claim_cpt_code != 'H0038' THEN 'IN'
            ELSE 'OUT'
        END as claim_type,
        COUNT(*)::bigint as claim_count,
        SUM(COALESCE(amount, 0))::numeric as total_claims_amount,
        SUM(COALESCE(expected_amount, 0))::numeric as total_expected_amount,
        SUM(units_numeric)::numeric as total_units
    FROM unique_claims_base
    GROUP BY month_number, month_name, claim_type
)
SELECT 
    month_number,
    month_name,
    claim_type,
    claim_count,
    total_claims_amount,
    total_expected_amount,
    total_units
FROM monthly_aggregates
ORDER BY month_number, claim_type;

