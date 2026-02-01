-- Migration: Add card_pk to external_events table
-- Description: Allow linking external calendar events to cards (same pattern as tasks)
-- Created: 2026-02-01

-- Add card_pk column to external_events
ALTER TABLE external_events
ADD COLUMN card_pk INTEGER REFERENCES cards(id) ON DELETE SET NULL;

-- Create index for efficient queries by card
CREATE INDEX idx_external_events_card_pk ON external_events(card_pk);

-- Add comment
COMMENT ON COLUMN external_events.card_pk IS 'Optional link to a card. Allows calendar events to be associated with specific cards for context and reference.';
