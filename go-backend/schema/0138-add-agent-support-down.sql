-- Rollback migration for agent support

-- Remove constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_has_api_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_not_admin;

-- Remove indexes
DROP INDEX IF EXISTS idx_users_agent;
DROP INDEX IF EXISTS idx_users_owner;

-- Remove columns
ALTER TABLE users DROP COLUMN IF EXISTS api_key_hash;
ALTER TABLE users DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS is_agent;
