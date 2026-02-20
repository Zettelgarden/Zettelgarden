-- Migration: Add card_id to emails for tracking converted cards
-- Description: Stores the ID of the card created from an email
-- Created: 2026-02-20

-- Add card_id column to emails
ALTER TABLE emails
ADD COLUMN card_id INT REFERENCES cards(id) ON DELETE SET NULL;

-- Create index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_emails_card_id ON emails(card_id);

-- Add comment for documentation
COMMENT ON COLUMN emails.card_id IS 'ID of the card created from this email, if converted';
