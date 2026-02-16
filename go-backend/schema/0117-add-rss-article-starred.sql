-- Migration: Add is_starred to rss_articles
-- Description: Add boolean column for starring articles and index for filtering
-- Created: 2026-02-15

ALTER TABLE rss_articles ADD COLUMN IF NOT EXISTS is_starred BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_rss_articles_starred ON rss_articles(user_id, is_starred);

COMMENT ON COLUMN rss_articles.is_starred IS 'Whether the article is starred by the user';
