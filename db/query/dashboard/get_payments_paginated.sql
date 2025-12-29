-- name: GetDashboardPaymentsPaginated :many
-- Get paginated payments with filters for month, year, category, search, claim_type
-- Returns unique payments grouped by claim_id + charge_id (matching claims API logic)
-- IMPORTANT: Payments are filtered by claim_from_date (matching dashboard API) to match total expected calculations
-- 
-- IMPORTANT: This query first gets unique claims from new_claim table for the specified month/year,
-- then gets payments for those unique claims. Uses claim_from_date for filtering (matching dashboard API)
-- to ensure we only show payments for claims that were billed (claim_from_date) in that month.
--
-- $1::int = year (0 = all years)
-- $2::int = month (0 = all months)
-- $3::boolean = exclude_peer (exclude payments with CPT code H0038)
-- $4::boolean = peer_only (include only payments with CPT code H0038)
-- $5::text = search (optional search term)
-- $6::int = limit
-- $7::int = offset
-- $8::text = claim_type filter (NULL/empty = all types, 'inpatient' = IN only, 'outpatient' = OUT only)
-- $9::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $10::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
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
        nc.claim_from_date,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code
    FROM collaboratemd.new_claim nc
    INNER JOIN filtered_claims fc ON nc.claim_id = fc.claim_id
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), ''), nc.claim_from_date DESC
),
all_claim_categories AS (
    -- Get all categories for each claim (a claim can have multiple categories)
    -- This matches the materialized view logic in 000006_create_dashboard_materialized_view.up.sql
    SELECT DISTINCT
        uc.claim_id,
        uc.charge_id,
        uc.cpt_code,
        uc.modifiers_str,
        uc.claim_from_date,
        uc.revenue_code,
        CASE 
            WHEN uc.revenue_code IN ('0128', '0126') 
              OR uc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 'IN'
            WHEN uc.cpt_code = 'H0038' THEN 'PEER'
            ELSE 'OUT'
        END AS charge_category,
        CASE 
            WHEN uc.revenue_code IN ('0128', '0126') 
              OR uc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 1
            WHEN uc.cpt_code = 'H0038' THEN 2
            ELSE 3
        END AS category_priority
    FROM unique_claims uc
),
claim_primary_category AS (
    -- Determine primary category for each claim using priority: IN (1) > PEER (2) > OUT (3)
    -- This matches the materialized view logic
    SELECT DISTINCT ON (claim_id, charge_id)
        claim_id,
        charge_id,
        cpt_code,
        modifiers_str,
        claim_from_date,
        revenue_code,
        CASE 
            WHEN MIN(category_priority) OVER (PARTITION BY claim_id) = 1 THEN 'inpatient'
            WHEN MIN(category_priority) OVER (PARTITION BY claim_id) = 2 THEN 'peer'
            ELSE 'outpatient'
        END as determined_claim_type
    FROM all_claim_categories
    ORDER BY claim_id, charge_id, category_priority ASC
),
filtered_claims_by_type AS (
    -- Apply claim type filter (if provided) - same logic as claims API
    SELECT 
        claim_id,
        charge_id,
        cpt_code,
        modifiers_str,
        claim_from_date,
        revenue_code,
        determined_claim_type
    FROM claim_primary_category
    WHERE 
        -- Filter by claim type (if provided) - match EXACT logic from claims API
        ($8::text IS NULL OR $8::text = '' OR LOWER(determined_claim_type) = LOWER($8::text))
        -- Exclude peer claims if exclude_peer is true
        AND NOT ($3::boolean = true AND determined_claim_type = 'peer')
        -- Include only peer claims if peer_only is true
        AND (($4::boolean IS NULL OR $4::boolean = false) OR ($4::boolean = true AND determined_claim_type = 'peer'))
),
unique_claims_by_charge AS (
    -- Get unique claims by claim_id + charge_id only (for payments matching)
    -- This ensures we match all payments for a claim_id + charge_id, regardless of cpt_code/modifiers
    -- Only include claims that passed the type filter
    SELECT DISTINCT ON (fct.claim_id, fct.charge_id)
        fct.claim_id,
        fct.charge_id,
        fct.claim_from_date,
        fct.revenue_code
    FROM filtered_claims_by_type fct
    ORDER BY fct.claim_id, fct.charge_id, fct.claim_from_date DESC
),
claim_units_summary AS (
    -- Sum units, expected_amount, and amount (total_claims) for all unique claims (by claim_id + charge_id + cpt_code + modifiers) 
    -- that share the same claim_id + charge_id
    -- This matches the dashboard API logic which counts all unique claims
    SELECT 
        fct.claim_id,
        fct.charge_id,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0))::numeric as total_units,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected_amount,
        SUM(COALESCE(nc.amount, 0))::numeric as total_claims_amount
    FROM filtered_claims_by_type fct
    INNER JOIN collaboratemd.new_claim nc ON fct.claim_id = nc.claim_id
        AND COALESCE(fct.charge_id, '') = COALESCE(nc.charge_id, '')
        AND fct.cpt_code = nc.cpt_code
    GROUP BY fct.claim_id, fct.charge_id
),
payments_deduplicated AS (
    -- First deduplicate payments by unique constraint to avoid summing duplicate charge_amount
    -- Filter out payments with 0 insurance_paid_a
    SELECT DISTINCT ON (
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
        p.*
    FROM collaboratemd.new_payment p
    INNER JOIN unique_claims_by_charge ucc ON p.claim_id = ucc.claim_id
        AND COALESCE(p.charge_id, '') = ucc.charge_id
    WHERE 
        -- Filter out payments with 0 insurance_paid_a
        p.insurance_paid_a != 0
        -- Note: Claim type filtering is already applied in filtered_claims_by_type CTE above
        -- Search filter (optional) - search across multiple fields
        AND ($5::text IS NULL OR $5::text = '' OR 
            p.claim_id::text ILIKE '%' || $5::text || '%' OR
            p.charge_id::text ILIKE '%' || $5::text || '%' OR
            p.patient_id::text ILIKE '%' || $5::text || '%' OR
            p.patient_name ILIKE '%' || $5::text || '%' OR
            p.payment_check ILIKE '%' || $5::text || '%' OR
            p.payer_id::text ILIKE '%' || $5::text || '%' OR
            p.payment_name ILIKE '%' || $5::text || '%' OR
            p.cpt_code ILIKE '%' || $5::text || '%' OR
            p.revenue_code ILIKE '%' || $5::text || '%' OR
            p.credit_id::text ILIKE '%' || $5::text || '%' OR
            p.remark ILIKE '%' || $5::text || '%' OR
            p.unpaid_reason ILIKE '%' || $5::text || '%')
    ORDER BY 
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment,
        p.payment_received_date DESC
),
check_number_counts_all AS (
    -- Count distinct check numbers from ALL payments (not deduplicated) for each claim_id + charge_id
    -- This matches the logic used in GetPaymentChecksByClaimAndCharge to ensure counts match
    SELECT
        p.claim_id,
        COALESCE(p.charge_id, '') AS charge_id,
        COUNT(DISTINCT p.payment_check) FILTER (WHERE p.payment_check IS NOT NULL AND p.payment_check <> '') AS total_check_count
    FROM collaboratemd.new_payment p
    INNER JOIN unique_claims_by_charge ucc ON p.claim_id = ucc.claim_id
        AND COALESCE(p.charge_id, '') = ucc.charge_id
    WHERE
        -- Filter by year if provided (0 = all years) using claim_from_date (matching payment-checks query)
        ($1::int = 0 OR EXTRACT(YEAR FROM ucc.claim_from_date) = $1::int)
        -- Filter by month if provided (0 = all months) using claim_from_date (matching payment-checks query)
        AND ($2::int = 0 OR EXTRACT(MONTH FROM ucc.claim_from_date) = $2::int)
    GROUP BY
        p.claim_id,
        COALESCE(p.charge_id, '')
),
payments_grouped AS (
    -- Group payments by unique claim_id + charge_id (matching claims API logic)
    -- Sum all payments for each unique claim_id + charge_id combination (after deduplication)
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        -- Sum all payments for this claim_id + charge_id (after deduplication)
        SUM(pd.insurance_paid_a)::numeric as total_paid_amount,
        -- Use MIN for charge_amount (same value for duplicates, don't sum - fixes duplicate charge_amount issue)
        MIN(pd.charge_amount)::numeric as total_claim_charge,
        -- MIN insurance_adjustment for this claim_id + charge_id
        MIN(pd.insurance_adjustment)::numeric as insurance_adjustment,
        -- Get expected_amount, amount (total_claims), and units from claim units summary (sums all unique claims by cpt_code/modifiers)
        -- Since cus is already grouped by claim_id + charge_id, we use MAX to get the single value
        MAX(cus.total_expected_amount)::numeric as total_expected_amount,
        MAX(cus.total_claims_amount)::numeric as total_claims_amount,
        MAX(cus.total_units)::numeric as units,
        -- Take first value for other fields (for display purposes)
        MIN(pd.patient_id) as patient_id,
        MIN(pd.patient_name) as patient_name,
        MIN(pd.payment_name) as insurance_name,
        MIN(pd.payer_id) as insurance_id,
        MIN(pd.payment_received_date) as payment_date,
        MIN(pd.deposit_date) as posted_date,
        MIN(pd.charge_from_date) as charge_date,
        MIN(pd.payment_entered_date) as bpr_date,
        -- Use check count from all payments (not deduplicated) to match payment-checks API
        COALESCE(MAX(cnc.total_check_count), 0)::text as trn,
        -- Also include total_check_number as integer for frontend display
        COALESCE(MAX(cnc.total_check_count), 0)::int as total_check_number,
        MIN(pd.remark) as remark,
        MIN(pd.unpaid_reason) as unpaid_reason,
        MIN(pd.revenue_code) as revenue_code,
        MIN(pd.cpt_code) as cpt_code,
        MIN(pd.credit_id) as credit_id,
        -- Arrays - return empty array for now (some payments have modifiers, others don't)
        ARRAY[]::text[] as modifiers,
        -- Use claim_from_date for month calculation to match dashboard API (same as total expected)
        EXTRACT(MONTH FROM ucc.claim_from_date) as month_number,
        TO_CHAR(ucc.claim_from_date, 'Month') as month_name
    FROM payments_deduplicated pd
    INNER JOIN unique_claims_by_charge ucc ON pd.claim_id = ucc.claim_id
        AND COALESCE(pd.charge_id, '') = ucc.charge_id
    INNER JOIN claim_units_summary cus ON pd.claim_id = cus.claim_id
        AND COALESCE(pd.charge_id, '') = cus.charge_id
    LEFT JOIN check_number_counts_all cnc ON pd.claim_id = cnc.claim_id
        AND COALESCE(pd.charge_id, '') = cnc.charge_id
        -- Match by claim_id + charge_id only (not cpt_code/modifiers) since payments are grouped by claim_id + charge_id
    GROUP BY 
        pd.claim_id,
        COALESCE(pd.charge_id, ''),
        EXTRACT(MONTH FROM ucc.claim_from_date),
        TO_CHAR(ucc.claim_from_date, 'Month')
)
SELECT 
    CONCAT(pg.claim_id, '-', pg.charge_id) as id,
    pg.claim_id,
    pg.patient_id,
    pg.patient_name,
    pg.total_claim_charge,
    pg.total_paid_amount,
    '' as claim_status, -- Not available in new_payment
    pg.insurance_name,
    pg.insurance_id,
    pg.payment_date,
    pg.posted_date,
    pg.charge_date,
    pg.bpr_date,
    pg.trn,
    pg.month_number,
    pg.month_name,
    pg.charge_id,
    pg.cpt_code,
    pg.modifiers,
    pg.insurance_adjustment,
    pg.credit_id,
    pg.remark,
    pg.unpaid_reason,
    pg.revenue_code,
    pg.total_expected_amount,
    pg.total_claims_amount,
    (pg.total_expected_amount - pg.total_paid_amount)::numeric as total_outstanding_amount,
    pg.units,
    pg.total_check_number,
    false as is_refund, -- Not meaningful when grouped by claim_id + charge_id
    false as is_provider_adjustment, -- Not meaningful when grouped by claim_id + charge_id
    false as is_era_adjustment -- Not meaningful when grouped by claim_id + charge_id
FROM payments_grouped pg
WHERE 
    -- Claims vs Expected filter
    -- claimsGreater: total_claims_amount (from amount field) > total_paid_amount
    -- expectedGreater: total_expected_amount > total_paid_amount
    (
        $9::text IS NULL OR $9::text = '' OR $9::text = 'all' OR
        ($9::text = 'claimsGreater' AND pg.total_claims_amount > pg.total_paid_amount) OR
        ($9::text = 'expectedGreater' AND pg.total_expected_amount > pg.total_paid_amount)
    )
    -- Payment status filter
    AND (
        $10::text IS NULL OR $10::text = '' OR $10::text = 'all' OR
        ($10::text = 'overpaid' AND (pg.total_paid_amount - pg.total_expected_amount) > 0) OR
        ($10::text = 'underpaid' AND (pg.total_expected_amount - pg.total_paid_amount) > 0) OR
        ($10::text = 'fullyPaid' AND ABS(pg.total_paid_amount - pg.total_expected_amount) < 0.01)
    )
ORDER BY 
    pg.claim_id,
    pg.charge_id,
    pg.payment_date DESC
LIMIT $6::int OFFSET $7::int;

-- name: GetDashboardPaymentsCount :one
-- Get total count of grouped payments by claim_id + charge_id (matching dashboard API logic)
-- Counts unique payments by claim_id + charge_id matching filters
-- IMPORTANT: Payments are filtered by claim_from_date (matching dashboard API) to match total expected calculations
-- IMPORTANT: Payments with 0 insurance_paid_a are excluded (matching GetDashboardPaymentsPaginated logic)
-- $1::int = year (0 = all years)
-- $2::int = month (0 = all months)
-- $3::boolean = exclude_peer (exclude payments with CPT code H0038)
-- $4::boolean = peer_only (include only payments with CPT code H0038)
-- $5::text = search (optional search term)
-- $6::text = claim_type filter (NULL/empty = all types, 'inpatient' = IN only, 'outpatient' = OUT only)
-- $7::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $8::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
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
        nc.claim_from_date,
        nc.revenue_code,
        nc.cpt_code as claim_cpt_code
    FROM collaboratemd.new_claim nc
    INNER JOIN filtered_claims fc ON nc.claim_id = fc.claim_id
    ORDER BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), ''), nc.claim_from_date DESC
),
all_claim_categories AS (
    -- Get all categories for each claim (a claim can have multiple categories)
    -- This matches the materialized view logic in 000006_create_dashboard_materialized_view.up.sql
    SELECT DISTINCT
        uc.claim_id,
        uc.charge_id,
        uc.cpt_code,
        uc.modifiers_str,
        uc.claim_from_date,
        uc.revenue_code,
        CASE 
            WHEN uc.revenue_code IN ('0128', '0126') 
              OR uc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 'IN'
            WHEN uc.cpt_code = 'H0038' THEN 'PEER'
            ELSE 'OUT'
        END AS charge_category,
        CASE 
            WHEN uc.revenue_code IN ('0128', '0126') 
              OR uc.cpt_code IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036') THEN 1
            WHEN uc.cpt_code = 'H0038' THEN 2
            ELSE 3
        END AS category_priority
    FROM unique_claims uc
),
claim_primary_category AS (
    -- Determine primary category for each claim using priority: IN (1) > PEER (2) > OUT (3)
    -- This matches the materialized view logic
    SELECT DISTINCT ON (claim_id, charge_id)
        claim_id,
        charge_id,
        cpt_code,
        modifiers_str,
        claim_from_date,
        revenue_code,
        CASE 
            WHEN MIN(category_priority) OVER (PARTITION BY claim_id) = 1 THEN 'inpatient'
            WHEN MIN(category_priority) OVER (PARTITION BY claim_id) = 2 THEN 'peer'
            ELSE 'outpatient'
        END as determined_claim_type
    FROM all_claim_categories
    ORDER BY claim_id, charge_id, category_priority ASC
),
filtered_claims_by_type AS (
    -- Apply claim type filter (if provided) - same logic as claims API
    SELECT 
        claim_id,
        charge_id,
        cpt_code,
        modifiers_str,
        claim_from_date,
        revenue_code,
        determined_claim_type
    FROM claim_primary_category
    WHERE 
        -- Filter by claim type (if provided) - match EXACT logic from claims API
        ($6::text IS NULL OR $6::text = '' OR LOWER(determined_claim_type) = LOWER($6::text))
        -- Exclude peer claims if exclude_peer is true
        AND NOT ($3::boolean = true AND determined_claim_type = 'peer')
        -- Include only peer claims if peer_only is true
        AND (($4::boolean IS NULL OR $4::boolean = false) OR ($4::boolean = true AND determined_claim_type = 'peer'))
),
unique_claims_by_charge AS (
    -- Get unique claims by claim_id + charge_id only (for payments matching)
    -- This ensures we match all payments for a claim_id + charge_id, regardless of cpt_code/modifiers
    -- Only include claims that passed the type filter
    SELECT DISTINCT ON (fct.claim_id, fct.charge_id)
        fct.claim_id,
        fct.charge_id,
        fct.claim_from_date,
        fct.revenue_code
    FROM filtered_claims_by_type fct
    ORDER BY fct.claim_id, fct.charge_id, fct.claim_from_date DESC
),
claim_units_summary AS (
    -- Sum units, expected_amount, and amount (total_claims) for all unique claims (by claim_id + charge_id + cpt_code + modifiers) 
    -- that share the same claim_id + charge_id
    -- This matches the dashboard API logic which counts all unique claims
    SELECT 
        fct.claim_id,
        fct.charge_id,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected_amount,
        SUM(COALESCE(nc.amount, 0))::numeric as total_claims_amount
    FROM filtered_claims_by_type fct
    INNER JOIN collaboratemd.new_claim nc ON fct.claim_id = nc.claim_id
        AND COALESCE(fct.charge_id, '') = COALESCE(nc.charge_id, '')
        AND fct.cpt_code = nc.cpt_code
    GROUP BY fct.claim_id, fct.charge_id
),
payments_deduplicated AS (
    -- First deduplicate payments by unique constraint to avoid summing duplicate charge_amount
    -- Filter out payments with 0 insurance_paid_a
    SELECT DISTINCT ON (
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
        p.claim_id,
        p.charge_id,
        p.insurance_paid_a
    FROM collaboratemd.new_payment p
    INNER JOIN unique_claims_by_charge ucc ON p.claim_id = ucc.claim_id
        AND COALESCE(p.charge_id, '') = ucc.charge_id
    WHERE 
        -- Filter out payments with 0 insurance_paid_a
        p.insurance_paid_a != 0
        -- Note: Claim type filtering is already applied in filtered_claims_by_type CTE above
        -- Search filter (optional) - search across multiple fields
        AND ($5::text IS NULL OR $5::text = '' OR 
            p.claim_id::text ILIKE '%' || $5::text || '%' OR
            p.charge_id::text ILIKE '%' || $5::text || '%' OR
            p.patient_id::text ILIKE '%' || $5::text || '%' OR
            p.patient_name ILIKE '%' || $5::text || '%' OR
            p.payment_check ILIKE '%' || $5::text || '%' OR
            p.payer_id::text ILIKE '%' || $5::text || '%' OR
            p.payment_name ILIKE '%' || $5::text || '%' OR
            p.cpt_code ILIKE '%' || $5::text || '%' OR
            p.revenue_code ILIKE '%' || $5::text || '%' OR
            p.credit_id::text ILIKE '%' || $5::text || '%' OR
            p.remark ILIKE '%' || $5::text || '%' OR
            p.unpaid_reason ILIKE '%' || $5::text || '%')
    ORDER BY 
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment,
        p.payment_received_date DESC
),
payments_grouped AS (
    -- Group payments by unique claim_id + charge_id (matching claims API logic)
    -- Sum all payments for each unique claim_id + charge_id combination (after deduplication)
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        -- Sum all payments for this claim_id + charge_id (after deduplication)
        SUM(pd.insurance_paid_a)::numeric as total_paid_amount,
        -- Get expected_amount and amount (total_claims)
        MAX(cus.total_expected_amount)::numeric as total_expected_amount,
        MAX(cus.total_claims_amount)::numeric as total_claims_amount
    FROM payments_deduplicated pd
    INNER JOIN unique_claims_by_charge ucc ON pd.claim_id = ucc.claim_id
        AND COALESCE(pd.charge_id, '') = ucc.charge_id
    INNER JOIN claim_units_summary cus ON pd.claim_id = cus.claim_id
        AND COALESCE(pd.charge_id, '') = cus.charge_id
    GROUP BY 
        pd.claim_id,
        COALESCE(pd.charge_id, '')
)
SELECT COUNT(*)
FROM payments_grouped pg
WHERE 
    -- Claims vs Expected filter
    -- claimsGreater: total_claims_amount (from amount field) > total_paid_amount
    -- expectedGreater: total_expected_amount > total_paid_amount
    (
        $7::text IS NULL OR $7::text = '' OR $7::text = 'all' OR
        ($7::text = 'claimsGreater' AND pg.total_claims_amount > pg.total_paid_amount) OR
        ($7::text = 'expectedGreater' AND pg.total_expected_amount > pg.total_paid_amount)
    )
    -- Payment status filter
    AND (
        $8::text IS NULL OR $8::text = '' OR $8::text = 'all' OR
        ($8::text = 'overpaid' AND (pg.total_paid_amount - pg.total_expected_amount) > 0) OR
        ($8::text = 'underpaid' AND (pg.total_expected_amount - pg.total_paid_amount) > 0) OR
        ($8::text = 'fullyPaid' AND ABS(pg.total_paid_amount - pg.total_expected_amount) < 0.01)
    );

