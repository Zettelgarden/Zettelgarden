-- Migration: Fix habits frequency check constraint
-- Description: Change 'custom' to 'custom_days' to match the codebase
-- Created: 2026-03-08

-- Drop the old constraint and add the corrected one
ALTER TABLE habits DROP CONSTRAINT IF EXISTS habits_frequency_check;
ALTER TABLE habits ADD CONSTRAINT habits_frequency_check 
    CHECK (frequency IN ('daily', 'weekly', 'monthly', 'custom_days'));
