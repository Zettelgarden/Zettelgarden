-- Migration: Add notification_preferences table
-- Description: User preferences for notification filtering
-- Created: 2026-02-16

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Filter toggles - what to include in unified inbox
    show_unprocessed_emails BOOLEAN DEFAULT TRUE,
    show_starred_articles BOOLEAN DEFAULT TRUE,
    show_priority_tasks BOOLEAN DEFAULT TRUE,
    show_priority_feeds BOOLEAN DEFAULT TRUE,
    -- Display settings
    items_per_page INTEGER DEFAULT 50 CHECK (items_per_page >= 10 AND items_per_page <= 200),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create default preferences for existing users
INSERT INTO notification_preferences (user_id)
SELECT id FROM users
ON CONFLICT (user_id) DO NOTHING;

-- Add table and column comments for documentation
COMMENT ON TABLE notification_preferences IS 'User preferences for notification filtering and display';
COMMENT ON COLUMN notification_preferences.show_unprocessed_emails IS 'Include unprocessed emails in unified inbox';
COMMENT ON COLUMN notification_preferences.show_starred_articles IS 'Include starred RSS articles in unified inbox';
COMMENT ON COLUMN notification_preferences.show_priority_tasks IS 'Include priority tasks in unified inbox';
COMMENT ON COLUMN notification_preferences.show_priority_feeds IS 'Include priority RSS feeds in unified inbox';
COMMENT ON COLUMN notification_preferences.items_per_page IS 'Number of items to display per page (10-200)';
