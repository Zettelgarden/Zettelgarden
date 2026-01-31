-- Migration: Add CalDAV token field to users table
-- Description: Add support for secure calendar feed access via token
-- Created: 2026-01-31

-- Add caldav_token column to users table
-- This token allows external calendar apps to access the user's task calendar
ALTER TABLE users ADD COLUMN caldav_token TEXT UNIQUE;

-- Add comment for documentation
COMMENT ON COLUMN users.caldav_token IS 'Secure token for accessing iCal feed at /api/user/calendar.ics?token=XYZ';

-- Create index on caldav_token for faster lookups
CREATE INDEX idx_users_caldav_token ON users(caldav_token) WHERE caldav_token IS NOT NULL;
