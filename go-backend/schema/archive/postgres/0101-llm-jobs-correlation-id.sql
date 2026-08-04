-- Add correlation_id column to llm_jobs for distributed tracing
ALTER TABLE llm_jobs ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(36);

-- Create index on correlation_id for efficient lookups during tracing
CREATE INDEX IF NOT EXISTS idx_llm_jobs_correlation_id ON llm_jobs(correlation_id) WHERE correlation_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN llm_jobs.correlation_id IS 'Correlation ID for distributed tracing - links job to request context';
