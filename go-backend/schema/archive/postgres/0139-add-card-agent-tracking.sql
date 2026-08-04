-- Migration: Add agent tracking to cards table
-- Description: Track which agent created a card for multi-user agent support
-- Created: 2026-04-06

-- Track which agent created a card
ALTER TABLE cards ADD COLUMN IF NOT EXISTS created_by_agent_id INT NULL REFERENCES users(id) ON DELETE SET NULL;

-- Index for faster filtering
CREATE INDEX IF NOT EXISTS idx_cards_created_by_agent ON cards(created_by_agent_id) WHERE created_by_agent_id IS NOT NULL;

-- Add column comment for documentation
COMMENT ON COLUMN cards.created_by_agent_id IS 'For cards created by AI agents: the agent user who created this card';
