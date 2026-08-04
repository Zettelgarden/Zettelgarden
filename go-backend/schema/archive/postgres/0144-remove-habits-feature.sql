-- Migration: Remove habits feature
-- Description: Drop habit_logs and habits tables; the habits feature has been
-- removed from the frontend, backend handlers/services, and the zg CLI.
-- Created: 2026-07-11

-- habit_logs depends on habits, so drop it first (CASCADE also covers this)
DROP TABLE IF EXISTS habit_logs;
DROP TABLE IF EXISTS habits;
