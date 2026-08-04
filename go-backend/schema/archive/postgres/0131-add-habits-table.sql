-- Migration: Add habits table
-- Description: Table for tracking recurring habits with customizable frequencies
-- Created: 2026-03-05

CREATE TABLE IF NOT EXISTS habits (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    frequency VARCHAR(20) NOT NULL DEFAULT 'daily' CHECK (frequency IN ('daily', 'weekly', 'monthly', 'custom')),
    custom_days JSONB,
    icon VARCHAR(50),
    color VARCHAR(7),
    position INTEGER DEFAULT 0,
    linked_task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_habits_user_id ON habits(user_id);
CREATE INDEX IF NOT EXISTS idx_habits_position ON habits(user_id, position);
CREATE INDEX IF NOT EXISTS idx_habits_linked_task ON habits(linked_task_id);

-- Add table and column comments for documentation
COMMENT ON TABLE habits IS 'User-defined habits for recurring behaviors and goals';
COMMENT ON COLUMN habits.title IS 'Name/title of the habit';
COMMENT ON COLUMN habits.description IS 'Optional description or notes about the habit';
COMMENT ON COLUMN habits.frequency IS 'How often the habit repeats: daily, weekly, monthly, or custom';
COMMENT ON COLUMN habits.custom_days IS 'JSONB for custom schedule (e.g., specific days of week for weekly)';
COMMENT ON COLUMN habits.icon IS 'Optional icon name for UI display';
COMMENT ON COLUMN habits.color IS 'Optional hex color code for UI display (e.g., #FF5733)';
COMMENT ON COLUMN habits.position IS 'Sort order for display in UI';
COMMENT ON COLUMN habits.linked_task_id IS 'Optional reference to a recurring task';
