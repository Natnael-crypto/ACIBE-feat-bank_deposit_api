-- Migration: Add OAuth Provider ID columns to account.user table
-- Date: 2025-01-XX
-- Description: Adds google_id and microsoft_id columns to support OAuth authentication

-- Add columns
ALTER TABLE account.user 
ADD COLUMN IF NOT EXISTS google_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS microsoft_id VARCHAR(255);

-- Create unique indexes (only for non-null values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_google_id ON account.user(google_id) WHERE google_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_microsoft_id ON account.user(microsoft_id) WHERE microsoft_id IS NOT NULL;

-- Add comments
COMMENT ON COLUMN account.user.google_id IS 'Google OAuth provider user ID';
COMMENT ON COLUMN account.user.microsoft_id IS 'Microsoft OAuth provider user ID';

