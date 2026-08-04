-- Migration: Update sync_status check constraint to include 'syncing' state
-- The code uses 'syncing' to indicate a sync is in progress

-- Drop the old constraint
ALTER TABLE email_accounts DROP CONSTRAINT IF EXISTS email_accounts_sync_status_check;

-- Add the updated constraint with 'syncing' included
ALTER TABLE email_accounts ADD CONSTRAINT email_accounts_sync_status_check
    CHECK (sync_status IN ('active', 'syncing', 'error', 'disabled'));

-- Update comment
COMMENT ON COLUMN email_accounts.sync_status IS 'Current sync status: active, syncing, error, or disabled';
