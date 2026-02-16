-- Migration: Add notifications table
-- Description: Unified view of email, RSS, and task notifications
-- Created: 2026-02-16

CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type VARCHAR(20) NOT NULL CHECK (source_type IN ('email', 'rss', 'task')),
    source_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    preview TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    importance_score INTEGER DEFAULT 0 CHECK (importance_score >= 0),
    is_read BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    filter_tags TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_type, source_id)
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_timestamp
    ON notifications(user_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications(user_id, is_read, is_archived)
    WHERE is_read = FALSE AND is_archived = FALSE;

CREATE INDEX IF NOT EXISTS idx_notifications_filter_tags
    ON notifications USING GIN(filter_tags);

COMMENT ON TABLE notifications IS 'Unified view of important items from email, RSS, and tasks';
COMMENT ON COLUMN notifications.source_type IS 'Type of source: email, rss, or task';
COMMENT ON COLUMN notifications.importance_score IS 'Computed score for sorting';
COMMENT ON COLUMN notifications.filter_tags IS 'Tags for filtering';
