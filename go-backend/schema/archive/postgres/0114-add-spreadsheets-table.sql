-- Migration: Add spreadsheets table
-- Description: Dedicated table for spreadsheet data attached to cards
-- Created: 2026-02-11

CREATE TABLE IF NOT EXISTS spreadsheets (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  rows INTEGER NOT NULL DEFAULT 5,
  cols INTEGER NOT NULL DEFAULT 5,
  data JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spreadsheets_card_id ON spreadsheets(card_id);
CREATE INDEX IF NOT EXISTS idx_spreadsheets_user_id ON spreadsheets(user_id);
CREATE INDEX IF NOT EXISTS idx_spreadsheets_user_card ON spreadsheets(user_id, card_id);

-- Add comments for documentation
COMMENT ON TABLE spreadsheets IS 'Spreadsheet data attached to cards, migrated from card body JSON blocks';
COMMENT ON COLUMN spreadsheets.user_id IS 'Owner of the spreadsheet';
COMMENT ON COLUMN spreadsheets.card_id IS 'Card that contains this spreadsheet';
COMMENT ON COLUMN spreadsheets.name IS 'Display name of the spreadsheet';
COMMENT ON COLUMN spreadsheets.rows IS 'Number of rows in the spreadsheet';
COMMENT ON COLUMN spreadsheets.cols IS 'Number of columns in the spreadsheet';
COMMENT ON COLUMN spreadsheets.data IS 'Cell data stored as JSONB';
