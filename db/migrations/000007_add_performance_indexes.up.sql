-- Migration: Add performance indexes for dashboard queries
-- Purpose: Optimize claims and payments API queries to reduce response time from ~1.5s to <200ms
-- Created: 2025-12-07
--
-- NOTE: Existing indexes from schema:
-- - new_claim: PRIMARY KEY (claim_id, charge_id) - creates index automatically
-- - new_payment: UNIQUE constraint creates index on (payment_entered_date, claim_id, credit_id, charge_id, cpt_code, payment_check, is_refund, is_provider_adjustment, is_era_adjustment)
--
-- These existing indexes are NOT optimized for our query patterns:
-- - Date filtering (claim_from_date)
-- - Month/year filtering
-- - CPT code filtering alone
-- - claim_id-only joins (PK requires both claim_id + charge_id)
--
-- This migration adds indexes specifically optimized for dashboard query patterns.

-- ============================================
-- Indexes for new_claim table
-- ============================================

-- Index for date filtering (most common filter)
-- Used in: WHERE claim_from_date >= year_start AND claim_from_date < year_end
CREATE INDEX IF NOT EXISTS idx_new_claim_claim_from_date 
ON collaboratemd.new_claim(claim_from_date);

-- Composite index for date + month filtering (very common pattern)
-- Used in: WHERE claim_from_date >= year_start AND EXTRACT(MONTH FROM claim_from_date) = month
CREATE INDEX IF NOT EXISTS idx_new_claim_claim_from_date_month 
ON collaboratemd.new_claim(claim_from_date, EXTRACT(MONTH FROM claim_from_date));

-- Index for claim_id lookups (used in joins and searches)
-- Note: PRIMARY KEY includes claim_id but requires charge_id, so this single-column index helps with claim_id-only lookups
CREATE INDEX IF NOT EXISTS idx_new_claim_claim_id 
ON collaboratemd.new_claim(claim_id);

-- Composite index for unique claim identification (claim_id + charge_id + cpt_code)
-- Used in DISTINCT ON operations
CREATE INDEX IF NOT EXISTS idx_new_claim_unique_claim 
ON collaboratemd.new_claim(claim_id, charge_id, cpt_code);

-- Index for CPT code filtering (peer filtering, type determination)
CREATE INDEX IF NOT EXISTS idx_new_claim_cpt_code 
ON collaboratemd.new_claim(cpt_code);

-- Composite index for claim_id + charge_id (common join pattern)
CREATE INDEX IF NOT EXISTS idx_new_claim_claim_charge 
ON collaboratemd.new_claim(claim_id, charge_id);

-- Index for revenue_code (used in type determination)
CREATE INDEX IF NOT EXISTS idx_new_claim_revenue_code 
ON collaboratemd.new_claim(revenue_code);

-- Partial index for peer claims (H0038) - smaller, faster for peer-only queries
CREATE INDEX IF NOT EXISTS idx_new_claim_peer 
ON collaboratemd.new_claim(claim_id, charge_id, claim_from_date) 
WHERE cpt_code = 'H0038';

-- Partial index for non-peer claims - optimizes exclude_peer=true queries
CREATE INDEX IF NOT EXISTS idx_new_claim_non_peer 
ON collaboratemd.new_claim(claim_id, charge_id, claim_from_date) 
WHERE cpt_code != 'H0038';

-- Composite index for date range + claim_id (optimizes filtered joins)
CREATE INDEX IF NOT EXISTS idx_new_claim_date_claim_id 
ON collaboratemd.new_claim(claim_from_date, claim_id);

-- ============================================
-- Indexes for new_payment table
-- ============================================

-- Index for claim_id (most common join key)
CREATE INDEX IF NOT EXISTS idx_new_payment_claim_id 
ON collaboratemd.new_payment(claim_id);

-- Composite index for claim_id + charge_id (common join pattern)
CREATE INDEX IF NOT EXISTS idx_new_payment_claim_charge 
ON collaboratemd.new_payment(claim_id, charge_id);

-- Index for CPT code filtering
CREATE INDEX IF NOT EXISTS idx_new_payment_cpt_code 
ON collaboratemd.new_payment(cpt_code);

-- Composite index for payment deduplication (DISTINCT ON pattern)
-- Note: UNIQUE constraint already exists on these columns, but this index order may be optimized differently
-- The UNIQUE constraint index may not be optimal for DISTINCT ON queries with different ordering
CREATE INDEX IF NOT EXISTS idx_new_payment_dedupe 
ON collaboratemd.new_payment(payment_entered_date, claim_id, credit_id, charge_id, cpt_code, payment_check, is_refund, is_provider_adjustment, is_era_adjustment);

-- Index for filtering non-zero payments
CREATE INDEX IF NOT EXISTS idx_new_payment_non_zero 
ON collaboratemd.new_payment(claim_id, charge_id) 
WHERE insurance_paid_a != 0;

-- Index for payment_check (TRN) lookups
CREATE INDEX IF NOT EXISTS idx_new_payment_check 
ON collaboratemd.new_payment(payment_check);

-- Composite index for claim_id + charge_id + cpt_code (payment grouping)
CREATE INDEX IF NOT EXISTS idx_new_payment_grouping 
ON collaboratemd.new_payment(claim_id, charge_id, cpt_code);

-- ============================================
-- Text search indexes (for ILIKE searches)
-- ============================================

-- GIN index for claim_id text search (if using pg_trgm extension)
-- Note: Requires CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- Uncomment if pg_trgm is available:
-- CREATE INDEX IF NOT EXISTS idx_new_claim_claim_id_trgm 
-- ON collaboratemd.new_claim USING gin(claim_id gin_trgm_ops);

-- ============================================
-- Analyze tables to update statistics
-- ============================================

ANALYZE collaboratemd.new_claim;
ANALYZE collaboratemd.new_payment;

