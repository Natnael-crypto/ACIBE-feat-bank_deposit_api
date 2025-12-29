-- name: GetPaymentsByCheckNumber :many
-- Get all payments for a specific check number (TRN) filtered by year only (no month filter)
-- Returns paginated payments grouped by claim_id + charge_id
-- $1::text = check_number (payment_check/TRN)
-- $2::int = year (required, 0 = all years)
-- $3::int = limit
-- $4::int = offset
-- $5::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $6::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
-- $7::text = search (optional search term)
WITH year_date_range AS (
    -- Date range for filtering by year only
    SELECT 
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) END as year_start,
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) + INTERVAL '1 year' END as year_end
),
filtered_claims AS (
    -- Filter claims by year using claim_from_date (matching dashboard API logic)
    SELECT DISTINCT claim_id
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE 
        -- Filter by year using claim_from_date (same as dashboard API)
        (ydr.year_start IS NULL OR (nc.claim_from_date >= ydr.year_start AND nc.claim_from_date < ydr.year_end))
),
payments_deduplicated AS (
    -- First deduplicate payments by unique constraint
    -- Filter by check number and filter out payments with 0 insurance_paid_a
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
    INNER JOIN filtered_claims fc ON p.claim_id = fc.claim_id
    WHERE 
        -- Filter by exact check number match
        p.payment_check = $1::text
        -- Filter out payments with 0 insurance_paid_a
        AND p.insurance_paid_a != 0
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
claim_units_summary AS (
    -- Get expected amounts, claims amounts (from amount field), and units for claims that have payments with this check number
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0))::numeric as total_units,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected_amount,
        SUM(COALESCE(nc.amount, 0))::numeric as total_claims_amount
    FROM payments_deduplicated pd
    INNER JOIN collaboratemd.new_claim nc ON pd.claim_id = nc.claim_id
        AND COALESCE(pd.charge_id, '') = COALESCE(nc.charge_id, '')
    GROUP BY pd.claim_id, COALESCE(pd.charge_id, '')
),
payments_grouped AS (
    -- Group payments by claim_id + charge_id
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        -- Sum all payments for this claim_id + charge_id (after deduplication)
        SUM(pd.insurance_paid_a)::numeric as total_paid_amount,
        -- Use MIN for charge_amount (same value for duplicates, don't sum)
        MIN(pd.charge_amount)::numeric as total_claim_charge,
        -- MIN insurance_adjustment for this claim_id + charge_id
        MIN(pd.insurance_adjustment)::numeric as insurance_adjustment,
        -- Get expected_amount, claims_amount, and units from claim units summary
        MIN(cus.total_expected_amount)::numeric as total_expected_amount,
        MIN(cus.total_claims_amount)::numeric as total_claims_amount,
        MIN(cus.total_units)::numeric as units,
        -- Calculate total outstanding amount (expected - paid)
        (MIN(cus.total_expected_amount)::numeric - SUM(pd.insurance_paid_a)::numeric)::numeric as total_outstanding_amount,
        -- Take first value for other fields (for display purposes)
        MIN(pd.patient_id) as patient_id,
        MIN(pd.patient_name) as patient_name,
        MIN(pd.payment_name) as insurance_name,
        MIN(pd.payer_id) as insurance_id,
        MIN(pd.payment_received_date) as payment_date,
        MIN(pd.deposit_date) as posted_date,
        MIN(pd.charge_from_date) as charge_date,
        MIN(pd.payment_entered_date) as bpr_date,
        MIN(pd.payment_check)::text as trn,
        MIN(pd.remark) as remark,
        MIN(pd.unpaid_reason) as unpaid_reason,
        MIN(pd.adjustment_reasons) as adjustment_reasons,
        MIN(pd.revenue_code) as revenue_code,
        MIN(pd.cpt_code) as cpt_code,
        MIN(pd.credit_id) as credit_id,
        -- Arrays - return empty array for now
        ARRAY[]::text[] as modifiers,
        -- Get month from claim_from_date
        EXTRACT(MONTH FROM nc.claim_from_date) as month_number,
        TO_CHAR(nc.claim_from_date, 'Month') as month_name
    FROM payments_deduplicated pd
    INNER JOIN collaboratemd.new_claim nc ON pd.claim_id = nc.claim_id
        AND COALESCE(pd.charge_id, '') = COALESCE(nc.charge_id, '')
    INNER JOIN claim_units_summary cus ON pd.claim_id = cus.claim_id
        AND COALESCE(pd.charge_id, '') = cus.charge_id
    GROUP BY 
        pd.claim_id,
        COALESCE(pd.charge_id, ''),
        EXTRACT(MONTH FROM nc.claim_from_date),
        TO_CHAR(nc.claim_from_date, 'Month')
)
SELECT 
    CONCAT(pg.claim_id, '-', pg.charge_id) as id,
    pg.claim_id,
    pg.patient_id,
    pg.patient_name,
    pg.total_claim_charge,
    pg.total_paid_amount,
    '' as claim_status,
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
    pg.adjustment_reasons,
    pg.revenue_code,
    pg.total_expected_amount,
    pg.total_claims_amount,
    pg.total_outstanding_amount,
    pg.units,
    false as is_refund,
    false as is_provider_adjustment,
    false as is_era_adjustment
FROM payments_grouped pg
WHERE 
    -- Claims vs Expected filter
    (
        $5::text IS NULL OR $5::text = '' OR $5::text = 'all' OR
        ($5::text = 'claimsGreater' AND pg.total_claims_amount > pg.total_paid_amount) OR
        ($5::text = 'expectedGreater' AND pg.total_expected_amount > pg.total_paid_amount)
    )
    -- Payment status filter
    AND (
        $6::text IS NULL OR $6::text = '' OR $6::text = 'all' OR
        ($6::text = 'overpaid' AND (pg.total_paid_amount - pg.total_expected_amount) > 0) OR
        ($6::text = 'underpaid' AND (pg.total_expected_amount - pg.total_paid_amount) > 0) OR
        ($6::text = 'fullyPaid' AND ABS(pg.total_paid_amount - pg.total_expected_amount) < 0.01)
    )
    -- Search filter (optional) - search across multiple fields
    AND (
        $7::text IS NULL OR $7::text = '' OR
        pg.claim_id::text ILIKE '%' || $7::text || '%' OR
        pg.charge_id::text ILIKE '%' || $7::text || '%' OR
        pg.patient_id::text ILIKE '%' || $7::text || '%' OR
        pg.patient_name ILIKE '%' || $7::text || '%' OR
        pg.insurance_id ILIKE '%' || $7::text || '%' OR
        pg.insurance_name ILIKE '%' || $7::text || '%' OR
        pg.trn ILIKE '%' || $7::text || '%' OR
        pg.cpt_code ILIKE '%' || $7::text || '%' OR
        pg.revenue_code ILIKE '%' || $7::text || '%' OR
        pg.credit_id::text ILIKE '%' || $7::text || '%' OR
        pg.remark ILIKE '%' || $7::text || '%' OR
        pg.unpaid_reason ILIKE '%' || $7::text || '%'
    )
ORDER BY 
    pg.payment_date DESC,
    pg.posted_date DESC
LIMIT $3::int OFFSET $4::int;

-- name: GetPaymentsByCheckNumberCount :one
-- Get total count of payments grouped by claim_id + charge_id for a specific check number filtered by year
-- $1::text = check_number (payment_check/TRN)
-- $2::int = year (required, 0 = all years)
-- $3::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $4::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
-- $5::text = search (optional search term)
WITH year_date_range AS (
    -- Date range for filtering by year only
    SELECT 
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) END as year_start,
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) + INTERVAL '1 year' END as year_end
),
filtered_claims AS (
    -- Filter claims by year using claim_from_date (matching dashboard API logic)
    SELECT DISTINCT claim_id
    FROM collaboratemd.new_claim nc
    CROSS JOIN year_date_range ydr
    WHERE 
        -- Filter by year using claim_from_date (same as dashboard API)
        (ydr.year_start IS NULL OR (nc.claim_from_date >= ydr.year_start AND nc.claim_from_date < ydr.year_end))
),
payments_deduplicated AS (
    -- First deduplicate payments by unique constraint
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
    INNER JOIN filtered_claims fc ON p.claim_id = fc.claim_id
    WHERE 
        -- Filter by exact check number match
        p.payment_check = $1::text
        -- Filter out payments with 0 insurance_paid_a
        AND p.insurance_paid_a != 0
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
claim_units_summary AS (
    -- Get expected amounts, claims amounts, and units for claims that have payments with this check number
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected_amount,
        SUM(COALESCE(nc.amount, 0))::numeric as total_claims_amount
    FROM payments_deduplicated pd
    INNER JOIN collaboratemd.new_claim nc ON pd.claim_id = nc.claim_id
        AND COALESCE(pd.charge_id, '') = COALESCE(nc.charge_id, '')
    GROUP BY pd.claim_id, COALESCE(pd.charge_id, '')
),
payments_grouped AS (
    -- Group payments by claim_id + charge_id
    SELECT 
        pd.claim_id,
        COALESCE(pd.charge_id, '') as charge_id,
        SUM(pd.insurance_paid_a)::numeric as total_paid_amount,
        MIN(cus.total_expected_amount)::numeric as total_expected_amount,
        MIN(cus.total_claims_amount)::numeric as total_claims_amount,
        -- Add fields needed for search filter
        MIN(pd.patient_id) as patient_id,
        MIN(pd.patient_name) as patient_name,
        MIN(pd.payer_id) as insurance_id,
        MIN(pd.payment_name) as insurance_name,
        MIN(pd.payment_check)::text as trn,
        MIN(pd.cpt_code) as cpt_code,
        MIN(pd.revenue_code) as revenue_code,
        MIN(pd.credit_id) as credit_id,
        MIN(pd.remark) as remark,
        MIN(pd.unpaid_reason) as unpaid_reason,
        MIN(pd.adjustment_reasons) as adjustment_reasons
    FROM payments_deduplicated pd
    INNER JOIN claim_units_summary cus ON pd.claim_id = cus.claim_id
        AND COALESCE(pd.charge_id, '') = cus.charge_id
    GROUP BY pd.claim_id, COALESCE(pd.charge_id, '')
)
SELECT COUNT(DISTINCT (pg.claim_id, pg.charge_id))::bigint as total_count
FROM payments_grouped pg
WHERE 
    -- Claims vs Expected filter
    (
        $3::text IS NULL OR $3::text = '' OR $3::text = 'all' OR
        ($3::text = 'claimsGreater' AND pg.total_claims_amount > pg.total_paid_amount) OR
        ($3::text = 'expectedGreater' AND pg.total_expected_amount > pg.total_paid_amount)
    )
    -- Payment status filter
    AND (
        $4::text IS NULL OR $4::text = '' OR $4::text = 'all' OR
        ($4::text = 'overpaid' AND (pg.total_paid_amount - pg.total_expected_amount) > 0) OR
        ($4::text = 'underpaid' AND (pg.total_expected_amount - pg.total_paid_amount) > 0) OR
        ($4::text = 'fullyPaid' AND ABS(pg.total_paid_amount - pg.total_expected_amount) < 0.01)
    )
    -- Search filter (optional) - search across multiple fields
    AND (
        $5::text IS NULL OR $5::text = '' OR
        pg.claim_id::text ILIKE '%' || $5::text || '%' OR
        pg.charge_id::text ILIKE '%' || $5::text || '%' OR
        pg.patient_id::text ILIKE '%' || $5::text || '%' OR
        pg.patient_name ILIKE '%' || $5::text || '%' OR
        pg.insurance_id ILIKE '%' || $5::text || '%' OR
        pg.insurance_name ILIKE '%' || $5::text || '%' OR
        pg.trn ILIKE '%' || $5::text || '%' OR
        pg.cpt_code ILIKE '%' || $5::text || '%' OR
        pg.revenue_code ILIKE '%' || $5::text || '%' OR
        pg.credit_id::text ILIKE '%' || $5::text || '%' OR
        pg.remark ILIKE '%' || $5::text || '%' OR
        pg.unpaid_reason ILIKE '%' || $5::text || '%'
    );

