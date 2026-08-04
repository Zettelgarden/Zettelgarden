-- Migration: Add chat instructions table
-- This table stores user-specific instructions for chat conversations

CREATE TABLE IF NOT EXISTS chat_instructions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    instructions TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Create index for faster lookups by user_id
CREATE INDEX IF NOT EXISTS idx_chat_instructions_user_id ON chat_instructions(user_id);