-- name: GetDashboardPaymentsTotalBilled :one
-- Get total billed amount for all payments in a specific month/year
-- Groups payments by unique constraint first, then sums insurance_paid_a
-- Filters by claim type (inpatient/outpatient) by joining with claims table
-- IMPORTANT: Payments are filtered by claim_from_date (matching dashboard API) to match total expected calculations
-- $1::int = year
-- $2::int = month (0 = all months)
-- $3::boolean = exclude_peer (exclude payments with CPT code H0038)
-- $4::boolean = peer_only (include only payments with CPT code H0038)
-- $5::text = claim_type filter (NULL/empty = all types, 'inpatient' = IN only, 'outpatient' = OUT only)
WITH date_range AS (
    -- Date range for filtering (matching dashboard API logic)
    SELECT 
        CASE WHEN $1::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) END as year_start,
        CASE WHEN $1::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' END as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
filtered_claims AS (
    -- Filter claims by year/month using claim_from_date (matching dashboard API logic)
    SELECT DISTINCT claim_id
    FROM collaboratemd.new_claim nc
    CROSS JOIN date_range dr
    WHERE 
        -- Filter by year using claim_from_date (same as dashboard API)
        (dr.year_start IS NULL OR (nc.claim_from_date >= dr.year_start AND nc.claim_from_date < dr.year_end))
        -- Filter by month using claim_from_date (same as dashboard API)
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM nc.claim_from_date) = dr.month_filter)
),
unique_claims AS (
    -- Get unique claims by (claim_id, charge_id, cpt_code, modifiers) using DISTINCT ON
    -- This matches the dashboard API logic for filtering by claim_from_date
    SELECT DISTINCT ON (nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), ''))
        nc.claim_id,
        COALESCE(nc.charge_id, '') as charge_id,
        COALESCE(nc.cpt_code, '') as cpt_code,
        COALESCE(array_to_string(nc.modifiers, ','), '') as modifiers_str,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code
    FROM collaboratemd.new_claim nc
    INNER JOIN filtered_claims fc ON nc.claim_id = fc.claim_id
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), ''), nc.claim_from_date DESC
),
payments_grouped AS (
    SELECT 
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment,
        -- Sum insurance_paid_a for duplicate payments (same unique constraint)
        -- Include both peer and non-peer payments
        SUM(p.insurance_paid_a) as total_billed
    FROM collaboratemd.new_payment p
    INNER JOIN unique_claims uc ON p.claim_id = uc.claim_id
        AND COALESCE(p.charge_id, '') = uc.charge_id
        AND COALESCE(p.cpt_code, '') = uc.cpt_code
        AND COALESCE(array_to_string(p.modifiers, ','), '') = uc.modifiers_str
    WHERE 
        -- Include only peer payments (H0038) if peer_only is true
        (($4::boolean IS NULL OR $4::boolean = false) OR ($4::boolean = true AND p.cpt_code = 'H0038'))
        -- Exclude peer payments (H0038) if exclude_peer is true (but not if peer_only is true)
        AND NOT ($3::boolean = true AND ($4::boolean IS NULL OR $4::boolean = false) AND p.cpt_code = 'H0038')
        -- Filter by claim type (inpatient/outpatient) if provided
        -- Special inpatient CPT codes: H2019, H0017, H0019, 99235, H0010, H2036
        AND ($5::text IS NULL OR $5::text = '' OR 
            CASE 
                WHEN LOWER($5::text) = 'inpatient' THEN 
                    (uc.revenue_code IN ('0126', '0128') OR UPPER(uc.claim_cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036'))
                    AND uc.claim_cpt_code != 'H0038'
                WHEN LOWER($5::text) = 'outpatient' THEN 
                    uc.claim_cpt_code IS NOT NULL 
                    AND uc.claim_cpt_code != '' 
                    AND uc.claim_cpt_code != 'H0038' 
                    AND (uc.revenue_code IS NULL OR uc.revenue_code NOT IN ('0126', '0128'))
                    AND UPPER(uc.claim_cpt_code) NOT IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')
                ELSE true
            END)
    GROUP BY 
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment
)
SELECT 
    COALESCE(SUM(total_billed), 0)::numeric as total_billed
FROM payments_grouped;

