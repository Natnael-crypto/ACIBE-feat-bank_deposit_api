-- name: GetBankDepositClaimsPaginated :many
-- Get paginated claims filtered by year and month for bank deposit data
-- Returns claims that have payments linked to insurance checks deposited in the specified month
-- Uses the same filtering logic as GetBankDepositData to ensure consistency
-- 
-- IMPORTANT: Filtering is based ONLY on insurance check deposit_date (check numbers), NOT on claim_from_date
-- claim_from_date is ONLY used for ordering/pagination, never for filtering
-- 
-- Logic (matches GetBankDepositData exactly):
-- 1. Get insurance checks deposited in each month (based on deposit_date) - THIS IS THE ONLY FILTER
-- 2. Join with new_payment using insurance_check.check_number = new_payment.payment_check AND matching deposit_date
-- 3. Get unique (claim_id, charge_id) pairs from those payments
-- 4. Match those (claim_id, charge_id) pairs in new_claim to get ALL charge lines (regardless of claim_from_date month)
--    - If charge_id is empty, match all charge lines for that claim_id
--    - If charge_id exists, match the specific (claim_id, charge_id) pair
-- 5. Aggregate all charge lines for each unique (claim_id, charge_id) pair
-- 6. Return paginated claims (ordered by claim_from_date for pagination cursor, but NOT filtered by it)
--
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for claim_id)
-- $4::int = limit
-- $5::text = cursor (optional, for pagination: format "claim_from_date|claim_id")
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        EXTRACT(MONTH FROM ic.deposit_date)::int as month_number,
        TO_CHAR(ic.deposit_date, 'Month') as month_name,
        ic.check_number,
        ic.check_amount,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        -- Filter by year using deposit_date
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        -- Filter by month if specified (using deposit_date month)
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_linked_to_checks AS (
    -- 2) Get payments linked to insurance check numbers from step 1
    -- Join insurance_check.check_number = new_payment.payment_check
    -- Also match deposit_date to ensure payment is linked to the correct check deposit
    -- Only include payments that have payment_check and claim_id (charge_id may be NULL or empty)
    SELECT
        icbm.month_number,
        icbm.month_name,
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id
    FROM insurance_checks_by_month icbm
    INNER JOIN collaboratemd.new_payment p
        ON p.payment_check = icbm.check_number
    WHERE 
        p.payment_check IS NOT NULL 
        AND p.payment_check != ''
        AND p.claim_id IS NOT NULL
        AND TRIM(p.claim_id) != ''
),
unique_claims AS (
    -- 3) Get unique claim_id from those payments (per month)
    -- Count and list by claim_id only (no duplication based on charge_id)
    SELECT DISTINCT
        pltc.month_number,
        pltc.month_name,
        pltc.claim_id
    FROM payments_linked_to_checks pltc
    WHERE pltc.claim_id IS NOT NULL
        AND TRIM(pltc.claim_id) != ''
        -- Apply search filter early (before matching in new_claim)
        AND ($3::text IS NULL OR $3::text = '' OR pltc.claim_id::text ILIKE '%' || $3::text || '%')
),
payment_charge_ids AS (
    -- Get all (claim_id, charge_id) pairs from payments for matching charge lines in totals
    -- This is used for calculating Total Claims, Total Expected, Total Payment
    SELECT DISTINCT
        pltc.claim_id,
        COALESCE(NULLIF(TRIM(pltc.charge_id), ''), '') as charge_id
    FROM payments_linked_to_checks pltc
    WHERE pltc.claim_id IS NOT NULL
        AND TRIM(pltc.claim_id) != ''
),
paginated_claims_with_dates AS (
    -- Get claim_from_date for pagination ordering ONLY (NOT for filtering)
    -- Paginate unique claim_id (matching the count query logic)
    -- Use the most recent claim_from_date for each claim_id for ordering
    -- NOTE: We get ALL charge lines regardless of claim_from_date month - filtering is only by check deposit_date
    SELECT 
        uc.claim_id,
        uc.month_number,
        uc.month_name,
        COALESCE(MAX(nc.claim_from_date), '1900-01-01'::date) as claim_from_date
    FROM unique_claims uc
    LEFT JOIN collaboratemd.new_claim nc 
        ON nc.claim_id = uc.claim_id
        -- NO FILTER ON claim_from_date - we get ALL charge lines for these claims
    GROUP BY uc.claim_id, uc.month_number, uc.month_name
),
paginated_claims AS (
    -- Paginate unique claim_id with ordering
    -- This matches the count query which counts unique claim_id
    SELECT 
        pcwd.claim_id,
        pcwd.claim_from_date,
        pcwd.month_number,
        pcwd.month_name
    FROM paginated_claims_with_dates pcwd
    WHERE 
        -- Cursor-based pagination (if cursor provided)
        -- Cursor format: "claim_from_date|claim_id"
        (
            $5::text IS NULL OR $5::text = '' OR
            TO_CHAR(pcwd.claim_from_date, 'YYYY-MM-DD') < SPLIT_PART($5::text, '|', 1)
            OR (TO_CHAR(pcwd.claim_from_date, 'YYYY-MM-DD') = SPLIT_PART($5::text, '|', 1)
                AND pcwd.claim_id < SPLIT_PART($5::text, '|', 2))
        )
    ORDER BY pcwd.claim_from_date DESC, pcwd.claim_id
    LIMIT $4::int
),
matched_charge_lines AS (
    -- Match charge lines in new_claim using payment charge_id matching logic
    -- Get unique charge lines (claim_id + charge_id) and sum their values
    -- If payment charge_id is empty, match all charge lines for that claim_id
    -- If payment charge_id exists, match the specific (claim_id, charge_id) pair
    -- Aggregate by unique (claim_id, charge_id) to avoid double counting
    SELECT 
        pc.claim_id,
        COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') as charge_id,
        SUM(COALESCE(nc.amount, 0))::numeric as amount,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as expected_amount,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0))::numeric as units
    FROM paginated_claims pc
    INNER JOIN payment_charge_ids pci ON pci.claim_id = pc.claim_id
    INNER JOIN collaboratemd.new_claim nc
        ON nc.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all charge lines for the claim
            (pci.charge_id = '')
            -- If payment charge_id exists, match the specific charge line
            OR (pci.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = pci.charge_id)
        )
        -- NO WHERE clause filtering by claim_from_date - filtering is ONLY by check deposit_date
    GROUP BY pc.claim_id, COALESCE(NULLIF(TRIM(nc.charge_id), ''), '')
),
claim_representative_values AS (
    -- Get representative values from new_claim for matched charge lines only
    -- Join with matched_charge_lines to ensure we only get values for charge lines that have payments
    SELECT DISTINCT ON (mcl.claim_id)
        mcl.claim_id,
        nc.claim_from_date,
        nc.claim_to_date,
        nc.claim_entered_date,
        nc.patient_id,
        nc.patient_name,
        nc.claim_type,
        nc.primary_payer,
        nc.secondary_payer,
        nc.tertiary_payer,
        nc.charge_id,
        nc.cpt_code,
        nc.modifiers
    FROM matched_charge_lines mcl
    INNER JOIN collaboratemd.new_claim nc
        ON nc.claim_id = mcl.claim_id
        AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = mcl.charge_id
    ORDER BY mcl.claim_id, nc.claim_from_date DESC
),
claim_aggregations AS (
    -- Aggregate all charges for each paginated claim_id
    -- IMPORTANT: This must return ONE row per row in paginated_claims
    -- Use pre-aggregated values from matched_charge_lines (already unique by claim_id + charge_id)
    -- IMPORTANT: NO filtering by claim_from_date - we get ALL charge lines for these claims
    SELECT 
        pc.claim_id,
        pc.month_number,
        pc.month_name,
        pc.claim_from_date,  -- Use the claim_from_date from paginated_claims for ordering
        -- Get representative values from matched charge lines
        COALESCE(crv.claim_from_date, pc.claim_from_date) as claim_from_date_agg,
        crv.claim_to_date,
        crv.claim_entered_date,
        crv.patient_id,
        crv.patient_name,
        crv.claim_type as type_of_service,
        crv.primary_payer,
        crv.secondary_payer,
        crv.tertiary_payer,
        -- Get first charge_id for display
        crv.charge_id as first_charge_id,
        -- Aggregate arrays from all matched charge lines
        COALESCE((SELECT ARRAY_AGG(DISTINCT nc.cpt_code) 
                  FROM matched_charge_lines mcl2
                  INNER JOIN collaboratemd.new_claim nc
                      ON nc.claim_id = mcl2.claim_id
                      AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = mcl2.charge_id
                  WHERE mcl2.claim_id = pc.claim_id
                    AND nc.cpt_code IS NOT NULL 
                    AND nc.cpt_code != ''), ARRAY[]::text[]) as cpt_codes,
        -- Get modifiers from first claim with modifiers
        COALESCE(crv.modifiers, 
                 (SELECT nc2.modifiers 
                  FROM collaboratemd.new_claim nc2 
                  WHERE nc2.claim_id = pc.claim_id 
                    AND nc2.modifiers IS NOT NULL 
                    AND array_length(nc2.modifiers, 1) > 0
                  ORDER BY nc2.claim_from_date DESC 
                  LIMIT 1)) as modifiers,
        -- Sum pre-aggregated amounts from matched charge lines (already unique by claim_id + charge_id)
        COALESCE(SUM(mcl.amount), 0)::numeric as total_claims_sum,
        COALESCE(SUM(mcl.expected_amount), 0)::numeric as total_expected_sum,
        COALESCE(SUM(mcl.units), 0)::numeric as total_units_sum,
        COUNT(DISTINCT mcl.charge_id)::int as total_charges
    FROM paginated_claims pc
    LEFT JOIN matched_charge_lines mcl ON mcl.claim_id = pc.claim_id
    LEFT JOIN claim_representative_values crv ON crv.claim_id = pc.claim_id
    GROUP BY pc.claim_id, pc.month_number, pc.month_name, pc.claim_from_date,
             crv.claim_from_date, crv.claim_to_date, crv.claim_entered_date,
             crv.patient_id, crv.patient_name, crv.claim_type,
             crv.primary_payer, crv.secondary_payer, crv.tertiary_payer,
             crv.charge_id, crv.modifiers
),
payment_totals_by_claim AS (
    -- Aggregate total payments per claim_id using charge_id matching logic
    -- Match payments that have the same (claim_id, charge_id) as in payment_charge_ids
    SELECT 
        pc.claim_id,
        COALESCE(SUM(p.insurance_paid_a), 0)::numeric as total_payment_sum
    FROM paginated_claims pc
    LEFT JOIN payment_charge_ids pci ON pci.claim_id = pc.claim_id
    LEFT JOIN collaboratemd.new_payment p
        ON p.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all payments for the claim
            (pci.charge_id = '' AND COALESCE(p.charge_id, '') = '')
            -- If payment charge_id exists, match payments with that specific charge_id
            OR (pci.charge_id != '' AND COALESCE(p.charge_id, '') = pci.charge_id)
        )
    GROUP BY pc.claim_id
),
check_number_counts AS (
    -- Count unique check numbers per claim_id
    SELECT 
        pc.claim_id,
        COUNT(DISTINCT p.payment_check)::int as total_check_number
    FROM paginated_claims pc
    LEFT JOIN payment_charge_ids pci ON pci.claim_id = pc.claim_id
    LEFT JOIN collaboratemd.new_payment p
        ON p.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all payments for the claim
            (pci.charge_id = '' AND COALESCE(p.charge_id, '') = '')
            -- If payment charge_id exists, match payments with that specific charge_id
            OR (pci.charge_id != '' AND COALESCE(p.charge_id, '') = pci.charge_id)
        )
        AND p.payment_check IS NOT NULL 
        AND p.payment_check != ''
    GROUP BY pc.claim_id
)
-- Final SELECT - one row per unique claim_id with aggregated values
SELECT
    ca.claim_id,
    'outpatient' as claim_type,  -- Default type, can be determined from cpt_codes if needed
    0::numeric as total_charge,
    0::numeric as total_billed,
    COALESCE(ca.claim_from_date_agg, ca.claim_from_date) as claim_submission_date,
    COALESCE(ca.claim_from_date_agg, ca.claim_from_date) as service_date_from,
    ca.claim_to_date as service_date_to,
    ca.patient_id,
    ca.patient_name,
    ca.type_of_service,
    COALESCE((ca.primary_payer->>'Name')::text, '') as primary_payer_name,
    COALESCE((ca.primary_payer->>'Id')::text, '') as primary_payer_id,
    COALESCE((ca.secondary_payer->>'Name')::text, '') as secondary_payer_name,
    COALESCE((ca.secondary_payer->>'Id')::text, '') as secondary_payer_id,
    COALESCE(ca.total_charges, 0)::int as charge_count,
    ca.cpt_codes as cpt_codes,
    CASE WHEN ca.modifiers IS NULL THEN ARRAY[]::text[] ELSE ca.modifiers END as modifiers,
    NULL::text as units,
    NULL::text as facility_pos_code,
    NULL::text as charge_pos_code,
    COALESCE((ca.tertiary_payer->>'Name')::text, '') as tertiary_payer_name,
    COALESCE((ca.tertiary_payer->>'Id')::text, '') as tertiary_payer_id,
    COALESCE(ca.total_expected_sum, 0)::numeric as total_expected,
    COALESCE(ca.total_claims_sum, 0)::numeric as total_claims,
    COALESCE(ca.total_expected_sum, 0)::numeric as total_expected_agg,
    COALESCE(ptc.total_payment_sum, 0)::numeric as total_payment,
    COALESCE(ca.total_units_sum, 0)::numeric as total_units,
    (COALESCE(ca.total_expected_sum, 0) - COALESCE(ptc.total_payment_sum, 0))::numeric as outstanding_amount,
    ca.month_number,
    ca.month_name,
    COALESCE(ca.total_charges, 0)::int as total_charges,
    ca.claim_entered_date as claim_entered_date,
    COALESCE(cnc.total_check_number, 0)::int as total_check_number
FROM claim_aggregations ca
LEFT JOIN payment_totals_by_claim ptc ON ca.claim_id = ptc.claim_id
LEFT JOIN check_number_counts cnc ON ca.claim_id = cnc.claim_id
ORDER BY COALESCE(ca.claim_from_date_agg, ca.claim_from_date) DESC, ca.claim_id;

-- name: GetBankDepositClaimsCount :one
-- Get total count of unique claim_id filtered by year and month for bank deposit data
-- Uses the same filtering logic as GetBankDepositData
-- 
-- Logic:
-- 1. Get insurance checks deposited in each month (based on deposit_date)
-- 2. Join with new_payment using insurance_check.check_number = new_payment.payment_check
-- 3. Get unique (claim_id, charge_id) pairs from those payments
-- 4. Match those in new_claim to get unique claim_id from new_claim table
-- 5. Count unique claim_id = total_claims_number
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for claim_id)
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        EXTRACT(MONTH FROM ic.deposit_date)::int as month_number,
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_linked_to_checks AS (
    -- 2) Get payments linked to insurance check numbers from step 1
    -- Join insurance_check.check_number = new_payment.payment_check
    -- Also match deposit_date to ensure payment is linked to the correct check deposit
    -- Only include payments that have payment_check and claim_id (charge_id may be NULL or empty)
    SELECT
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id
    FROM insurance_checks_by_month icbm
    INNER JOIN collaboratemd.new_payment p
        ON p.payment_check = icbm.check_number
    WHERE 
        p.payment_check IS NOT NULL 
        AND p.payment_check != ''
        AND p.claim_id IS NOT NULL
        AND TRIM(p.claim_id) != ''
),
payment_charge_ids AS (
    -- 3) Get unique (claim_id, charge_id) pairs from those payments
    SELECT DISTINCT
        pltc.claim_id,
        COALESCE(NULLIF(TRIM(pltc.charge_id), ''), '') as charge_id
    FROM payments_linked_to_checks pltc
    WHERE 
        -- Search filter (optional)
        ($3::text IS NULL OR $3::text = '' OR pltc.claim_id::text ILIKE '%' || $3::text || '%')
),
unique_claims_from_new_claim AS (
    -- 4) Get unique claim_id from new_claim that match the payment charge_ids
    -- This ensures we count claim_ids that actually exist in new_claim table
    SELECT DISTINCT
        nc.claim_id
    FROM payment_charge_ids pci
    INNER JOIN collaboratemd.new_claim nc
        ON nc.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all charge lines for the claim
            (pci.charge_id = '')
            -- If payment charge_id exists, match the specific charge line
            OR (pci.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = pci.charge_id)
        )
)
-- 5) Count unique claim_id from new_claim = total_claims_number
SELECT COUNT(*)::bigint as total_count
FROM unique_claims_from_new_claim;

-- name: GetBankDepositClaimsTotalAmount :one
-- Get total claims amount for claims filtered by year and month for bank deposit data
-- Uses the same filtering logic as GetBankDepositData
-- 
-- Logic:
-- 1. Get insurance checks deposited in each month (based on deposit_date)
-- 2. Join with new_payment using insurance_check.check_number = new_payment.payment_check
-- 3. Get unique (claim_id, charge_id) pairs from those payments (for matching charge lines)
-- 4. Match those (claim_id, charge_id) pairs in new_claim to get specific charge lines
-- 5. Sum amount from those specific charge lines = total_claims_amount
-- Note: Uses charge_id matching for totals, but counts by claim_id only
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for claim_id)
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        EXTRACT(MONTH FROM ic.deposit_date)::int as month_number,
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_linked_to_checks AS (
    -- 2) Get payments linked to insurance check numbers from step 1
    -- Join insurance_check.check_number = new_payment.payment_check
    -- Also match deposit_date to ensure payment is linked to the correct check deposit
    -- Only include payments that have payment_check and claim_id (charge_id may be NULL or empty)
    SELECT
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id
    FROM insurance_checks_by_month icbm
    INNER JOIN collaboratemd.new_payment p
        ON p.payment_check = icbm.check_number
        AND p.deposit_date = icbm.deposit_date
    WHERE 
        p.payment_check IS NOT NULL 
        AND p.payment_check != ''
        AND p.claim_id IS NOT NULL
        AND TRIM(p.claim_id) != ''
),
unique_claim_lines AS (
    -- 3) Get unique (claim_id, charge_id) pairs from those payments
    -- Normalize charge_id: treat NULL and empty strings as empty string
    SELECT DISTINCT
        pltc.claim_id,
        COALESCE(NULLIF(TRIM(pltc.charge_id), ''), '') as charge_id
    FROM payments_linked_to_checks pltc
    WHERE 
        -- Search filter (optional)
        ($3::text IS NULL OR $3::text = '' OR pltc.claim_id::text ILIKE '%' || $3::text || '%')
)
-- 4-5) Match those (claim_id, charge_id) pairs in new_claim and sum amount from those specific charge lines
-- If charge_id is empty, match all charge lines for that claim_id
-- If charge_id exists, match the specific (claim_id, charge_id) pair
SELECT COALESCE(SUM(COALESCE(nc.amount, 0)), 0)::numeric as total_amount
FROM unique_claim_lines ucl
INNER JOIN collaboratemd.new_claim nc
    ON nc.claim_id = ucl.claim_id
    AND (
        -- If payment charge_id is empty, match all charge lines for the claim
        (ucl.charge_id = '')
        -- If payment charge_id exists, match the specific charge line
        OR (ucl.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = ucl.charge_id)
    );

-- name: GetBankDepositClaimsTotalExpectedAmount :one
-- Get total expected amount for claims filtered by year and month for bank deposit data
-- Uses the same filtering logic as GetBankDepositData
-- 
-- Logic:
-- 1. Get insurance checks deposited in each month (based on deposit_date)
-- 2. Join with new_payment using insurance_check.check_number = new_payment.payment_check
-- 3. Get unique (claim_id, charge_id) pairs from those payments (for matching charge lines)
-- 4. Match those (claim_id, charge_id) pairs in new_claim to get specific charge lines
-- 5. Sum expected_amount from those specific charge lines = total_expected_amount
-- Note: Uses charge_id matching for totals, but counts by claim_id only
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for claim_id)
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        EXTRACT(MONTH FROM ic.deposit_date)::int as month_number,
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_linked_to_checks AS (
    -- 2) Get payments linked to insurance check numbers from step 1
    -- Join insurance_check.check_number = new_payment.payment_check
    -- Also match deposit_date to ensure payment is linked to the correct check deposit
    -- Only include payments that have payment_check and claim_id (charge_id may be NULL or empty)
    SELECT
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id
    FROM insurance_checks_by_month icbm
    INNER JOIN collaboratemd.new_payment p
        ON p.payment_check = icbm.check_number
    WHERE 
        p.payment_check IS NOT NULL 
        AND p.payment_check != ''
        AND p.claim_id IS NOT NULL
        AND TRIM(p.claim_id) != ''
),
unique_claim_lines AS (
    -- 3) Get unique (claim_id, charge_id) pairs from those payments
    -- Normalize charge_id: treat NULL and empty strings as empty string
    SELECT DISTINCT
        pltc.claim_id,
        COALESCE(NULLIF(TRIM(pltc.charge_id), ''), '') as charge_id
    FROM payments_linked_to_checks pltc
    WHERE 
        -- Search filter (optional)
        ($3::text IS NULL OR $3::text = '' OR pltc.claim_id::text ILIKE '%' || $3::text || '%')
)
-- 4-5) Match those (claim_id, charge_id) pairs in new_claim and sum expected_amount from those specific charge lines
-- If charge_id is empty, match all charge lines for that claim_id
-- If charge_id exists, match the specific (claim_id, charge_id) pair
SELECT COALESCE(SUM(COALESCE(nc.expected_amount, 0)), 0)::numeric as total_expected_amount
FROM unique_claim_lines ucl
INNER JOIN collaboratemd.new_claim nc
    ON nc.claim_id = ucl.claim_id
    AND (
        -- If payment charge_id is empty, match all charge lines for the claim
        (ucl.charge_id = '')
        -- If payment charge_id exists, match the specific charge line
        OR (ucl.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = ucl.charge_id)
    );

-- name: GetBankDepositAppliedPaymentsPaginated :many
-- Get paginated payments for bank deposit applied amount
-- Returns all payments linked to insurance check numbers deposited in the specified month
-- Uses the same filtering logic as GetBankDepositData applied_amounts_by_month to ensure consistency
-- 
-- Logic:
-- 1. Get insurance checks deposited in each month (based on deposit_date)
-- 2. Get all payments linked to those insurance check numbers
-- 3. Return paginated payments
--
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for claim_id, check_number, etc.)
-- $4::int = limit
-- $5::text = cursor (optional, for pagination: format "payment_date|claim_id|charge_id")
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        EXTRACT(MONTH FROM ic.deposit_date)::int as month_number,
        TO_CHAR(ic.deposit_date, 'Month') as month_name,
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        -- Filter by year using deposit_date
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        -- Filter by month if specified (using deposit_date month)
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_filtered AS (
    -- 2) Get all payments linked to those insurance check numbers
    -- Match by check_number only (not deposit_date) for better performance
    -- Use DISTINCT to remove duplicates based on unique constraint columns
    SELECT DISTINCT
        p.payment_entered_date,
        p.claim_id,
        p.credit_id,
        p.charge_id,
        p.cpt_code,
        p.payment_check,
        p.is_refund,
        p.is_provider_adjustment,
        p.is_era_adjustment,
        p.insurance_paid_a,
        p.charge_amount,
        p.insurance_adjustment,
        p.patient_id,
        p.patient_name,
        p.payment_name,
        p.payer_id,
        p.payment_received_date,
        p.deposit_date,
        p.charge_from_date,
        p.remark,
        p.unpaid_reason,
        p.revenue_code,
        icbm.month_number,
        icbm.month_name
    FROM collaboratemd.new_payment p
    INNER JOIN insurance_checks_by_month icbm
        ON p.payment_check = icbm.check_number
    WHERE 
        -- Filter out payments with 0 insurance_paid_a
        p.insurance_paid_a != 0
        -- Search filter (optional) - apply early for better performance
        AND ($3::text IS NULL OR $3::text = '' OR 
            p.claim_id ILIKE '%' || $3::text || '%' OR
            p.payment_check ILIKE '%' || $3::text || '%')
),
payments_grouped AS (
    -- Group payments by claim_id + charge_id
    -- Optimize by doing grouping first, then join with new_claim for expected amounts
    SELECT 
        pf.claim_id,
        COALESCE(pf.charge_id, '') as charge_id,
        -- Sum all payments for this claim_id + charge_id
        SUM(pf.insurance_paid_a)::numeric as total_paid_amount,
        -- Use MIN for charge_amount (same value for duplicates, don't sum)
        MIN(pf.charge_amount)::numeric as total_claim_charge,
        -- Sum insurance_adjustment
        SUM(pf.insurance_adjustment)::numeric as insurance_adjustment,
        -- Take first value for other fields
        MIN(pf.patient_id) as patient_id,
        MIN(pf.patient_name) as patient_name,
        MIN(pf.payment_name) as insurance_name,
        MIN(pf.payer_id) as insurance_id,
        MIN(pf.payment_received_date) as payment_date,
        MIN(pf.deposit_date) as posted_date,
        MIN(pf.charge_from_date) as charge_date,
        MIN(pf.payment_entered_date) as bpr_date,
        MIN(pf.payment_check) as trn,
        MIN(pf.remark) as remark,
        MIN(pf.unpaid_reason) as unpaid_reason,
        MIN(pf.revenue_code) as revenue_code,
        MIN(pf.cpt_code) as cpt_code,
        MIN(pf.credit_id) as credit_id,
        -- Arrays - return empty array for now
        ARRAY[]::text[] as modifiers,
        MIN(pf.month_number) as month_number,
        MIN(pf.month_name) as month_name
    FROM payments_filtered pf
    GROUP BY 
        pf.claim_id,
        COALESCE(pf.charge_id, '')
),
claim_units_summary AS (
    -- Get expected amounts and units for claims that have payments (optimized - join after grouping)
    SELECT 
        pg.claim_id,
        pg.charge_id,
        SUM(COALESCE(NULLIF(TRIM(nc.units), '')::numeric, 0))::numeric as total_units,
        SUM(COALESCE(nc.expected_amount, 0))::numeric as total_expected_amount
    FROM payments_grouped pg
    INNER JOIN collaboratemd.new_claim nc 
        ON nc.claim_id = pg.claim_id
        AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = pg.charge_id
    GROUP BY pg.claim_id, pg.charge_id
),
check_number_counts AS (
    -- Count unique check numbers per (claim_id, charge_id) pair
    -- Count from ALL payments for this (claim_id, charge_id) pair, not just from payments_filtered
    -- This matches the logic used in the regular payments API
    SELECT 
        pg.claim_id,
        pg.charge_id,
        COUNT(DISTINCT p.payment_check)::int as total_check_number
    FROM payments_grouped pg
    LEFT JOIN collaboratemd.new_payment p
        ON p.claim_id = pg.claim_id
        AND COALESCE(p.charge_id, '') = pg.charge_id
    WHERE p.payment_check IS NOT NULL 
        AND p.payment_check != ''
    GROUP BY pg.claim_id, pg.charge_id
),
payments_with_expected AS (
    -- Combine grouped payments with expected amounts
    SELECT 
        pg.*,
        COALESCE(cus.total_expected_amount, 0)::numeric as total_expected_amount,
        COALESCE(cus.total_units, 0)::numeric as units,
        COALESCE(cnc.total_check_number, 0)::int as total_check_number
    FROM payments_grouped pg
    LEFT JOIN claim_units_summary cus 
        ON pg.claim_id = cus.claim_id
        AND pg.charge_id = cus.charge_id
    LEFT JOIN check_number_counts cnc
        ON pg.claim_id = cnc.claim_id
        AND pg.charge_id = cnc.charge_id
)
SELECT 
    CONCAT(pwe.claim_id, '-', pwe.charge_id) as id,
    pwe.claim_id,
    pwe.patient_id,
    pwe.patient_name,
    pwe.total_claim_charge,
    pwe.total_paid_amount,
    '' as claim_status,
    pwe.insurance_name,
    pwe.insurance_id,
    pwe.payment_date,
    pwe.posted_date,
    pwe.charge_date,
    pwe.bpr_date,
    pwe.trn,
    pwe.month_number,
    pwe.month_name,
    pwe.charge_id,
    pwe.cpt_code,
    pwe.modifiers,
    pwe.insurance_adjustment,
    pwe.credit_id,
    pwe.remark,
    pwe.unpaid_reason,
    pwe.revenue_code,
    pwe.total_expected_amount,
    pwe.units,
    pwe.total_check_number,
    false as is_refund,
    false as is_provider_adjustment,
    false as is_era_adjustment
FROM payments_with_expected pwe
WHERE 
    -- Cursor-based pagination (if cursor provided)
    -- Cursor format: "payment_date|claim_id|charge_id"
    -- Apply cursor filter AFTER grouping to paginate grouped results
    (
        $5::text IS NULL OR $5::text = '' OR
        TO_CHAR(pwe.payment_date, 'YYYY-MM-DD') < SPLIT_PART($5::text, '|', 1)
        OR (TO_CHAR(pwe.payment_date, 'YYYY-MM-DD') = SPLIT_PART($5::text, '|', 1)
            AND pwe.claim_id < SPLIT_PART($5::text, '|', 2))
        OR (TO_CHAR(pwe.payment_date, 'YYYY-MM-DD') = SPLIT_PART($5::text, '|', 1)
            AND pwe.claim_id = SPLIT_PART($5::text, '|', 2)
            AND pwe.charge_id < SPLIT_PART($5::text, '|', 3))
    )
ORDER BY 
    pwe.payment_date DESC,
    pwe.posted_date DESC,
    pwe.claim_id,
    pwe.charge_id
LIMIT $4::int;

-- name: GetBankDepositAppliedPaymentsCount :one
-- Get total count of payments for bank deposit applied amount
-- Uses the same filtering logic as GetBankDepositAppliedPaymentsPaginated
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term)
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
),
payments_filtered AS (
    -- 2) Get all payments linked to those insurance check numbers (deduplicated)
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
        COALESCE(p.charge_id, '') as charge_id
    FROM collaboratemd.new_payment p
    INNER JOIN insurance_checks_by_month icbm
        ON p.payment_check = icbm.check_number
        AND p.deposit_date = icbm.deposit_date
    WHERE 
        p.insurance_paid_a != 0
        AND ($3::text IS NULL OR $3::text = '' OR 
            p.claim_id ILIKE '%' || $3::text || '%' OR
            p.payment_check ILIKE '%' || $3::text || '%')
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
    -- Group by claim_id + charge_id (matching paginated query logic)
    SELECT DISTINCT
        pf.claim_id,
        pf.charge_id
    FROM payments_filtered pf
)
-- Count distinct payments grouped by claim_id + charge_id
SELECT COUNT(*)::bigint as total_count
FROM payments_grouped;

-- name: GetBankDepositAppliedPaymentsTotalAmount :one
-- Get total applied amount (sum of insurance_paid_a) for payments linked to insurance check numbers
-- Uses the same filtering logic as GetBankDepositData applied_amounts_by_month
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term)
WITH date_range AS (
    -- Date range for filtering by year and month
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_by_month AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    SELECT 
        ic.check_number,
        ic.deposit_date
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        AND ic.deposit_date IS NOT NULL
        -- Exclude checks deposited before March 1st, 2025
        AND ic.deposit_date >= '2025-03-01'::date
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
)
-- 2) Sum insurance_paid_a for all payments linked to those check numbers
SELECT COALESCE(SUM(COALESCE(p.insurance_paid_a, 0)), 0)::numeric as total_applied_amount
FROM insurance_checks_by_month icbm
INNER JOIN collaboratemd.new_payment p
    ON p.payment_check = icbm.check_number
    AND p.deposit_date = icbm.deposit_date
WHERE 
    -- Search filter (optional)
    ($3::text IS NULL OR $3::text = '' OR 
        p.claim_id ILIKE '%' || $3::text || '%' OR
        p.payment_check ILIKE '%' || $3::text || '%');

