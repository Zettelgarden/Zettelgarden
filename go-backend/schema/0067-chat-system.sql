-- Chat conversations table
CREATE TABLE IF NOT EXISTS chat_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL,
    title TEXT,
    model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
    system_prompt TEXT,
    starred BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Chat messages table (extends existing chat_completions for tool calling)
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT,
    tool_calls JSONB, -- Store tool calls as JSON
    tool_call_id TEXT, -- For tool response messages
    sequence_number INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE,
    UNIQUE (conversation_id, sequence_number)
);

-- Track tool usage for analytics and quotas
CREATE TABLE IF NOT EXISTS chat_tool_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL,
    conversation_id UUID NOT NULL,
    message_id UUID NOT NULL,
    tool_name TEXT NOT NULL,
    tool_arguments JSONB,
    tool_result JSONB,
    execution_time_ms INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE
);

-- Usage quotas for rate limiting
CREATE TABLE IF NOT EXISTS chat_usage_quotas (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    quota_type TEXT NOT NULL CHECK (quota_type IN ('messages_per_day', 'tool_calls_per_day', 'conversations_per_day')),
    current_usage INT DEFAULT 0,
    max_limit INT NOT NULL,
    reset_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, quota_type, reset_date)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_updated_at ON chat_conversations(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sequence ON chat_messages(conversation_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_chat_tool_calls_user_id ON chat_tool_calls(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_tool_calls_conversation_id ON chat_tool_calls(conversation_id);
CREATE INDEX IF NOT EXISTS idx_chat_usage_quotas_user_reset ON chat_usage_quotas(user_id, reset_date);

-- Trigger to update conversation updated_at when messages are added
CREATE OR REPLACE FUNCTION update_conversation_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE chat_conversations
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_conversation_timestamp
    AFTER INSERT ON chat_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_conversation_timestamp();