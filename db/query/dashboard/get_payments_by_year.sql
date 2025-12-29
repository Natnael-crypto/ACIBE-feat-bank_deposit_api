-- name: GetDashboardPaymentsByYear :many
-- Get all unique payments for a specific year grouped by claim month for totalBilled calculation
-- OPTIMIZED: Groups payments by unique constraint first, then aggregates by month and type
-- Returns payments grouped by unique constraint, then aggregated by claim month and claim type
-- IMPORTANT: Payments are filtered by claim_from_date (matching dashboard API) to match total expected calculations
-- $1::int = year
WITH year_date_range AS (
    -- Pre-compute date range once
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end
),
unique_claims AS (
    -- Get unique claims for the year (one row per unique claim key) - using array directly for better performance
    SELECT DISTINCT ON (nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(nc.modifiers, ARRAY[]::text[]))
        nc.claim_id,
        COALESCE(nc.charge_id, '') as charge_id,
        COALESCE(nc.cpt_code, '') as cpt_code,
        COALESCE(nc.modifiers, ARRAY[]::text[]) as modifiers,
        EXTRACT(MONTH FROM nc.claim_from_date)::int as payment_month,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE nc.claim_from_date >= ydr.year_start
      AND nc.claim_from_date < ydr.year_end
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(nc.modifiers, ARRAY[]::text[]), nc.claim_from_date DESC
),
payments_grouped AS (
    -- First group payments by unique payment constraint to handle duplicates
    SELECT 
        uc.payment_month,
        SUM(p.insurance_paid_a)::numeric as total_billed,
        MAX(uc.revenue_code) as revenue_code,
        MAX(uc.claim_cpt_code) as claim_cpt_code,
        MAX(p.cpt_code) as payment_cpt_code
    FROM collaboratemd.new_payment p
    INNER JOIN unique_claims uc ON p.claim_id = uc.claim_id
        AND COALESCE(p.charge_id, '') = uc.charge_id
        AND COALESCE(p.cpt_code, '') = uc.cpt_code
        AND COALESCE(p.modifiers, ARRAY[]::text[]) = uc.modifiers
    GROUP BY 
        uc.payment_month,
        -- Group by unique payment constraint to handle duplicate payments
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment
),
payments_by_month_type AS (
    -- Then aggregate by month and claim type
    SELECT 
        payment_month,
        TO_CHAR(TO_DATE(payment_month::text, 'MM'), 'Month') as month_name,
        CASE 
            WHEN payment_cpt_code = 'H0038' THEN 'peer'
            WHEN (revenue_code IN ('0126', '0128') OR UPPER(claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND claim_cpt_code != 'H0038' THEN 'inpatient'
            WHEN claim_cpt_code IS NOT NULL AND claim_cpt_code != '' AND claim_cpt_code != 'H0038' 
                AND (revenue_code IS NULL OR revenue_code NOT IN ('0126', '0128'))
                AND UPPER(claim_cpt_code) NOT IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 'outpatient'
            ELSE 'outpatient'
        END as claim_type,
        SUM(total_billed)::numeric as total_billed
    FROM payments_grouped
    GROUP BY payment_month, claim_type
)
SELECT 
    payment_month as month_number,
    month_name,
    claim_type,
    total_billed
FROM payments_by_month_type
ORDER BY payment_month, claim_type;

