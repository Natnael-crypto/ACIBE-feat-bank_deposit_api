-- Rollback: Remove performance indexes
-- Migration: 000007_add_performance_indexes

-- Drop indexes for new_claim table
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_claim_from_date;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_claim_from_date_month;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_claim_id;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_unique_claim;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_cpt_code;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_claim_charge;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_revenue_code;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_peer;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_non_peer;
DROP INDEX IF EXISTS collaboratemd.idx_new_claim_date_claim_id;

-- Drop indexes for new_payment table
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_claim_id;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_claim_charge;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_cpt_code;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_dedupe;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_non_zero;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_check;
DROP INDEX IF EXISTS collaboratemd.idx_new_payment_grouping;

