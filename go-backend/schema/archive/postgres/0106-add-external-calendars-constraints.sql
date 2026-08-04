-- Migration: Add constraints to external_calendars table
-- Description: Add CHECK constraint for sync_interval_hours to enforce bounds at database level
-- Created: 2026-01-31

-- Add CHECK constraint for sync_interval_hours (1-168 hours)
-- Use IF NOT EXISTS pattern by checking if constraint exists first
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sync_interval_hours_bounds'
    ) THEN
        ALTER TABLE external_calendars
        ADD CONSTRAINT sync_interval_hours_bounds
        CHECK (sync_interval_hours >= 1 AND sync_interval_hours <= 168);
    END IF;
END $$;

-- Add constraint to ensure name is not empty (beyond NOT NULL)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'name_not_empty'
    ) THEN
        ALTER TABLE external_calendars
        ADD CONSTRAINT name_not_empty
        CHECK (LENGTH(TRIM(name)) > 0);
    END IF;
END $$;

COMMENT ON CONSTRAINT sync_interval_hours_bounds ON external_calendars IS 'Ensures sync interval is between 1 and 168 hours (1 hour to 7 days)';
COMMENT ON CONSTRAINT name_not_empty ON external_calendars IS 'Ensures calendar name is not empty or whitespace only';
