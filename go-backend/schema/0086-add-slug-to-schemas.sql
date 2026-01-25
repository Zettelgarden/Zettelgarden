-- Add slug field to schema_definitions table
ALTER TABLE schema_definitions ADD COLUMN IF NOT EXISTS slug TEXT;

-- Create unique index on slug for each owner (slugs must be unique per user)
CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_definitions_owner_slug ON schema_definitions(owner_id, slug);

-- Generate slugs for existing schemas based on names
-- This creates slugs like 'book-review', 'book-review-2', etc. for duplicates
UPDATE schema_definitions s1
SET slug = CASE
    WHEN s1_base.slug IS NULL THEN s1_base.slug_candidate
    ELSE s1_base.slug || '-' || (s1_base.row_num + 1)::TEXT
END
FROM (
    SELECT
        s.id,
        s.owner_id,
        lower(regexp_replace(trim(s.name), '[^a-zA-Z0-9]+', '-', 'g')) as slug_candidate,
        ROW_NUMBER() OVER (PARTITION BY s.owner_id, lower(regexp_replace(trim(s.name), '[^a-zA-Z0-9]+', '-', 'g')) ORDER BY s.id) as row_num,
        LAG(lower(regexp_replace(trim(s.name), '[^a-zA-Z0-9]+', '-', 'g')), 1) OVER (PARTITION BY s.owner_id ORDER BY s.id) as slug
    FROM schema_definitions s
    WHERE s.slug IS NULL
) s1_base
WHERE s1.id = s1_base.id;
