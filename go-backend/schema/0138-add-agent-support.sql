-- Migration: Add agent support to users table
-- Description: Enable AI agents as special user accounts with API key auth
-- Created: 2026-04-06

-- Add agent support columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_agent BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_user_id INT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS api_key_hash CHAR(60) NULL;

-- Add constraint: agents cannot be admins
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_not_admin;
ALTER TABLE users ADD CONSTRAINT check_agent_not_admin 
    CHECK (NOT (is_agent = TRUE AND is_admin = TRUE));

-- Add constraint: agents must have API keys
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_has_api_key;
ALTER TABLE users ADD CONSTRAINT check_agent_has_api_key
    CHECK (NOT is_agent OR api_key_hash IS NOT NULL);

-- Indexes for faster lookups of agents by owner
CREATE INDEX IF NOT EXISTS idx_users_owner ON users(owner_user_id) WHERE is_agent = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_agent ON users(is_agent) WHERE is_agent = TRUE;

-- Add table and column comments for documentation
COMMENT ON COLUMN users.is_agent IS 'Whether this user is an AI agent';
COMMENT ON COLUMN users.owner_user_id IS 'For agents: the user who owns this agent';
COMMENT ON COLUMN users.api_key_hash IS 'For agents: hashed API key for authentication';
