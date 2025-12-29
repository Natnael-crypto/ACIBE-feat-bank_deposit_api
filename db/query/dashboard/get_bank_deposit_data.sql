-- name: GetBankDepositData :many
-- Get bank deposit data aggregated by month for a specific year
-- Returns: month_number, month_name, total_claims_number, total_claims_amount, total_expected_amount, total_deposit_amount, total_applied_amount
-- 
-- Logic:
-- 1. Get insurance checks deposited in each month (based on deposit_date)
--    - Group by check_number first to handle duplicates (sum check_amount for duplicate check_numbers)
-- 2. Join with new_payment using insurance_check.check_number = new_payment.payment_check to get payments for that month
-- 3. Get unique (claim_id, charge_id) pairs from those payments
-- 4. Count unique (claim_id, charge_id) pairs = total_claims_number
-- 5. Match those (claim_id, charge_id) pairs in new_claim to get specific charge lines
-- 6. Sum amount from those specific charge lines = total_claims_amount
-- 7. Sum expected_amount from those specific charge lines = total_expected_amount
-- 8. Sum insurance_paid_a from payments = total_applied_amount
--    - Group by check_number first, then sum totals to handle duplicate check_numbers properly

-- $1::int = year (required)
WITH date_range AS (
    -- Date range for filtering by year
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end
),
insurance_checks_by_month_raw AS (
    -- 1) Get insurance checks deposited in each month (based on deposit_date)
    -- First get all checks with month info
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
),
insurance_checks_by_month AS (
    -- Group by check_number to handle duplicates - sum check_amount for duplicate check_numbers
    SELECT 
        month_number,
        month_name,
        check_number,
        SUM(COALESCE(check_amount, 0))::numeric as check_amount,
        MIN(deposit_date) as deposit_date
    FROM insurance_checks_by_month_raw
    GROUP BY month_number, month_name, check_number
),
deposit_amounts_by_month AS (
    -- Total deposit amount from insurance checks (already deduplicated by check_number)
    SELECT 
        month_number,
        month_name,
        SUM(COALESCE(check_amount, 0))::numeric as total_deposit_amount
    FROM insurance_checks_by_month
    GROUP BY month_number, month_name
),
payments_linked_to_checks AS (
    -- 2) Get payments linked to insurance check numbers from step 1
    -- Join insurance_check.check_number = new_payment.payment_check
    -- Also match deposit_date to ensure payment is linked to the correct check deposit
    -- Only include payments that have payment_check and claim_id (charge_id may be NULL or empty)
    SELECT
        icbm.month_number,
        icbm.month_name,
        icbm.check_number,
        p.claim_id,
        COALESCE(p.charge_id, '') as charge_id,
        p.insurance_paid_a
    FROM insurance_checks_by_month icbm
    INNER JOIN collaboratemd.new_payment p
        ON p.payment_check = icbm.check_number
    WHERE 
        p.payment_check IS NOT NULL 
        AND p.payment_check != ''
        AND p.claim_id IS NOT NULL
        AND TRIM(p.claim_id) != ''
),
payment_charge_ids_by_month AS (
    -- Get all (claim_id, charge_id) pairs from payments for matching charge lines in totals
    -- This is used for calculating Total Claims, Total Expected (using charge_id matching)
    SELECT DISTINCT
        pltc.month_number,
        pltc.month_name,
        pltc.claim_id,
        COALESCE(NULLIF(TRIM(pltc.charge_id), ''), '') as charge_id
    FROM payments_linked_to_checks pltc
    WHERE pltc.claim_id IS NOT NULL
        AND TRIM(pltc.claim_id) != ''
),
unique_claims_by_month AS (
    -- 3) Get unique claim_id from new_claim that match the payment charge_ids (per month)
    -- This ensures we count claim_ids that actually exist in new_claim table
    -- Count by claim_id only (no duplication based on charge_id)
    SELECT DISTINCT
        pci.month_number,
        pci.month_name,
        nc.claim_id
    FROM payment_charge_ids_by_month pci
    INNER JOIN collaboratemd.new_claim nc
        ON nc.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all charge lines for the claim
            (pci.charge_id = '')
            -- If payment charge_id exists, match the specific charge line
            OR (pci.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = pci.charge_id)
        )
),
claims_count_by_month AS (
    -- 4) Count unique claim_id from new_claim = total_claims_number
    -- Count from unique_claims_by_month (no duplication based on charge_id)
    SELECT 
        ucm.month_number,
        ucm.month_name,
        COUNT(*)::bigint as total_claims_number
    FROM unique_claims_by_month ucm
    GROUP BY ucm.month_number, ucm.month_name
),
unique_claim_lines AS (
    -- 5) Match payment (claim_id, charge_id) to actual service lines in new_claim
    -- Get unique charge lines (claim_id + charge_id) - each counted only once
    -- If payment charge_id is empty, match all charge lines for that claim_id
    -- If payment charge_id exists, match the specific (claim_id, charge_id) pair
    SELECT DISTINCT
        pci.month_number,
        pci.month_name,
        nc.claim_id,
        COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') as charge_id,
        nc.amount,
        nc.expected_amount
    FROM payment_charge_ids_by_month pci
    INNER JOIN collaboratemd.new_claim nc
        ON nc.claim_id = pci.claim_id
        AND (
            -- If payment charge_id is empty, match all charge lines for the claim
            (pci.charge_id = '')
            -- If payment charge_id exists, match the specific charge line
            OR (pci.charge_id != '' AND COALESCE(NULLIF(TRIM(nc.charge_id), ''), '') = pci.charge_id)
        )
),
matched_charge_lines AS (
    -- Aggregate amounts by unique (claim_id, charge_id) to avoid double counting
    SELECT 
        ucl.month_number,
        ucl.month_name,
        ucl.claim_id,
        ucl.charge_id,
        SUM(COALESCE(ucl.amount, 0))::numeric as amount,
        SUM(COALESCE(ucl.expected_amount, 0))::numeric as expected_amount
    FROM unique_claim_lines ucl
    GROUP BY ucl.month_number, ucl.month_name, ucl.claim_id, ucl.charge_id
),
claims_amounts_by_month AS (
    -- 6) Sum amount from unique charge lines = total_claims_amount
    -- 7) Sum expected_amount from unique charge lines = total_expected_amount
    -- Each unique (claim_id, charge_id) is counted only once
    SELECT 
        mcl.month_number,
        mcl.month_name,
        SUM(mcl.amount)::numeric as total_claims_amount,
        SUM(mcl.expected_amount)::numeric as total_expected_amount
    FROM matched_charge_lines mcl
    GROUP BY mcl.month_number, mcl.month_name
),
claims_data_by_month AS (
    -- Combine count and amounts
    SELECT 
        COALESCE(ccm.month_number, cam.month_number) as month_number,
        COALESCE(ccm.month_name, cam.month_name) as month_name,
        COALESCE(ccm.total_claims_number, 0)::bigint as total_claims_number,
        COALESCE(cam.total_claims_amount, 0)::numeric as total_claims_amount,
        COALESCE(cam.total_expected_amount, 0)::numeric as total_expected_amount
    FROM claims_count_by_month ccm
    FULL OUTER JOIN claims_amounts_by_month cam
        ON ccm.month_number = cam.month_number
),
applied_amounts_by_check AS (
    -- Sum insurance_paid_a per check_number first to handle duplicates properly
    -- This ensures each check_number is counted only once
    SELECT
        pltc.month_number,
        pltc.month_name,
        pltc.check_number,
        SUM(COALESCE(pltc.insurance_paid_a, 0))::numeric as applied_amount_per_check
    FROM payments_linked_to_checks pltc
    GROUP BY pltc.month_number, pltc.month_name, pltc.check_number
),
applied_amounts_by_month AS (
    -- 8) Sum insurance_paid_a from payments = total_applied_amount
    -- Sum of insurance_paid_a for all payments linked to insurance check numbers deposited in each month
    -- Now summing from deduplicated check-level totals
    SELECT 
        aac.month_number,
        aac.month_name,
        SUM(COALESCE(aac.applied_amount_per_check, 0))::numeric as total_applied_amount
    FROM applied_amounts_by_check aac
    GROUP BY aac.month_number, aac.month_name
)
-- Final SELECT: Combine all data
SELECT 
    COALESCE(dam.month_number, cdm.month_number, aam.month_number)::int as month_number,
    COALESCE(TRIM(dam.month_name), TRIM(cdm.month_name), TRIM(aam.month_name)) as month_name,
    COALESCE(cdm.total_claims_number, 0)::bigint as total_claims_number,
    COALESCE(cdm.total_claims_amount, 0)::numeric as total_claims_amount,
    COALESCE(cdm.total_expected_amount, 0)::numeric as total_expected_amount,
    COALESCE(dam.total_deposit_amount, 0)::numeric as total_deposit_amount,
    COALESCE(aam.total_applied_amount, 0)::numeric as total_applied_amount
FROM deposit_amounts_by_month dam
FULL OUTER JOIN claims_data_by_month cdm
    ON dam.month_number = cdm.month_number
FULL OUTER JOIN applied_amounts_by_month aam
    ON COALESCE(dam.month_number, cdm.month_number) = aam.month_number
ORDER BY COALESCE(dam.month_number, cdm.month_number, aam.month_number);
