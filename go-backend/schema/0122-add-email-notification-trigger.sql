-- Migration: Add email notification sync trigger
-- Description: Sync email changes to notifications table
-- Created: 2026-02-16

CREATE OR REPLACE FUNCTION sync_email_notification()
RETURNS TRIGGER AS $$
DECLARE
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for unprocessed and triaged emails (not archived)
    IF NEW.status NOT IN ('unprocessed', 'triaged') THEN
        -- If archived, delete the notification
        IF NEW.status = 'archived' THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'email' AND source_id = NEW.id;
            RETURN NEW;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := COALESCE(NEW.subject, '(No subject)');
    notification_preview := LEFT(COALESCE(NEW.body_text, ''), 200);

    -- Calculate importance score
    -- Importance scores: unprocessed=10 (highest priority), triaged=5 (medium priority)
    -- This ensures new unprocessed emails appear at top of notification list
    IF NEW.status = 'unprocessed' THEN
        notification_importance := 10;
    ELSE
        notification_importance := 5;  -- triaged
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[NEW.status];

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'email', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.received_at, NEW.created_at), notification_importance, notification_tags)
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

-- Trigger on emails table
-- Uses AFTER timing to ensure the email record is fully committed before creating notification
-- This prevents notification creation for transactions that might be rolled back
DROP TRIGGER IF EXISTS email_notification_trigger ON emails;
CREATE TRIGGER email_notification_trigger
    AFTER INSERT OR UPDATE ON emails
    FOR EACH ROW
    EXECUTE FUNCTION sync_email_notification();

-- Trigger to delete notification when email is deleted
DROP TRIGGER IF EXISTS email_delete_notification_trigger ON emails;
CREATE TRIGGER email_delete_notification_trigger
    AFTER DELETE ON emails
    FOR EACH ROW
    EXECUTE FUNCTION delete_notification();
