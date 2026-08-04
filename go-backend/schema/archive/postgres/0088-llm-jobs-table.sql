-- Create llm_jobs table for asynchronous LLM operation queue
CREATE TABLE IF NOT EXISTS llm_jobs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    payload JSONB NOT NULL,
    result JSONB,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    timeout_seconds INT NOT NULL DEFAULT 300,

    -- Constraint: status must be one of the valid states
    CONSTRAINT llm_jobs_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),

    -- Constraint: job_type must be one of the valid types
    CONSTRAINT llm_jobs_type_check CHECK (job_type IN ('embedding', 'summarization', 'entity_extraction', 'chat', 'memory', 'email'))
);

-- Create index on (user_id, status) for querying user's active jobs
CREATE INDEX IF NOT EXISTS idx_llm_jobs_user_status ON llm_jobs(user_id, status);

-- Create index on created_at for time-based queries and pagination
CREATE INDEX IF NOT EXISTS idx_llm_jobs_created_at ON llm_jobs(created_at DESC);

-- Create index on priority for priority queue processing
CREATE INDEX IF NOT EXISTS idx_llm_jobs_priority ON llm_jobs(priority) WHERE status = 'pending';

-- Create index on status for efficient worker polling
CREATE INDEX IF NOT EXISTS idx_llm_jobs_status ON llm_jobs(status) WHERE status IN ('pending', 'running');

-- Add comment for documentation
COMMENT ON TABLE llm_jobs IS 'Queue for asynchronous LLM operations with status tracking and retry logic';
COMMENT ON COLUMN llm_jobs.job_type IS 'Type of LLM job: embedding, summarization, entity_extraction, chat, memory, email';
COMMENT ON COLUMN llm_jobs.status IS 'Job status: pending, running, completed, failed, cancelled';
COMMENT ON COLUMN llm_jobs.priority IS 'Job priority (lower number = higher priority, default 5)';
COMMENT ON COLUMN llm_jobs.payload IS 'Job-specific input data as JSONB';
COMMENT ON COLUMN llm_jobs.result IS 'Job result data as JSONB';
COMMENT ON COLUMN llm_jobs.retry_count IS 'Number of retry attempts made';
COMMENT ON COLUMN llm_jobs.max_retries IS 'Maximum retry attempts before marking as failed';
COMMENT ON COLUMN llm_jobs.timeout_seconds IS 'Maximum time allowed for job execution';
