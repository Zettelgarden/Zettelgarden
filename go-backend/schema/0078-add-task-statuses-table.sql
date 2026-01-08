-- Migration: Create task_statuses table for user-configurable task statuses
-- This allows users to customize their task workflow statuses

CREATE TABLE IF NOT EXISTS task_statuses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,           -- identifier like "todo", "custom_status"
    display_name VARCHAR(100) NOT NULL,  -- display text like "To Do", "Custom Status"
    color VARCHAR(7) NOT NULL,           -- hex color like "#6B7280"
    icon VARCHAR(10),                    -- emoji like "⭕"
    position INTEGER NOT NULL,           -- for ordering in UI (lower = earlier)
    is_default BOOLEAN DEFAULT FALSE,    -- whether this is the default status for new tasks
    is_complete_state BOOLEAN DEFAULT FALSE, -- whether this status means "complete"
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, name),
    UNIQUE(user_id, position)
);

CREATE INDEX idx_task_statuses_user ON task_statuses(user_id);
CREATE INDEX idx_task_statuses_position ON task_statuses(user_id, position);

-- Seed default statuses for all existing users
INSERT INTO task_statuses (user_id, name, display_name, color, icon, position, is_default, is_complete_state)
SELECT
    u.id,
    'todo',
    'To Do',
    '#6B7280',
    '⭕',
    0,
    TRUE,  -- todo is the default status
    FALSE
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM task_statuses ts WHERE ts.user_id = u.id AND ts.name = 'todo'
);

INSERT INTO task_statuses (user_id, name, display_name, color, icon, position, is_default, is_complete_state)
SELECT
    u.id,
    'in_progress',
    'In Progress',
    '#3B82F6',
    '🔄',
    1,
    FALSE,
    FALSE
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM task_statuses ts WHERE ts.user_id = u.id AND ts.name = 'in_progress'
);

INSERT INTO task_statuses (user_id, name, display_name, color, icon, position, is_default, is_complete_state)
SELECT
    u.id,
    'blocked',
    'Blocked',
    '#EF4444',
    '🚫',
    2,
    FALSE,
    FALSE
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM task_statuses ts WHERE ts.user_id = u.id AND ts.name = 'blocked'
);

INSERT INTO task_statuses (user_id, name, display_name, color, icon, position, is_default, is_complete_state)
SELECT
    u.id,
    'done',
    'Done',
    '#10B981',
    '✅',
    3,
    FALSE,
    TRUE  -- done is the complete state
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM task_statuses ts WHERE ts.user_id = u.id AND ts.name = 'done'
);
