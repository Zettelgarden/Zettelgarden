-- Add indexes for summarizations table to improve query performance
-- These indexes support the common query patterns in the summarization system

-- Index for GetSummariesByCardRoute and GetCardAnalysis queries
-- Supports: WHERE user_id = $1 AND card_pk = $2 ORDER BY created_at DESC
CREATE INDEX idx_summarizations_user_card_created ON summarizations(user_id, card_pk, created_at DESC);

-- Index for ListSummarizationsRoute query
-- Supports: WHERE user_id = $1 ORDER BY created_at DESC
CREATE INDEX idx_summarizations_user_created ON summarizations(user_id, created_at DESC);
