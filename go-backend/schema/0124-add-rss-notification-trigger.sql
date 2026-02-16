-- Migration: Add RSS article notification sync trigger
-- Description: Sync RSS article changes to notifications table
-- Created: 2026-02-16

-- This trigger automatically creates and updates notifications for RSS articles.
--
-- Trigger timing: AFTER INSERT OR UPDATE to ensure article data is committed
-- before notification is created. This prevents notifications for articles
-- that might fail validation or rollback during transaction.
--
-- Importance scoring logic:
-- - Starred articles: 10 (highest priority - user explicitly marked as important)
-- - Priority feed unread: 5 (medium priority - from high-priority feed, not yet read)
-- - Other articles: 0 (low priority - no notification created)
--
-- Notification lifecycle:
-- - Created on INSERT for starred or priority-unread articles
-- - Updated on UPDATE when starring/unstarring or reading priority articles
-- - Deleted automatically when article is unstarred (if not priority-unread)
-- - Deleted when article is deleted (via delete_notification helper)

CREATE OR REPLACE FUNCTION sync_rss_notification()
RETURNS TRIGGER AS $$
DECLARE
    feed_record RECORD;
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for starred articles or priority feed articles
    SELECT * INTO feed_record FROM rss_feeds WHERE id = NEW.feed_id;

    -- Safety check: if feed doesn't exist, skip notification processing
    IF feed_record IS NULL THEN
        RETURN NEW;
    END IF;

    IF NOT (NEW.is_starred = TRUE OR (feed_record.priority = TRUE AND NEW.read = FALSE)) THEN
        -- If no longer starred, delete the notification
        IF NEW.is_starred = FALSE THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'rss' AND source_id = NEW.id;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := NEW.title;
    notification_preview := LEFT(COALESCE(NEW.content, ''), 200);

    -- Calculate importance score
    -- Starred articles: 10 (highest priority)
    -- Priority feed unread articles: 5 (medium priority)
    IF NEW.is_starred = TRUE THEN
        notification_importance := 10;
    ELSIF feed_record.priority = TRUE THEN
        notification_importance := 5;
    ELSE
        notification_importance := 0;
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[]::TEXT[];
    IF NEW.is_starred = TRUE THEN
        notification_tags := array_append(notification_tags, 'starred');
    END IF;
    IF feed_record.priority = TRUE THEN
        notification_tags := array_append(notification_tags, 'priority');
    END IF;

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'rss', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.published_at, NEW.fetched_at), notification_importance, notification_tags)
    ON CONFLICT (user_id, source_type, source_id)
    DO UPDATE SET
        title = EXCLUDED.title,
        preview = EXCLUDED.preview,
        timestamp = EXCLUDED.timestamp,
        importance_score = EXCLUDED.importance_score,
        filter_tags = EXCLUDED.filter_tags,
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on rss_articles table
-- Fires after INSERT and UPDATE operations to sync notifications
DROP TRIGGER IF EXISTS rss_article_notification_trigger ON rss_articles;
CREATE TRIGGER rss_article_notification_trigger
    AFTER INSERT OR UPDATE ON rss_articles
    FOR EACH ROW
    EXECUTE FUNCTION sync_rss_notification();

-- Trigger to delete notification when article is deleted
-- Uses the delete_notification helper which maps TG_TABLE_NAME to source_type
DROP TRIGGER IF EXISTS rss_article_delete_notification_trigger ON rss_articles;
CREATE TRIGGER rss_article_delete_notification_trigger
    AFTER DELETE ON rss_articles
    FOR EACH ROW
    EXECUTE FUNCTION delete_notification();
