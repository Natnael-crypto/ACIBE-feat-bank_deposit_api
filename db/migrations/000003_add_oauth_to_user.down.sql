-- Rollback migration: Remove OAuth Provider ID columns from account.user table

-- Drop indexes
DROP INDEX IF EXISTS account.idx_user_google_id;
DROP INDEX IF EXISTS account.idx_user_microsoft_id;

-- Drop columns
ALTER TABLE account.user 
DROP COLUMN IF EXISTS google_id,
DROP COLUMN IF EXISTS microsoft_id;

