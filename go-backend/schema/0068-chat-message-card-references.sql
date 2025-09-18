-- Add referenced_cards field to chat_messages table
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS referenced_cards JSONB;

-- Index for efficient queries on referenced cards
CREATE INDEX IF NOT EXISTS idx_chat_messages_referenced_cards ON chat_messages USING GIN (referenced_cards);