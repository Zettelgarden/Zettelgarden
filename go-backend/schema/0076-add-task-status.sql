-- Add status column to tasks table
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'todo';

-- Create index for filtering by status
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Update existing completed tasks to have 'done' status
UPDATE tasks SET status = 'done' WHERE is_complete = true;

-- Update existing incomplete tasks to have 'todo' status (already default, but explicit)
UPDATE tasks SET status = 'todo' WHERE is_complete = false AND status IS NULL;
