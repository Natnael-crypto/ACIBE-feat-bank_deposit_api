-- name: GetInsuranceChecksPaginated :many
-- Get paginated insurance checks filtered by year and month based on deposit_date
-- Returns insurance checks filtered by deposit_date (matching bank deposit API logic)
-- 
-- Logic:
-- 1. Filter insurance checks by year/month using deposit_date (matching bank deposit API)
-- 2. Apply search filter if provided
-- 3. Return paginated results ordered by deposit_date
--
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term for check_number, check_payer, etc.)
-- $4::int = limit
-- $5::text = cursor (optional, for pagination: format "deposit_date|check_number|id")
WITH date_range AS (
    -- Date range for filtering by year and month based on deposit_date
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
),
insurance_checks_filtered AS (
    -- Filter insurance checks by deposit_date year and month (matching bank deposit API logic)
    SELECT 
        ic.id,
        ic.check_payer,
        ic.check_date,
        ic.deposit_date,
        ic.check_number,
        ic.check_amount,
        ic.check_applied_amount,
        ic.provider_adjustment_amount,
        ic.check_remaining_amount,
        ic.check_committed,
        EXTRACT(MONTH FROM ic.deposit_date)::int as deposit_month_number
    FROM collaboratemd.insurance_check ic
    CROSS JOIN date_range dr
    WHERE 
        -- Filter by year using deposit_date (matching bank deposit API)
        ic.deposit_date >= dr.year_start 
        AND ic.deposit_date < dr.year_end
        -- Filter by month if specified (using deposit_date month)
        AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
        -- Search filter (optional)
        AND ($3::text IS NULL OR $3::text = '' OR 
            ic.check_number ILIKE '%' || $3::text || '%' OR
            ic.check_payer ILIKE '%' || $3::text || '%')
),
filtered_checks AS (
    -- Apply cursor-based pagination filters
    SELECT 
        icf.id,
        icf.check_payer,
        icf.check_date,
        icf.deposit_date,
        icf.check_number,
        icf.check_amount,
        icf.provider_adjustment_amount,
        icf.check_remaining_amount,
        icf.check_committed,
        icf.deposit_month_number
    FROM insurance_checks_filtered icf
    WHERE 
        -- Cursor-based pagination (if cursor provided)
        -- Cursor format: "deposit_date|check_number|id"
        (
            $5::text IS NULL OR $5::text = '' OR
            TO_CHAR(icf.deposit_date, 'YYYY-MM-DD') < SPLIT_PART($5::text, '|', 1)
            OR (TO_CHAR(icf.deposit_date, 'YYYY-MM-DD') = SPLIT_PART($5::text, '|', 1)
                AND icf.check_number < SPLIT_PART($5::text, '|', 2))
            OR (TO_CHAR(icf.deposit_date, 'YYYY-MM-DD') = SPLIT_PART($5::text, '|', 1)
                AND icf.check_number = SPLIT_PART($5::text, '|', 2)
                AND icf.id < SPLIT_PART($5::text, '|', 3))
        )
),
applied_amounts AS (
    -- Calculate applied amount from payments: sum of insurance_paid_a for payments with each check_number
    -- IMPORTANT: Match by check_number only (not deposit_date) - all payments with same check_number are counted
    SELECT 
        fc.id,
        fc.check_number,
        SUM(COALESCE(p.insurance_paid_a, 0))::numeric as check_applied_amount
    FROM filtered_checks fc
    LEFT JOIN collaboratemd.new_payment p
        ON p.payment_check = fc.check_number
    WHERE p.payment_check IS NOT NULL 
        AND p.payment_check != ''
    GROUP BY fc.id, fc.check_number
)
SELECT 
    fc.id,
    fc.check_payer,
    fc.check_date,
    fc.deposit_date,
    fc.check_number,
    fc.check_amount,
    COALESCE(aa.check_applied_amount, 0)::numeric as check_applied_amount,
    fc.provider_adjustment_amount,
    -- Use database column value directly (not calculated)
    fc.check_remaining_amount,
    fc.check_committed,
    fc.deposit_month_number::int as month_number,
    TO_CHAR(MAKE_DATE($1::int, fc.deposit_month_number, 1), 'Month') as month_name
FROM filtered_checks fc
LEFT JOIN applied_amounts aa ON fc.id = aa.id
ORDER BY 
    fc.deposit_date DESC,
    fc.check_date DESC,
    fc.check_number,
    fc.id
LIMIT $4::int;

-- name: GetInsuranceChecksCount :one
-- Get total count of insurance checks matching filters
-- Uses same logic as GetInsuranceChecksPaginated (filter by deposit_date)
-- 
-- $1::int = year (required)
-- $2::int = month (0 = all months, 1-12 = specific month)
-- $3::text = search (optional search term)
WITH date_range AS (
    -- Date range for filtering by year and month based on deposit_date
    SELECT 
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) as year_start,
        DATE_TRUNC('year', MAKE_DATE($1::int, 1, 1)) + INTERVAL '1 year' as year_end,
        CASE WHEN $2::int = 0 THEN NULL ELSE $2::int END as month_filter
)
SELECT COUNT(*)::bigint as total_count
FROM collaboratemd.insurance_check ic
CROSS JOIN date_range dr
WHERE 
    -- Filter by year using deposit_date (matching bank deposit API)
    ic.deposit_date >= dr.year_start 
    AND ic.deposit_date < dr.year_end
    -- Filter by month if specified (using deposit_date month)
    AND (dr.month_filter IS NULL OR EXTRACT(MONTH FROM ic.deposit_date) = dr.month_filter)
    -- Search filter (optional)
    AND ($3::text IS NULL OR $3::text = '' OR 
        ic.check_number ILIKE '%' || $3::text || '%' OR
        ic.check_payer ILIKE '%' || $3::text || '%');

