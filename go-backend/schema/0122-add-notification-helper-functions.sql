-- Migration: Add notification helper functions
-- Description: Helper function to delete notifications when source is deleted
-- Created: 2026-02-16

CREATE OR REPLACE FUNCTION delete_notification()
RETURNS TRIGGER AS $$
BEGIN
    -- Delete notification for this user and source
    -- Note: TG_TABLE_NAME returns schema-qualified name, so we map it to source_type
    DELETE FROM notifications
    WHERE user_id = OLD.user_id
      AND source_id = OLD.id
      AND source_type = CASE
          WHEN TG_TABLE_NAME LIKE '%emails' THEN 'email'
          WHEN TG_TABLE_NAME LIKE '%rss_articles' THEN 'rss'
          WHEN TG_TABLE_NAME LIKE '%tasks' THEN 'task'
          ELSE TG_TABLE_NAME
        END;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION delete_notification() IS 'Deletes corresponding notification rows when source record is deleted. Automatically determines source_type from table name.';
