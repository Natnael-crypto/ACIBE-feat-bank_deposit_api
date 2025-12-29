-- name: GetClaimCharges :many
-- Get all charges (claim_id, cpt_code pairs) for a specific claim
-- Returns distinct charges by (claim_id, charge_id, cpt_code, modifiers)
-- OPTIMIZED: Uses date ranges instead of EXTRACT, direct array comparison instead of array_to_string
-- $1::text = claim_id
-- $2::int = year (optional, for filtering by year, 0 = all years)
-- $3::int = month (optional, for filtering by month, 0 = all months)
WITH date_range AS (
    -- Pre-compute date range for efficient filtering (allows index usage)
    SELECT 
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) END as year_start,
        CASE WHEN $2::int = 0 THEN NULL ELSE DATE_TRUNC('year', MAKE_DATE($2::int, 1, 1)) + INTERVAL '1 year' END as year_end,
        CASE WHEN $3::int = 0 THEN NULL ELSE $3::int END as month_filter
),
charge_payments AS (
    -- Sum insurance_paid_a from payments for each charge (by claim_id, charge_id, cpt_code, modifiers)
    -- Also get remark and unpaid_reason from the most recent payment
    -- OPTIMIZED: Use direct array comparison instead of array_to_string
    SELECT 
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id,
        p.cpt_code,
        COALESCE(p.modifiers, ARRAY[]::text[]) as modifiers,
        SUM(p.insurance_paid_a)::numeric as insurance_paid_amount,
        -- Get remark, unpaid_reason, and adjustment_reasons from the most recent payment (by payment_entered_date)
        (array_agg(p.remark ORDER BY p.payment_entered_date DESC NULLS LAST, p.payment_received_date DESC NULLS LAST))[1] as remark,
        (array_agg(p.unpaid_reason ORDER BY p.payment_entered_date DESC NULLS LAST, p.payment_received_date DESC NULLS LAST))[1] as unpaid_reason,
        (array_agg(p.adjustment_reasons ORDER BY p.payment_entered_date DESC NULLS LAST, p.payment_received_date DESC NULLS LAST))[1] as adjustment_reasons
    FROM collaboratemd.new_payment p
    WHERE p.claim_id = $1::text
    GROUP BY p.claim_id, COALESCE(p.charge_id, ''), p.cpt_code, COALESCE(p.modifiers, ARRAY[]::text[])
),
check_number_counts AS (
    -- Count unique check numbers per charge (claim_id, charge_id)
    SELECT 
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id,
        COUNT(DISTINCT p.payment_check)::int as total_check_number_count
    FROM collaboratemd.new_payment p
    WHERE p.claim_id = $1::text
        AND p.payment_check IS NOT NULL 
        AND p.payment_check != ''
    GROUP BY p.claim_id, COALESCE(p.charge_id, '')
),
ranked_claims AS (
    -- Get unique charges (take first row per unique charge: claim_id, charge_id, cpt_code, modifiers)
    -- OPTIMIZED: Use date range filtering instead of EXTRACT for better index usage
    -- Uses claim_from_date instead of claim_entered_date for filtering and ordering
    -- Returns ALL charges for the claim, regardless of category
    SELECT 
        nc.*,
        ROW_NUMBER() OVER (
            PARTITION BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(nc.modifiers, ARRAY[]::text[])
            ORDER BY nc.claim_from_date DESC
        ) as rn
    FROM collaboratemd.new_claim nc
    CROSS JOIN date_range dr
    WHERE nc.claim_id = $1::text
        -- Filter by year if provided (0 = all years) - use date range for index usage
        AND (dr.year_start IS NULL OR (nc.claim_from_date >= dr.year_start AND nc.claim_from_date < dr.year_end))
        -- Filter by month if provided (0 = all months)
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM nc.claim_from_date) = dr.month_filter)
)
SELECT 
    nc.claim_id,
    nc.cpt_code,
    COALESCE(nc.charge_id, '')::text as charge_id,
    COALESCE(nc.modifiers, ARRAY[]::text[]) as modifiers,
    nc.amount::numeric as amount,
    nc.expected_amount::numeric as expected_amount,
    COALESCE(cp.insurance_paid_amount, 0)::numeric as insurance_paid_amount,
    COALESCE(nc.revenue_code, '')::text as revenue_code,
    COALESCE(nc.units, '')::text as units,
    -- Claim details (from first row per unique charge)
    nc.patient_id,
    nc.patient_name,
    nc.claim_from_date,
    nc.claim_to_date,
    nc.claim_entered_date,
    nc.claim_type,
    COALESCE((nc.primary_payer->>'Name')::text, '') as primary_payer_name,
    COALESCE((nc.primary_payer->>'Id')::text, '') as primary_payer_id,
    COALESCE((nc.secondary_payer->>'Name')::text, '') as secondary_payer_name,
    COALESCE((nc.secondary_payer->>'Id')::text, '') as secondary_payer_id,
    COALESCE((nc.tertiary_payer->>'Name')::text, '') as tertiary_payer_name,
    COALESCE((nc.tertiary_payer->>'Id')::text, '') as tertiary_payer_id,
    nc.facility_pos_code,
    nc.charge_pos_code,
    COALESCE(cp.remark, '')::text as remark,
    COALESCE(cp.unpaid_reason, '')::text as unpaid_reason,
    COALESCE(cp.adjustment_reasons, '')::text as adjustment_reasons,
    EXTRACT(YEAR FROM nc.claim_from_date)::int as year,
    EXTRACT(MONTH FROM nc.claim_from_date)::int as month,
    -- Count of unique check numbers for this charge
    COALESCE(cnc.total_check_number_count, 0)::int as total_check_number_count
FROM ranked_claims nc
LEFT JOIN charge_payments cp ON nc.claim_id = cp.claim_id
    AND COALESCE(nc.charge_id, '') = cp.charge_id
    AND nc.cpt_code = cp.cpt_code
    -- OPTIMIZED: Use direct array comparison instead of array_to_string (much faster)
    AND COALESCE(nc.modifiers, ARRAY[]::text[]) = cp.modifiers
LEFT JOIN check_number_counts cnc ON nc.claim_id = cnc.claim_id
    AND COALESCE(nc.charge_id, '') = cnc.charge_id
WHERE nc.rn = 1
ORDER BY nc.cpt_code, COALESCE(nc.revenue_code, ''), nc.amount;
