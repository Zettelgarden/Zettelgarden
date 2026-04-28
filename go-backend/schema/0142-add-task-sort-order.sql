-- Migration: Add sort_order column to tasks for manual ordering
-- Description: Allows users to manually reorder tasks via drag-and-drop in list view
-- Created: 2026-04-28

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT NULL;

-- Index for efficient sorting by sort_order
CREATE INDEX IF NOT EXISTS idx_tasks_sort_order ON tasks(user_id, sort_order) WHERE sort_order IS NOT NULL;
