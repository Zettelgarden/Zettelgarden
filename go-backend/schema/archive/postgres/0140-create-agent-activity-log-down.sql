-- Rollback migration for agent activity log

-- Drop indexes
DROP INDEX IF EXISTS idx_agent_activity_action;
DROP INDEX IF EXISTS idx_agent_activity_created;
DROP INDEX IF EXISTS idx_agent_activity_agent;

-- Drop table
DROP TABLE IF EXISTS agent_activity_log;
