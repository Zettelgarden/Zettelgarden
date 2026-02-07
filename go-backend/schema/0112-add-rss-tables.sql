-- Migration: Add RSS feed client tables
-- Description: Tables for RSS feeds, articles, and folders
-- Created: 2025-02-06

-- RSS Feeds table
CREATE TABLE IF NOT EXISTS rss_feeds (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    name TEXT NOT NULL,
    folder TEXT,
    auto_tags TEXT DEFAULT '',
    fetch_interval INTEGER DEFAULT 60,
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, url)
);

-- RSS Articles table
CREATE TABLE IF NOT EXISTS rss_articles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    author TEXT,
    url TEXT NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    read BOOLEAN DEFAULT false,
    UNIQUE(user_id, url)
);

-- RSS Folders table
CREATE TABLE IF NOT EXISTS rss_folders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    order_index INTEGER DEFAULT 0,
    UNIQUE(user_id, name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_rss_articles_user_feed ON rss_articles(user_id, feed_id);
CREATE INDEX IF NOT EXISTS idx_rss_articles_read ON rss_articles(user_id, read);
CREATE INDEX IF NOT EXISTS idx_rss_feeds_user ON rss_feeds(user_id);
CREATE INDEX IF NOT EXISTS idx_rss_feeds_enabled ON rss_feeds(enabled);
CREATE INDEX IF NOT EXISTS idx_rss_folders_user ON rss_folders(user_id);

-- Add comments for documentation
COMMENT ON TABLE rss_feeds IS 'RSS feed subscriptions per user';
COMMENT ON TABLE rss_articles IS 'Articles fetched from RSS feeds';
COMMENT ON TABLE rss_folders IS 'User-defined folders for organizing feeds';
COMMENT ON COLUMN rss_feeds.auto_tags IS 'Comma-separated tags to apply when converting articles to cards';
COMMENT ON COLUMN rss_feeds.fetch_interval IS 'Fetch interval in minutes';
