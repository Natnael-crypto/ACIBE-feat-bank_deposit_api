-- name: GetDashboardData :many
-- Get 12-month aggregated claim data for the selected year from materialized view
-- This query is optimized for sub-100ms response times by using pre-computed data
-- 
-- The materialized view dashboard_annual_data contains pre-aggregated data for ALL years
-- This eliminates the need for complex CTEs and multiple table scans
--
-- Key Rules (preserved from original query):
-- 1. Monthly grouping based on new_claim.claim_from_date
-- 2. Unique claim = claim_id (even if multiple rows with different charge_id or CPT)
-- 3. Units = SUM(units) for all charge_id rows for that claim_id
-- 4. Total Claim Amount = SUM(amount) across all charge lines
-- 5. Total Expected Amount = SUM(expected_amount) from new_claim grouped by (claim_id + charge_id)
-- 6. Categorization: IN > PEER > OUT (priority-based)
-- 7. Payments matched by (claim_id + charge_id + cpt_code + modifiers)
-- 8. Each claim counted only once in primary category
--
-- $1::int = year
SELECT 
    month_number,
    month_name,
    claim_category,
    unique_claims,
    total_units,
    total_amount,
    total_expected_amount,
    unique_claims_with_payments,
    total_units_with_payments,
    total_payments_amount,
    total_expected_amount_with_payments,
    unique_claims_with_outstanding,
    total_units_with_outstanding,
    total_outstanding_amount
FROM collaboratemd.dashboard_annual_data
WHERE year = $1::int
ORDER BY month_number, claim_category;
