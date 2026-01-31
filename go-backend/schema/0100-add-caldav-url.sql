-- Migration: Add CalDAV URL field to users table
-- Description: Add support for CalDAV calendar sync by storing CalDAV server URL
-- Created: 2026-01-31

-- Add caldav_url column to users table
ALTER TABLE users ADD COLUMN caldav_url TEXT;

-- Add comment for documentation
COMMENT ON COLUMN users.caldav_url IS 'CalDAV server URL for calendar sync (e.g., https://calendar.google.com/dav/user@example.com/user)';
