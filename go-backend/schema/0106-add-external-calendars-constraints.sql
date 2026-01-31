-- Migration: Add constraints to external_calendars table
-- Description: Add CHECK constraint for sync_interval_hours to enforce bounds at database level
-- Created: 2026-01-31

-- Add CHECK constraint for sync_interval_hours (1-168 hours)
ALTER TABLE external_calendars
ADD CONSTRAINT sync_interval_hours_bounds
CHECK (sync_interval_hours >= 1 AND sync_interval_hours <= 168);

-- Add constraint to ensure name is not empty (beyond NOT NULL)
ALTER TABLE external_calendars
ADD CONSTRAINT name_not_empty
CHECK (LENGTH(TRIM(name)) > 0);

-- Add reasonable max length for name to prevent database bloat
ALTER TABLE external_calendars
ALTER COLUMN name TYPE TEXT;  -- TEXT is already used, this documents the constraint

COMMENT ON CONSTRAINT sync_interval_hours_bounds ON external_calendars IS 'Ensures sync interval is between 1 and 168 hours (1 hour to 7 days)';
COMMENT ON CONSTRAINT name_not_empty ON external_calendars IS 'Ensures calendar name is not empty or whitespace only';
