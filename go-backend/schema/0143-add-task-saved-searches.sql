-- Migration: Add task_saved_searches table
-- Description: Stores user-saved task searches (filterString + sort + viewMode)
--              so frequent searches can be recalled and synced across devices.
-- Created: 2026-06-18

CREATE TABLE IF NOT EXISTS task_saved_searches (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  filter_string TEXT NOT NULL DEFAULT '',
  sort_field TEXT NOT NULL DEFAULT 'priority',
  sort_direction TEXT NOT NULL DEFAULT 'asc',
  view_mode TEXT NOT NULL DEFAULT 'list',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS task_saved_searches_user_id_idx ON task_saved_searches(user_id);

COMMENT ON TABLE task_saved_searches IS 'User-saved task searches synced across devices';
