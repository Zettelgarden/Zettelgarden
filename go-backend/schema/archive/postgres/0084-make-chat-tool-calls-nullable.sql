-- Make conversation_id and message_id nullable in chat_tool_calls
-- This allows logging subagent tool calls that don't have a parent message

-- Drop foreign key constraints
ALTER TABLE chat_tool_calls DROP CONSTRAINT IF EXISTS chat_tool_calls_conversation_id_fkey;
ALTER TABLE chat_tool_calls DROP CONSTRAINT IF EXISTS chat_tool_calls_message_id_fkey;

-- Make columns nullable
ALTER TABLE chat_tool_calls ALTER COLUMN conversation_id DROP NOT NULL;
ALTER TABLE chat_tool_calls ALTER COLUMN message_id DROP NOT NULL;

-- Re-add foreign key constraints (now that columns are nullable, the constraints work for NULL values)
ALTER TABLE chat_tool_calls ADD CONSTRAINT chat_tool_calls_conversation_id_fkey
    FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE;

ALTER TABLE chat_tool_calls ADD CONSTRAINT chat_tool_calls_message_id_fkey
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE;
