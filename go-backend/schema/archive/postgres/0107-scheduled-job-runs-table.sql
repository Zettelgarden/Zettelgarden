-- Migration: Add scheduled_job_runs table
-- Description: Track execution history for scheduled jobs
-- Created: 2026-01-31

CREATE TABLE IF NOT EXISTS scheduled_job_runs (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(255) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraint: status must be one of the valid states
    CONSTRAINT scheduled_job_runs_status_check CHECK (status IN ('running', 'completed', 'failed', 'cancelled'))
);

-- Index for job name lookups
CREATE INDEX IF NOT EXISTS idx_scheduled_job_runs_job_name ON scheduled_job_runs(job_name);

-- Index for status queries
CREATE INDEX IF NOT EXISTS idx_scheduled_job_runs_status ON scheduled_job_runs(status);

-- Index for started_at (for history queries)
CREATE INDEX IF NOT EXISTS idx_scheduled_job_runs_started_at ON scheduled_job_runs(started_at DESC);

-- Index for cleanup of old records
CREATE INDEX IF NOT EXISTS idx_scheduled_job_runs_created_at ON scheduled_job_runs(created_at);

-- Add documentation
COMMENT ON TABLE scheduled_job_runs IS 'Execution history for scheduled jobs - tracks start/end times, status, and errors';
COMMENT ON COLUMN scheduled_job_runs.job_name IS 'Identifier for the type of job (e.g., "daily-backup", "hourly-sync")';
COMMENT ON COLUMN scheduled_job_runs.status IS 'Current state: running, completed, failed, or cancelled';
COMMENT ON COLUMN scheduled_job_runs.retry_count IS 'Number of retry attempts for this job execution';
