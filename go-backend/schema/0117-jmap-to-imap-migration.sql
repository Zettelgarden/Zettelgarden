-- Migration: JMAP to IMAP for Email Sync
-- This migration adds IMAP-specific columns and renames api_token back to app_password

-- Add IMAP server configuration and state tracking to email_accounts
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS imap_server TEXT DEFAULT 'imap.fastmail.com:993';
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS imap_server_type TEXT DEFAULT 'imap';
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS imap_uid BIGINT;
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS imap_uid_validity BIGINT;

-- Rename api_token_encrypted back to app_password_encrypted for clarity
-- We need to do this by adding the new column first, then copying data, then dropping old
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS app_password_encrypted TEXT;
UPDATE email_accounts SET app_password_encrypted = api_token_encrypted WHERE app_password_encrypted IS NULL;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS api_token_encrypted;

-- Add comments for documentation
COMMENT ON COLUMN email_accounts.imap_server IS 'IMAP server address (e.g., imap.fastmail.com:993)';
COMMENT ON COLUMN email_accounts.imap_server_type IS 'Type of email server (default: imap)';
COMMENT ON COLUMN email_accounts.imap_uid IS 'Last IMAP message UID synced for this account (for incremental sync)';
COMMENT ON COLUMN email_accounts.imap_uid_validity IS 'IMAP UIDVALIDITY value to detect mailbox resets';
COMMENT ON COLUMN email_accounts.app_password_encrypted IS 'Encrypted app password for authentication';
