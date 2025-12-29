-- Check if critical dashboard indexes exist
-- Run this to verify your local database has the required indexes

-- Check for the most critical index for dashboard queries
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_indexes 
            WHERE schemaname = 'collaboratemd' 
            AND tablename = 'new_claim' 
            AND indexname = 'idx_new_claim_from_date_claim_id'
        ) THEN '✓ EXISTS' 
        ELSE '✗ MISSING - CRITICAL!' 
    END as idx_new_claim_from_date_claim_id,
    'Most important index for dashboard query performance' as description;

-- Check other critical indexes
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_indexes 
            WHERE schemaname = 'collaboratemd' 
            AND tablename = 'new_claim' 
            AND indexname = 'idx_new_claim_claim_from_date'
        ) THEN '✓ EXISTS' 
        ELSE '✗ MISSING' 
    END as idx_new_claim_claim_from_date;

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_indexes 
            WHERE schemaname = 'collaboratemd' 
            AND tablename = 'new_claim' 
            AND indexname = 'idx_new_claim_claim_id'
        ) THEN '✓ EXISTS' 
        ELSE '✗ MISSING' 
    END as idx_new_claim_claim_id;

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_indexes 
            WHERE schemaname = 'collaboratemd' 
            AND tablename = 'new_payment' 
            AND indexname = 'idx_new_payment_join_match'
        ) THEN '✓ EXISTS' 
        ELSE '✗ MISSING' 
    END as idx_new_payment_join_match;

-- List all indexes on new_claim table
SELECT 
    indexname,
    indexdef
FROM pg_indexes 
WHERE schemaname = 'collaboratemd' 
AND tablename = 'new_claim'
ORDER BY indexname;

