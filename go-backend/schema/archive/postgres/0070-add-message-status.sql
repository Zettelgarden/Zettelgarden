-- Add status field to chat_messages table for async processing
ALTER TABLE chat_messages
ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'completed'
CHECK (status IN ('pending', 'processing', 'completed', 'failed'));

-- Index for efficient status queries
CREATE INDEX IF NOT EXISTS idx_chat_messages_status ON chat_messages(status);

-- Index for conversation status queries
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_status ON chat_messages(conversation_id, status);