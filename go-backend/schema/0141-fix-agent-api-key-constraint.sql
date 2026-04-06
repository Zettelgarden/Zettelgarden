-- Migration: Fix agent API key constraint to allow revocation
-- Description: Update constraint to allow agents with NULL api_key_hash (revoked state)
-- Created: 2026-04-06

-- Drop the old constraint that requires agents to have API keys
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_has_api_key;

-- Add new constraint: agents CAN have NULL api_key_hash (for revocation)
-- This allows is_agent=TRUE with api_key_hash=NULL (revoked state)
-- No constraint needed - NULL is already allowed by the column definition
