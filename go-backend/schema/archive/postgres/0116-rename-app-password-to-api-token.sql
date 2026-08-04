-- Migration: Rename app_password_encrypted to api_token_encrypted
-- This reflects Fastmail's move from Basic auth (app passwords) to Bearer tokens (API tokens)

-- Rename the column
ALTER TABLE email_accounts RENAME COLUMN app_password_encrypted TO api_token_encrypted;

-- Update the comment
COMMENT ON COLUMN email_accounts.api_token_encrypted IS 'Encrypted Fastmail API token (Bearer token for JMAP access)';
