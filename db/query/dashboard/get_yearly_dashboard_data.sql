-- name: GetYearlyDashboardData :many
-- Get yearly aggregated claim data for all available years
SELECT 
    year,
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
FROM collaboratemd.dashboard_yearly_data
ORDER BY year DESC, claim_category;
