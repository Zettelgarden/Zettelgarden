-- Migration: Create agent activity log table
-- Description: Track all agent actions for auditability and multi-user support
-- Created: 2026-04-06

-- Create agent activity log table
CREATE TABLE IF NOT EXISTS agent_activity_log (
    id SERIAL PRIMARY KEY,
    agent_id INT NULL REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INT,
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for common queries
-- Agent ID index: Optimizes queries filtering by specific agent (e.g., "recent activity by agent X")
CREATE INDEX IF NOT EXISTS idx_agent_activity_agent ON agent_activity_log(agent_id);
-- Created_at index (DESC): Optimizes ordering by timestamp for pagination and "recent activity" queries
CREATE INDEX IF NOT EXISTS idx_agent_activity_created ON agent_activity_log(created_at DESC);
-- Action index: Optimizes filtering by action type (e.g., "all create_card actions")
CREATE INDEX IF NOT EXISTS idx_agent_activity_action ON agent_activity_log(action);
-- Partial index on target_id: Speed up queries filtering by target entity (used in WHERE target_id = X)
-- Note: Not created as partial index since target_id can be NULL, but query performance is still good
-- Consider adding idx_agent_activity_target(target_type, target_id) if target-based queries become common

-- Add table and column comments for documentation
COMMENT ON TABLE agent_activity_log IS 'Audit log of all AI agent actions';
COMMENT ON COLUMN agent_activity_log.agent_id IS 'The agent user who performed the action (NULL if agent was deleted/revoked)';
COMMENT ON COLUMN agent_activity_log.action IS 'Type of action performed (e.g., create_card, update_card, delete_card)';
COMMENT ON COLUMN agent_activity_log.target_type IS 'Type of target entity (e.g., card, task, file)';
COMMENT ON COLUMN agent_activity_log.target_id IS 'ID of the target entity (nullable for actions without specific targets)';
COMMENT ON COLUMN agent_activity_log.details IS 'Additional JSON details about the action';
COMMENT ON COLUMN agent_activity_log.created_at IS 'When the action was performed';
