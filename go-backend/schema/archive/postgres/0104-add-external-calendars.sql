-- Migration: Add external_calendars table
-- Description: Store subscriptions to external iCal feeds (Google Calendar, etc.)
-- Created: 2026-01-31

CREATE TABLE external_calendars (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Subscription details
    name TEXT NOT NULL,
    url TEXT NOT NULL,

    -- Sync settings
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_interval_hours INTEGER DEFAULT 1,

    -- Display
    color TEXT DEFAULT '#6366f1',

    -- Metadata
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    UNIQUE(user_id, url)
);

CREATE INDEX idx_external_calendars_user ON external_calendars(user_id);
COMMENT ON TABLE external_calendars IS 'External calendar subscriptions for importing iCal events';
COMMENT ON COLUMN external_calendars.color IS 'Display color for events from this calendar (hex, default indigo)';
COMMENT ON COLUMN external_calendars.sync_interval_hours IS 'Hours between automatic syncs (1 = hourly, for future background job)';
