-- Add llm_job_id column to summarizations table to track job association
ALTER TABLE summarizations ADD COLUMN llm_job_id INT NULL REFERENCES llm_jobs(id) ON DELETE CASCADE;

-- Create index for efficient lookups
CREATE INDEX idx_summarizations_llm_job_id ON summarizations(llm_job_id);

-- Add comment for documentation
COMMENT ON COLUMN summarizations.llm_job_id IS 'Reference to the LLM job processing this summarization';
