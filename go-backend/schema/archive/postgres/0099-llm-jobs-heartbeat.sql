-- Add last_heartbeat column to llm_jobs for monitoring stuck jobs
ALTER TABLE llm_jobs
ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMP;

-- Create index for efficient stuck job detection
-- This partial index covers running jobs with their heartbeat info
CREATE INDEX IF NOT EXISTS idx_llm_jobs_running_heartbeat
ON llm_jobs(id, started_at, last_heartbeat)
WHERE status = 'running';

-- Add comment for documentation
COMMENT ON COLUMN llm_jobs.last_heartbeat IS 'Last heartbeat timestamp from running job, used to detect stuck/abandoned jobs';
