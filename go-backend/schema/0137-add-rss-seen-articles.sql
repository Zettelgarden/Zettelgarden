-- Migration: Add RSS seen articles tracking
-- Description: Track seen article URLs to prevent re-syncing after cleanup
-- Created: 2026-03-15
-- 
-- Problem: When RSS articles are cleaned up (deleted), the next RSS fetch
-- would re-add them because we only checked rss_articles for existence.
-- 
-- Solution: Track seen URLs separately in rss_seen_articles, which is never
-- cleaned up. The fetch job checks this table before inserting new articles.

-- Create table to track seen article URLs
CREATE TABLE IF NOT EXISTS rss_seen_articles (
    id SERIAL PRIMARY KEY,
    feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    first_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(feed_id, url)
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_rss_seen_articles_feed_url ON rss_seen_articles(feed_id, url);

-- Backfill existing articles
INSERT INTO rss_seen_articles (feed_id, url, first_seen_at)
SELECT feed_id, url, fetched_at
FROM rss_articles
ON CONFLICT (feed_id, url) DO NOTHING;

-- Add comment for documentation
COMMENT ON TABLE rss_seen_articles IS 'Tracks all article URLs ever seen per feed to prevent re-syncing after cleanup';
