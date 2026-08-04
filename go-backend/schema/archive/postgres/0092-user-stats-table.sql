-- Create user_stats table for caching aggregate user statistics
-- This eliminates expensive correlated subqueries in QueryUsers

CREATE TABLE IF NOT EXISTS user_stats (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    card_count INTEGER NOT NULL DEFAULT 0,
    task_count INTEGER NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    chat_message_count INTEGER NOT NULL DEFAULT 0,
    llm_cost_usd NUMERIC(10, 4) NOT NULL DEFAULT 0.0000,
    revenue_cents INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for queries that might filter by specific stats
CREATE INDEX IF NOT EXISTS idx_user_stats_card_count ON user_stats(card_count);
CREATE INDEX IF NOT EXISTS idx_user_stats_revenue ON user_stats(revenue_cents DESC);

-- Add comment for documentation
COMMENT ON TABLE user_stats IS 'Cached aggregate statistics for users - updated via triggers';
