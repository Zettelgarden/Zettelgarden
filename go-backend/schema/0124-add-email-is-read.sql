-- Migration: Add is_read to emails table
-- Description: Tracks whether an email has been read (synced with IMAP \Seen flag)
-- Created: 2026-02-16

-- Add is_read column to emails table
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE;

-- Add comment
COMMENT ON COLUMN emails.is_read IS 'Whether the email has been read (synced with IMAP \Seen flag)';

-- Create index for efficient querying of unread emails
CREATE INDEX IF NOT EXISTS idx_emails_user_is_read_status ON emails(user_id, is_read, status) WHERE is_read = FALSE AND status = 'unprocessed';
