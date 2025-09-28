-- Add primary_card_id to chat_conversations table
ALTER TABLE chat_conversations
ADD COLUMN primary_card_id INT,
ADD CONSTRAINT fk_primary_card_id
FOREIGN KEY (primary_card_id) REFERENCES cards(id) ON DELETE SET NULL;

-- Index for performance when filtering by primary card
CREATE INDEX IF NOT EXISTS idx_chat_conversations_primary_card_id ON chat_conversations(primary_card_id);