-- name: GetDashboardClaimsByYear :many
-- Get all claims for a specific year with payment information for outstanding calculation
-- OPTIMIZED: Uses ROW_NUMBER() window function instead of DISTINCT ON for better performance
-- Returns DISTINCT rows from new_claim table (unique by claim_id, charge_id, cpt_code, modifiers) to match claims API
-- $1::int = year
WITH year_claims AS (
    -- First, filter claims by year to get only relevant claim_ids
    SELECT DISTINCT claim_id
    FROM collaboratemd.new_claim nc
    WHERE nc.claim_entered_date >= DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1))
      AND nc.claim_entered_date < DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year'
),
ranked_claims_with_aggregations AS (
    -- Use ROW_NUMBER() and window functions together for better performance
    -- Calculate aggregations using window functions before filtering
    SELECT 
        nc.claim_id,
        nc.charge_id,
        nc.cpt_code,
        nc.modifiers,
        nc.revenue_code,
        nc.claim_type,
        nc.claim_entered_date,
        nc.claim_from_date,
        nc.claim_to_date,
        nc.patient_id,
        nc.patient_name,
        nc.primary_payer,
        nc.secondary_payer,
        nc.tertiary_payer,
        nc.units,
        nc.facility_pos_code,
        nc.charge_pos_code,
        nc.expected_amount,
        nc.amount,
        EXTRACT(MONTH FROM nc.claim_entered_date) as month_number,
        TO_CHAR(nc.claim_entered_date, 'Month') as month_name,
        ROW_NUMBER() OVER (
            PARTITION BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), '')
            ORDER BY nc.claim_entered_date DESC
        ) as rn,
        -- Use window functions to aggregate across all rows in partition (before filtering)
        SUM(COALESCE(nc.amount, 0)) OVER (
            PARTITION BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), '')
        )::numeric as total_claims_sum,
        SUM(COALESCE(nc.expected_amount, 0)) OVER (
            PARTITION BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), '')
        )::numeric as total_expected_sum,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0)) OVER (
            PARTITION BY nc.claim_id, COALESCE(nc.charge_id, ''), COALESCE(nc.cpt_code, ''), COALESCE(array_to_string(nc.modifiers, ','), '')
        )::numeric as total_units_sum
    FROM collaboratemd.new_claim nc
    INNER JOIN year_claims yc ON nc.claim_id = yc.claim_id
),
unique_claims_base AS (
    -- Get only the first row per unique key (rn = 1) with pre-calculated aggregations
    SELECT 
        claim_id,
        charge_id,
        cpt_code,
        modifiers,
        revenue_code,
        claim_type,
        claim_entered_date,
        claim_from_date,
        claim_to_date,
        patient_id,
        patient_name,
        primary_payer,
        secondary_payer,
        tertiary_payer,
        units,
        facility_pos_code,
        charge_pos_code,
        expected_amount,
        amount,
        month_number,
        month_name,
        total_claims_sum,
        total_expected_sum,
        total_units_sum
    FROM ranked_claims_with_aggregations
    WHERE rn = 1
),
claim_aggregations AS (
    -- Aggregations are already calculated in unique_claims_base via window functions
    -- Just select them for consistency with the rest of the query
    SELECT 
        ucb.claim_id,
        COALESCE(ucb.charge_id, '') as charge_id,
        COALESCE(ucb.cpt_code, '') as cpt_code,
        COALESCE(array_to_string(ucb.modifiers, ','), '') as modifiers_str,
        ucb.total_claims_sum,
        ucb.total_expected_sum,
        ucb.total_units_sum
    FROM unique_claims_base ucb
),
payment_totals_by_unique_claim AS (
    -- Sum all insurance_paid_a from payments for each unique claim and charge_id
    -- Match payments to specific charges using both claim_id and charge_id
    SELECT 
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id,
        SUM(p.insurance_paid_a)::numeric as total_payment_sum
    FROM collaboratemd.new_payment p
    INNER JOIN year_claims yc ON p.claim_id = yc.claim_id
    GROUP BY p.claim_id, COALESCE(p.charge_id, '')
),
payment_totals_by_unique_payment AS (
    -- First, group payments by unique constraint and sum amounts for duplicates
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
        EXTRACT(MONTH FROM p.payment_entered_date) as payment_month,
        -- Sum insurance_paid_a for duplicate payments (same unique constraint)
        -- Include both peer and non-peer payments
        SUM(p.insurance_paid_a) as total_billed
    FROM collaboratemd.new_payment p
    INNER JOIN year_claims yc ON p.claim_id = yc.claim_id
    -- Include all payments (both peer and non-peer)
    -- Group by unique constraint to sum duplicate payments
    GROUP BY 
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment,
        EXTRACT(MONTH FROM p.payment_entered_date)
),
payment_totals_by_claim_monthly AS (
    -- Then aggregate by claim_id, charge_id and payment month (matching claim month)
    -- Separate peer and non-peer payments, match payments to specific charges
    SELECT 
        ptup.claim_id,
        COALESCE(ptup.charge_id, '') as charge_id,
        ptup.payment_month,
        -- Non-peer billed: sum for non-peer payments only
        SUM(CASE 
            WHEN ptup.cpt_code != 'H0038' OR ptup.cpt_code IS NULL THEN ptup.total_billed
            ELSE 0
        END) as non_peer_billed,
        -- Peer billed: sum for peer payments only
        SUM(CASE 
            WHEN ptup.cpt_code = 'H0038' THEN ptup.total_billed
            ELSE 0
        END) as peer_billed
    FROM payment_totals_by_unique_payment ptup
    INNER JOIN collaboratemd.new_claim nc ON ptup.claim_id = nc.claim_id
        AND COALESCE(ptup.charge_id, '') = COALESCE(nc.charge_id, '')
        AND ptup.payment_month = EXTRACT(MONTH FROM nc.claim_entered_date)
    GROUP BY ptup.claim_id, COALESCE(ptup.charge_id, ''), ptup.payment_month
),
claim_charge_counts AS (
    -- Count distinct (claim_id, charge_id) pairs for each claim_id
    -- OPTIMIZED: Use unique_claims_base to reduce data volume
    SELECT 
        ucb.claim_id,
        COUNT(DISTINCT COALESCE(ucb.charge_id, '')) as total_charges
    FROM unique_claims_base ucb
    GROUP BY ucb.claim_id
),
claims_with_aggregations AS (
    SELECT 
        ucb.*,
        -- Use aggregated values from claim_aggregations (sum of all duplicate claims)
        -- If join fails, use the single row's value (no duplicates case)
        COALESCE(ca.total_claims_sum, COALESCE(ucb.amount, 0))::numeric as total_claims_aggregated,
        COALESCE(ca.total_expected_sum, COALESCE(ucb.expected_amount, 0))::numeric as total_expected_aggregated,
        COALESCE(ca.total_units_sum, COALESCE(NULLIF(TRIM(ucb.units), '')::numeric, 0))::numeric as total_units_aggregated,
        -- Use total payment from payments (sum of all insurance_paid_a for this claim_id)
        COALESCE(ptuc.total_payment_sum, 0)::numeric as total_payment_sum,
        COALESCE(ccc.total_charges, 1)::int as total_charges
    FROM unique_claims_base ucb
    LEFT JOIN claim_aggregations ca ON ucb.claim_id = ca.claim_id
        AND COALESCE(ucb.charge_id, '') = ca.charge_id
        AND COALESCE(ucb.cpt_code, '') = ca.cpt_code
        -- Compare modifiers arrays properly - use array_to_string for reliable comparison
        AND COALESCE(array_to_string(ucb.modifiers, ','), '') = ca.modifiers_str
    LEFT JOIN payment_totals_by_unique_claim ptuc ON ucb.claim_id = ptuc.claim_id
        AND COALESCE(ucb.charge_id, '') = ptuc.charge_id
    LEFT JOIN claim_charge_counts ccc ON ucb.claim_id = ccc.claim_id
)
SELECT 
    uca.claim_id,
    COALESCE(uca.charge_id, '') as charge_id,
    -- Determine claim type: IN if has revenue codes 0126/0128 OR special inpatient CPT codes
    -- Special inpatient CPT codes: H2019, H0017, H0019, 99235, H0010, H2036
    CASE 
        WHEN uca.cpt_code = 'H0038' THEN 'peer'
        WHEN (uca.revenue_code IN ('0126', '0128') OR UPPER(uca.cpt_code) IN ('H2019', 'H0017', 'H0019', '99235', 'H0010', 'H2036')) AND uca.cpt_code != 'H0038' THEN 'inpatient'
        WHEN uca.cpt_code IS NOT NULL AND uca.cpt_code != '' AND uca.cpt_code != 'H0038' THEN 'outpatient'
        ELSE LOWER(uca.claim_type)
    END as claim_type,
    0::numeric as total_charge, -- Keep for backward compatibility
    -- For peer claims (H0038): use peer_billed (actual peer payment amounts)
    -- For non-peer claims: use non_peer_billed only if payment month matches claim month
    -- Don't use fallback to all payments - only count payments from the same month as the claim
    COALESCE(
        CASE 
            WHEN uca.cpt_code = 'H0038' THEN COALESCE(pt.peer_billed, 0)
            ELSE COALESCE(pt.non_peer_billed, 0)
        END, 
        0
    )::numeric as total_billed,
    uca.claim_entered_date as claim_submission_date,
    uca.claim_from_date as service_date_from,
    uca.claim_to_date as service_date_to,
    uca.patient_id,
    uca.patient_name,
    uca.claim_type as type_of_service, -- Keep original claim_type for backward compatibility
    COALESCE((uca.primary_payer->>'Name')::text, '') as primary_payer_name,
    COALESCE((uca.primary_payer->>'Id')::text, '') as primary_payer_id,
    COALESCE((uca.secondary_payer->>'Name')::text, '') as secondary_payer_name,
    COALESCE((uca.secondary_payer->>'Id')::text, '') as secondary_payer_id,
    1::int as charge_count, -- Each row is one charge
    ARRAY[uca.cpt_code]::text[] as cpt_codes, -- Single CPT code as array
    ARRAY[uca.revenue_code]::text[] as revenue_codes, -- Single revenue code as array
    uca.modifiers::text[] as modifiers,
    uca.units::text as units,
    uca.facility_pos_code::text as facility_pos_code,
    uca.charge_pos_code::text as charge_pos_code,
    COALESCE((uca.tertiary_payer->>'Name')::text, '') as tertiary_payer_name,
    COALESCE((uca.tertiary_payer->>'Id')::text, '') as tertiary_payer_id,
    uca.expected_amount::numeric as total_expected,
    uca.amount::numeric as total_claim,
    -- Add aggregated values (same as GetDashboardClaimsPaginated)
    uca.total_claims_aggregated as total_claims,
    uca.total_expected_aggregated as total_expected_agg,
    uca.total_payment_sum as total_payment,
    uca.total_units_aggregated as total_units,
    -- Outstanding = total_expected - total_billed (for all claims including peer)
    CASE 
        WHEN uca.cpt_code = 'H0038' THEN (uca.expected_amount::numeric - COALESCE(pt.peer_billed, 0))
        ELSE (uca.expected_amount::numeric - COALESCE(pt.non_peer_billed, 0))
    END::numeric as outstanding_amount,
    uca.month_number,
    uca.month_name,
    uca.total_charges
FROM claims_with_aggregations uca
LEFT JOIN payment_totals_by_claim_monthly pt ON uca.claim_id = pt.claim_id
    AND COALESCE(uca.charge_id, '') = pt.charge_id
    AND uca.month_number = pt.payment_month
    -- Join payments for both peer and non-peer claims, matching on claim_id and charge_id
ORDER BY uca.claim_entered_date DESC, uca.claim_id, uca.cpt_code;

-- name: GetDashboardMonthlyView :many
-- Get monthly aggregated view of claims data
-- $1::int = year
-- $2::boolean = exclude_peer (exclude claims with CPT code H0038)
WITH monthly_claims AS (
    SELECT 
        EXTRACT(MONTH FROM nc.claim_entered_date) as month_number,
        TO_CHAR(nc.claim_entered_date, 'Month') as month_name,
        COUNT(DISTINCT (nc.claim_id, nc.cpt_code)) as total_claim,
        SUM(nc.expected_amount)::numeric as total_expected
    FROM collaboratemd.new_claim nc
    WHERE EXTRACT(YEAR FROM nc.claim_entered_date) = $1::int
        -- Exclude peer claims (H0038) if exclude_peer is true
        AND NOT ($2::boolean = true AND nc.cpt_code = 'H0038')
    GROUP BY EXTRACT(MONTH FROM nc.claim_entered_date), TO_CHAR(nc.claim_entered_date, 'Month')
),
monthly_payments AS (
    SELECT 
        EXTRACT(MONTH FROM p.payment_entered_date) as month_number,
        SUM(p.insurance_paid_a) as total_billed
    FROM collaboratemd.new_payment p
    WHERE EXTRACT(YEAR FROM p.payment_entered_date) = $1::int
        AND (p.cpt_code != 'H0038' OR p.cpt_code IS NULL)  -- Exclude peer payments (peer claims and payments are independent)
        AND NOT ($2::boolean = true AND p.cpt_code = 'H0038')
    GROUP BY EXTRACT(MONTH FROM p.payment_entered_date)
)
SELECT 
    mc.month_number,
    mc.month_name,
    mc.total_claim,
    COALESCE(mp.total_billed, 0)::numeric as total_billed,
    -- Outstanding = total_expected - total_billed
    (mc.total_expected - COALESCE(mp.total_billed, 0))::numeric as total_outstanding,
    mc.total_expected
FROM monthly_claims mc
LEFT JOIN monthly_payments mp ON mc.month_number = mp.month_number
ORDER BY mc.month_number;