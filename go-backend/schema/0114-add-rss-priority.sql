--- Migration: Add priority column to rss_feeds
--- Description: Add boolean flag for manually prioritized feeds
--- Created: 2026-02-11

ALTER TABLE rss_feeds ADD COLUMN priority BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN rss_feeds.priority IS 'Manual priority flag for smart feed - feeds marked as priority always rank higher';
