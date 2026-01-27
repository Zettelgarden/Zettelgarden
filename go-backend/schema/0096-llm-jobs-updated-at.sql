-- Add updated_at column to llm_jobs table for tracking modification times
ALTER TABLE llm_jobs ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT NOW();

-- Create index on updated_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_llm_jobs_updated_at ON llm_jobs(updated_at DESC);

-- Update updated_at on status changes using a trigger
CREATE OR REPLACE FUNCTION update_llm_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_llm_jobs_updated_at
	BEFORE UPDATE ON llm_jobs
	FOR EACH ROW
	EXECUTE FUNCTION update_llm_jobs_updated_at();

-- Add comment for documentation
COMMENT ON COLUMN llm_jobs.updated_at IS 'Last time the job was updated (status change, retry, etc.)';
