-- Add last_memory_job_id to users table for tracking memory generation jobs
-- This replaces the per-user mutex mechanism with proper job queue tracking

ALTER TABLE users ADD COLUMN IF NOT EXISTS last_memory_job_id INTEGER REFERENCES llm_jobs(id);

-- Create index for efficient lookups of user's last memory job
CREATE INDEX IF NOT EXISTS idx_users_last_memory_job_id ON users(last_memory_job_id);
