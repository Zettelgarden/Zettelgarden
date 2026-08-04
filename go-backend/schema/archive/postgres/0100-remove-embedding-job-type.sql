-- Remove 'embedding' from llm_jobs job type check constraint
-- Embedding jobs are no longer used as Typesense handles embeddings automatically

-- Drop the old constraint
ALTER TABLE llm_jobs DROP CONSTRAINT IF EXISTS llm_jobs_type_check;

-- Add the updated constraint without embedding
ALTER TABLE llm_jobs ADD CONSTRAINT llm_jobs_type_check
    CHECK (job_type IN ('summarization', 'entity_extraction', 'fact_entity_extraction', 'chat', 'memory', 'email'));
