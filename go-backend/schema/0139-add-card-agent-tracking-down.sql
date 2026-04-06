-- Rollback migration for card agent tracking

-- Remove index
DROP INDEX IF EXISTS idx_cards_created_by_agent;

-- Remove column
ALTER TABLE cards DROP COLUMN IF EXISTS created_by_agent_id;
