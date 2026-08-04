-- Migration: Add parent_task_id column to tasks table
-- Description: Enables hierarchical task relationships for subtasks
-- Created: 2026-03-10

-- Add parent_task_id column for self-referential task hierarchy
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS parent_task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE;

-- Index for efficient child lookups
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_task_id);

-- Add column comment for documentation
COMMENT ON COLUMN tasks.parent_task_id IS 'References parent task for subtask hierarchy. NULL for root tasks. Single level nesting only.';

-- ============================================================================
-- DOWN MIGRATION (to reverse these changes)
-- ============================================================================
-- DROP INDEX IF EXISTS idx_tasks_parent_id;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS parent_task_id;
