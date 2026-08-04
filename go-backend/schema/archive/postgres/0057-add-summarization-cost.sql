-- Migration to add token usage and cost fields to summarizations table

ALTER TABLE summarizations
ADD COLUMN prompt_tokens INTEGER DEFAULT 0,
ADD COLUMN completion_tokens INTEGER DEFAULT 0,
ADD COLUMN total_tokens INTEGER DEFAULT 0,
ADD COLUMN cost DOUBLE PRECISION DEFAULT 0;
