-- name: GetDashboardPaymentsUniqueClaimCounts :many
-- Get count of unique claims that have payments, grouped by month and claim type
-- Returns the number of distinct claims (by claim_id + charge_id + cpt_code + modifiers) that have payments
-- IMPORTANT: Payments are filtered by claim_from_date (matching dashboard API) to match total expected calculations
-- $1::int = year
WITH year_date_range AS (
    -- Pre-compute date range once
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end
),
unique_claims_base AS (
    -- Get unique claims for the year - simplified to use array directly
    SELECT DISTINCT ON (nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(nc.modifiers, ARRAY[]::text[]))
        nc.claim_id,
        COALESCE(nc.charge_id, '') as charge_id,
        COALESCE(nc.cpt_code, '') as cpt_code,
        COALESCE(nc.modifiers, ARRAY[]::text[]) as modifiers,
        EXTRACT(MONTH FROM nc.claim_from_date)::int as month_number,
        TO_CHAR(nc.claim_from_date, 'Month') as month_name,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code,
        COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0) as units_numeric
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE nc.claim_from_date >= ydr.year_start
      AND nc.claim_from_date < ydr.year_end
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(nc.modifiers, ARRAY[]::text[]), nc.claim_from_date DESC
),
claims_with_payments AS (
    -- Get unique claims that have payments - use EXISTS for better performance
    SELECT 
        uc.claim_id,
        uc.charge_id,
        uc.cpt_code,
        uc.modifiers,
        uc.month_number,
        uc.month_name,
        uc.revenue_code,
        uc.claim_cpt_code,
        uc.units_numeric
    FROM unique_claims_base uc
    WHERE EXISTS (
        SELECT 1 FROM collaboratemd.new_payment p 
        WHERE p.claim_id = uc.claim_id
        AND COALESCE(p.charge_id, '') = uc.charge_id
        AND COALESCE(p.cpt_code, '') = uc.cpt_code
        AND COALESCE(p.modifiers, ARRAY[]::text[]) = uc.modifiers
        LIMIT 1
    )
),
claims_by_month_type AS (
    -- Group by month and claim type, count distinct claims and sum units
    SELECT 
        month_number,
        month_name,
        CASE 
            WHEN claim_cpt_code = 'H0038' THEN 'PEER'
            WHEN (revenue_code IN ('0126', '0128') OR UPPER(claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND claim_cpt_code != 'H0038' THEN 'IN'
            ELSE 'OUT'
        END as claim_type,
        COUNT(*)::bigint as unique_claim_count,
        SUM(COALESCE(units_numeric, 0))::numeric as total_units
    FROM claims_with_payments
    GROUP BY month_number, month_name, 
        CASE 
            WHEN claim_cpt_code = 'H0038' THEN 'PEER'
            WHEN (revenue_code IN ('0126', '0128') OR UPPER(claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND claim_cpt_code != 'H0038' THEN 'IN'
            ELSE 'OUT'
        END
)
SELECT 
    month_number,
    month_name,
    claim_type,
    unique_claim_count,
    total_units
FROM claims_by_month_type
ORDER BY month_number, claim_type;

-- name: GetDashboardOutstandingUniqueClaimCounts :many
-- Get count of unique claims that have outstanding amounts (expected - billed != 0), grouped by month and claim type
-- OPTIMIZED: Removed expensive array_to_string, simplified structure, use array comparison directly
-- Returns the number of distinct claims (by claim_id + charge_id + cpt_code + modifiers) that have outstanding amounts
-- For PEER: includes claims where expected != billed (no payment, partial payment, or overpayment)
-- For IN/OUT: includes claims with any non-zero outstanding amount (!= 0)
-- IMPORTANT: Uses claim_from_date for month calculation to match dashboard API and total expected calculations
-- $1::int = year
WITH year_date_range AS (
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end
),
claims_aggregated AS (
    -- Aggregate expected amounts and units per unique claim - group first, then take distinct
    SELECT 
        nc.claim_id,
        COALESCE(nc.charge_id, '') as charge_id,
        COALESCE(nc.cpt_code, '') as cpt_code,
        COALESCE(nc.modifiers, ARRAY[]::text[]) as modifiers,
        EXTRACT(MONTH FROM nc.claim_from_date)::int as month_number,
        TO_CHAR(nc.claim_from_date, 'Month') as month_name,
        MAX(nc.revenue_code) as revenue_code,
        MAX(nc.cpt_code) as claim_cpt_code,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0))::numeric as total_units
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE nc.claim_from_date >= ydr.year_start
      AND nc.claim_from_date < ydr.year_end
    GROUP BY 
        nc.claim_id,
        COALESCE(nc.charge_id, ''),
        COALESCE(nc.cpt_code, ''),
        COALESCE(nc.modifiers, ARRAY[]::text[]),
        EXTRACT(MONTH FROM nc.claim_from_date),
        TO_CHAR(nc.claim_from_date, 'Month')
),
claims_with_billed AS (
    -- Calculate total billed using LEFT JOIN for better performance
    SELECT 
        ca.claim_id,
        ca.charge_id,
        ca.cpt_code,
        ca.modifiers,
        ca.month_number,
        ca.month_name,
        ca.revenue_code,
        ca.claim_cpt_code,
        ca.total_expected,
        ca.total_units,
        COALESCE(SUM(p.insurance_paid_a), 0)::numeric as total_billed
    FROM claims_aggregated ca
    LEFT JOIN collaboratemd.new_payment p ON p.claim_id = ca.claim_id
        AND COALESCE(p.charge_id, '') = ca.charge_id
        AND COALESCE(p.cpt_code, '') = ca.cpt_code
        AND COALESCE(p.modifiers, ARRAY[]::text[]) = ca.modifiers
    GROUP BY 
        ca.claim_id, ca.charge_id, ca.cpt_code, ca.modifiers,
        ca.month_number, ca.month_name, ca.revenue_code, ca.claim_cpt_code,
        ca.total_expected, ca.total_units
),
claims_with_outstanding AS (
    -- Filter to only claims with outstanding amounts
    SELECT 
        claim_id,
        charge_id,
        cpt_code,
        modifiers,
        month_number,
        month_name,
        revenue_code,
        claim_cpt_code,
        total_units
    FROM claims_with_billed
    WHERE (total_expected - total_billed) != 0
),
claims_by_month_type AS (
    -- Group by month and claim type, count and sum units
    SELECT 
        month_number,
        month_name,
        CASE 
            WHEN claim_cpt_code = 'H0038' THEN 'PEER'
            WHEN (revenue_code IN ('0126', '0128') OR UPPER(claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND claim_cpt_code != 'H0038' THEN 'IN'
            ELSE 'OUT'
        END as claim_type,
        COUNT(*)::bigint as unique_claim_count,
        SUM(COALESCE(total_units, 0))::numeric as total_units
    FROM claims_with_outstanding
    GROUP BY month_number, month_name, claim_type
)
SELECT 
    month_number,
    month_name,
    claim_type,
    unique_claim_count,
    total_units
FROM claims_by_month_type
ORDER BY month_number, claim_type;

