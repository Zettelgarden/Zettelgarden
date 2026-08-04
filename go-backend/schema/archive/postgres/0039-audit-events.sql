-- Create audit_events table
CREATE TABLE IF NOT EXISTS audit_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    entity_id INTEGER NOT NULL,
    entity_type TEXT NOT NULL,
    action TEXT NOT NULL,
    details JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);