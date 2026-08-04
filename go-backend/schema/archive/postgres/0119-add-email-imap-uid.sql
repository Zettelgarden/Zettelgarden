-- Migration: Add IMAP UID to emails table
-- Description: Adds IMAP UID column to emails for moving messages to/from Archive folder

-- Add imap_uid column to store the IMAP message UID for each email
-- This is needed to move emails between folders via IMAP
ALTER TABLE emails ADD COLUMN IF NOT EXISTS imap_uid BIGINT;

-- Create index on imap_uid for faster lookups during IMAP operations
CREATE INDEX IF NOT EXISTS idx_emails_imap_uid ON emails(imap_uid) WHERE imap_uid IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN emails.imap_uid IS 'IMAP message UID for this email, used for folder operations like archive';
