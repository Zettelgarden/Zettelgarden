-- Migration: Add email_fact_junction table
-- Description: Links extracted facts to their source emails
-- Created: 2026-02-26

-- Email fact junction table - links facts extracted from emails
CREATE TABLE IF NOT EXISTS email_fact_junction (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    fact_id INT NOT NULL REFERENCES facts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, email_id, fact_id)
);

COMMENT ON TABLE email_fact_junction IS 'Links facts extracted from emails to their source emails';
COMMENT ON COLUMN email_fact_junction.email_id IS 'Reference to the source email';
COMMENT ON COLUMN email_fact_junction.fact_id IS 'Reference to the extracted fact';

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_email_fact_junction_user_id ON email_fact_junction(user_id);
CREATE INDEX IF NOT EXISTS idx_email_fact_junction_email_id ON email_fact_junction(email_id);
CREATE INDEX IF NOT EXISTS idx_email_fact_junction_fact_id ON email_fact_junction(fact_id);
