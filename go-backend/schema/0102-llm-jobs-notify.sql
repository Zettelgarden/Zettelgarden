-- Create a function to notify when a new job is inserted
CREATE OR REPLACE FUNCTION notify_llm_job_insert() RETURNS trigger AS $$
BEGIN
	PERFORM pg_notify('llm_job_queue', 'new_job');
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger that fires after a new job is inserted
DROP TRIGGER IF EXISTS llm_job_insert_trigger ON llm_jobs;
CREATE TRIGGER llm_job_insert_trigger
AFTER INSERT ON llm_jobs
FOR EACH STATEMENT
WHEN (pg_trigger_depth() = 0)  -- Only fire for top-level inserts
EXECUTE FUNCTION notify_llm_job_insert();

-- Add comment for documentation
COMMENT ON FUNCTION notify_llm_job_insert() IS 'Sends PostgreSQL NOTIFY event when a new job is added to the queue';
