-- Migration: Add authentication fields to external_calendars table
-- Description: Support Basic Auth for password-protected iCal/CalDAV feeds
-- Created: 2025-02-01

-- Add username field (plaintext, safe to expose)
ALTER TABLE external_calendars ADD COLUMN username TEXT;

-- Add password field (encrypted at rest using AES-256-GCM)
ALTER TABLE external_calendars ADD COLUMN password TEXT;

-- Add documentation
COMMENT ON COLUMN external_calendars.username IS 'Username for Basic Auth (if calendar requires authentication)';
COMMENT ON COLUMN external_calendars.password IS 'Encrypted password for Basic Auth (AES-256-GCM encrypted)';
