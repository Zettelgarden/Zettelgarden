-- Optimize stats queries for activity tracking
-- These indexes improve performance for date-range queries on cards and tasks

-- Index for cards created stats query
CREATE INDEX IF NOT EXISTS idx_cards_user_created
ON cards(user_id, created_at)
WHERE is_deleted = FALSE;

-- Index for tasks created stats query
CREATE INDEX IF NOT EXISTS idx_tasks_user_created
ON tasks(user_id, created_at)
WHERE is_deleted = FALSE;

-- Index for tasks completed stats query
CREATE INDEX IF NOT EXISTS idx_tasks_user_completed
ON tasks(user_id, completed_at)
WHERE is_deleted = FALSE AND completed_at IS NOT NULL;
