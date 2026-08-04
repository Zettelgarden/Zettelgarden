-- Migration: Drop trigger logic ported to Go (SQLite migration Phase 5)
-- Date: 2026-07-25
--
-- The Go backend now maintains the behaviour these triggers provided, on BOTH
-- Postgres and SQLite (single code path — see migration design doc Phase 5 /
-- decision 3b). The triggers are dropped here so Postgres does not double-fire
-- (trigger + Go) during the cutover window. SQLite never had them (the Phase 2
-- consolidated schema omits all triggers/functions).
--
-- What is replaced and where:
--   * user_stats counters (0093)      -> services/userstats.go at each write site
--       (cards insert/soft-delete, tasks insert/soft-delete, files insert,
--        llm_query_log cost, revenue).
--   * llm_jobs.updated_at (0096)      -> every UPDATE on llm_jobs now sets
--       updated_at = CURRENT_TIMESTAMP app-side (models/job.go + jobrunner.go).
--   * chat_conversations.updated_at (0067) and llm_jobs pg_notify (0102) are
--       DEAD (no Go writer for chat_messages; no LISTEN on llm_job_queue) and
--       are dropped here as cleanup — no Go replacement needed.
--
-- The RSS + email notification triggers (0122/0123/0124) are handled in a
-- separate migration alongside their Go port.

-- 0093: user_stats triggers + functions
DROP TRIGGER IF EXISTS trg_cards_insert ON cards;
DROP TRIGGER IF EXISTS trg_cards_delete ON cards;
DROP FUNCTION IF EXISTS update_user_card_count_insert();
DROP FUNCTION IF EXISTS update_user_card_count_delete();

DROP TRIGGER IF EXISTS trg_tasks_insert ON tasks;
DROP TRIGGER IF EXISTS trg_tasks_delete ON tasks;
DROP FUNCTION IF EXISTS update_user_task_count_insert();
DROP FUNCTION IF EXISTS update_user_task_count_delete();

DROP TRIGGER IF EXISTS trg_files_insert ON files;
DROP TRIGGER IF EXISTS trg_files_delete ON files;
DROP FUNCTION IF EXISTS update_user_file_count_insert();
DROP FUNCTION IF EXISTS update_user_file_count_delete();

DROP TRIGGER IF EXISTS trg_chat_messages_insert ON chat_messages;
DROP FUNCTION IF EXISTS update_user_chat_message_count();

DROP TRIGGER IF EXISTS trg_llm_query_log_insert ON llm_query_log;
DROP FUNCTION IF EXISTS update_user_llm_cost();

DROP TRIGGER IF EXISTS trg_revenue_insert ON revenue;
DROP FUNCTION IF EXISTS update_user_revenue();

-- 0067: chat_conversations.updated_at on chat_messages insert (dead — no Go
-- chat writer).
DROP TRIGGER IF EXISTS trigger_update_conversation_timestamp ON chat_messages;
DROP FUNCTION IF EXISTS update_conversation_timestamp();

-- 0096: llm_jobs.updated_at (now set app-side on every UPDATE).
DROP TRIGGER IF EXISTS trigger_llm_jobs_updated_at ON llm_jobs;
DROP FUNCTION IF EXISTS update_llm_jobs_updated_at();

-- 0102: llm_jobs pg_notify (dead — nothing LISTENs on llm_job_queue; the job
-- queue is now an inline runner with no dequeue step).
DROP TRIGGER IF EXISTS llm_job_insert_trigger ON llm_jobs;
DROP FUNCTION IF EXISTS notify_llm_job_insert();
