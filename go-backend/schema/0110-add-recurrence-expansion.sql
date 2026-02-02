-- Migration: Add recurrence expansion support for external events
-- Description: Add fields to track recurring event instances
-- Created: 2026-02-02

-- Add recurrence tracking fields to external_events
ALTER TABLE external_events
ADD COLUMN recurrence_id TEXT,
ADD COLUMN recurrence_instance INTEGER;

-- Create index for finding all instances of a recurring series
CREATE INDEX idx_external_events_recurrence_id ON external_events(recurrence_id);

-- Drop the old unique constraint on (user_id, external_uid)
-- We need to support multiple instances with different external_uid values
ALTER TABLE external_events DROP CONSTRAINT external_events_user_id_external_uid_key;

-- Add a new unique constraint that allows multiple instances
-- but prevents duplicates of the same instance
ALTER TABLE external_events
ADD CONSTRAINT external_events_user_id_external_uid_key
UNIQUE(user_id, external_uid);

-- Add index for querying by recurrence series
CREATE INDEX idx_external_events_recurrence_series ON external_events(user_id, recurrence_id);

COMMENT ON COLUMN external_events.recurrence_id IS 'Identifier for the recurring event series (same as external_uid for non-recurring events)';
COMMENT ON COLUMN external_events.recurrence_instance IS 'Instance index for recurring events (0-based, NULL for non-recurring events)';
