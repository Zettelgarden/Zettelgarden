-- Immediate fix for existing databases experiencing the foreign key constraint error
-- Run this manually if you're currently seeing the cleanup error

-- Step 1: Clear any references to jobs that would be deleted by cleanup
-- (jobs older than retention period that are completed/failed/cancelled)
UPDATE users
SET last_memory_job_id = NULL
WHERE last_memory_job_id IN (
    SELECT id FROM llm_jobs
    WHERE status IN ('completed', 'failed', 'cancelled')
      AND completed_at < NOW() - INTERVAL '30 days'
);

-- Step 2: Drop and recreate the constraint with ON DELETE SET NULL
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_last_memory_job_id_fkey;
ALTER TABLE users ADD CONSTRAINT users_last_memory_job_id_fkey
  FOREIGN KEY (last_memory_job_id) REFERENCES llm_jobs(id) ON DELETE SET NULL;

-- Step 3: Now the cleanup job should work
-- You can run it manually to verify:
-- SELECT cleanup_old_jobs(30);
