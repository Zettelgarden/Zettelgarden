-- Migration: Backfill existing data into notifications
-- Description: Populate notifications table with existing emails and RSS articles
-- Created: 2026-02-16

-- Backfill existing unprocessed/triaged emails as notifications
INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
SELECT
    user_id,
    'email'::VARCHAR,
    id,
    COALESCE(subject, '(No subject)'),
    LEFT(COALESCE(body_text, ''), 200),
    COALESCE(received_at, created_at),
    CASE WHEN status = 'unprocessed' THEN 10 ELSE 5 END,
    ARRAY[status]
FROM emails
WHERE status IN ('unprocessed', 'triaged')
ON CONFLICT (user_id, source_type, source_id) DO NOTHING;

-- Backfill existing starred articles and priority feed articles
INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
SELECT
    a.user_id,
    'rss'::VARCHAR,
    a.id,
    a.title,
    LEFT(COALESCE(a.content, ''), 200),
    COALESCE(a.published_at, a.fetched_at),
    CASE
        WHEN a.is_starred = TRUE THEN 10
        WHEN f.priority = TRUE THEN 5
        ELSE 0
    END,
    CASE
        WHEN a.is_starred = TRUE THEN ARRAY['starred']
        WHEN f.priority = TRUE THEN ARRAY['priority']
        ELSE ARRAY[]::TEXT[]
    END
FROM rss_articles a
JOIN rss_feeds f ON a.feed_id = f.id
WHERE a.is_starred = TRUE OR (f.priority = TRUE AND a.read = FALSE)
ON CONFLICT (user_id, source_type, source_id) DO NOTHING;
