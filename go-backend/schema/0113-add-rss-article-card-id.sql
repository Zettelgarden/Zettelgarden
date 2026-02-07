-- Migration: Add card_id to rss_articles for tracking converted cards
-- Description: Stores the ID of the card created from an RSS article
-- Created: 2026-02-07

-- Add card_id column to rss_articles
ALTER TABLE rss_articles
ADD COLUMN card_id INT REFERENCES cards(id) ON DELETE SET NULL;

-- Create index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_rss_articles_card_id ON rss_articles(card_id);

-- Add comment for documentation
COMMENT ON COLUMN rss_articles.card_id IS 'ID of the card created from this RSS article, if converted';
