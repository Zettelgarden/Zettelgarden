-- Rollback migration for task sort_order

DROP INDEX IF EXISTS idx_tasks_sort_order;
ALTER TABLE tasks DROP COLUMN IF EXISTS sort_order;
