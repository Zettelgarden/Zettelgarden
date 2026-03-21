-- Fix foreign key constraint on users.last_memory_job_id to allow job deletion
-- The constraint was missing ON DELETE SET NULL, causing cleanup jobs to fail
-- when trying to delete old llm_jobs that are still referenced by users

-- Drop the existing constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_last_memory_job_id_fkey;

-- Recreate with ON DELETE SET NULL
ALTER TABLE users ADD CONSTRAINT users_last_memory_job_id_fkey
  FOREIGN KEY (last_memory_job_id) REFERENCES llm_jobs(id) ON DELETE SET NULL;
