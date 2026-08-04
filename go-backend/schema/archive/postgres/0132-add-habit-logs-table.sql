-- Migration: Add habit_logs table
-- Description: Table for tracking habit completion history and notes
-- Created: 2026-03-05

CREATE TABLE IF NOT EXISTS habit_logs (
    id SERIAL PRIMARY KEY,
    habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    completed_at TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_habit_logs_habit_completed ON habit_logs(habit_id, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_habit_logs_user_completed ON habit_logs(user_id, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_habit_logs_habit_date ON habit_logs(habit_id, DATE(completed_at AT TIME ZONE 'UTC'));

-- Add table and column comments for documentation
COMMENT ON TABLE habit_logs IS 'Individual habit completion records with timestamps and notes';
COMMENT ON COLUMN habit_logs.habit_id IS 'Reference to the habit being logged';
COMMENT ON COLUMN habit_logs.user_id IS 'User who completed the habit';
COMMENT ON COLUMN habit_logs.completed_at IS 'When the habit was completed';
COMMENT ON COLUMN habit_logs.notes IS 'Optional notes about this specific completion';
COMMENT ON COLUMN habit_logs.created_at IS 'Record creation timestamp';
