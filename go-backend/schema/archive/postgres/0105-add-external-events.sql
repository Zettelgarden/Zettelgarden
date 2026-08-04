-- Migration: Add external_events table
-- Description: Store imported events from external iCal feeds
-- Created: 2026-01-31

CREATE TABLE external_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_calendar_id INTEGER REFERENCES external_calendars(id) ON DELETE SET NULL,

    -- Event details
    title TEXT NOT NULL,
    description TEXT,

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,

    -- Location
    location TEXT,

    -- External sync tracking
    external_uid TEXT,
    external_url TEXT,

    -- Recurrence (stored for reference, expansion not in initial scope)
    recurrence_rule TEXT,

    -- Display
    color TEXT,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_synced_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(user_id, external_uid)
);

CREATE INDEX idx_external_events_user_time ON external_events(user_id, start_time, end_time);
CREATE INDEX idx_external_events_calendar ON external_events(external_calendar_id);
COMMENT ON TABLE external_events IS 'Imported calendar events from external iCal feeds';
COMMENT ON COLUMN external_events.external_uid IS 'UID from iCal feed for deduplication';
COMMENT ON COLUMN external_events.color IS 'Override color for this specific event (falls back to calendar color if null)';
