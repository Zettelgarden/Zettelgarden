-- Add fact_entity_extraction to the llm_jobs job type check constraint

-- Drop the old constraint
ALTER TABLE llm_jobs DROP CONSTRAINT IF EXISTS llm_jobs_type_check;

-- Add the updated constraint with fact_entity_extraction included
ALTER TABLE llm_jobs ADD CONSTRAINT llm_jobs_type_check
    CHECK (job_type IN ('embedding', 'summarization', 'entity_extraction', 'fact_entity_extraction', 'chat', 'memory', 'email'));
