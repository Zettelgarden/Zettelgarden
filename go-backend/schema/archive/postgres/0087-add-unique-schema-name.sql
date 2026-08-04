-- Add unique constraint on (owner_id, name) for schema_definitions
-- This prevents users from creating schemas with duplicate names

-- First, handle any existing duplicate names by renaming them (keeping the oldest)
-- Find duplicates and rename them with a suffix
UPDATE schema_definitions s1
SET name = s1.name || ' (' || (s1_base.dup_num + 1)::TEXT || ')'
FROM (
    SELECT
        s.id,
        s.name,
        s.owner_id,
        ROW_NUMBER() OVER (PARTITION BY s.owner_id, lower(trim(s.name)) ORDER BY s.id) as dup_num
    FROM schema_definitions s
    WHERE s.is_deleted = FALSE
) s1_base
WHERE s1.id = s1_base.id
AND s1_base.dup_num > 0;

-- Create unique index on name for each owner (case-insensitive, trimmed)
-- This uses a partial index to only enforce uniqueness on non-deleted schemas
CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_definitions_owner_name
ON schema_definitions(owner_id, lower(trim(name)))
WHERE is_deleted = FALSE;
