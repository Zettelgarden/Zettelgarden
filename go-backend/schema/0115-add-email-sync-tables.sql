-- Migration: Add email sync tables
-- Description: Tables for Fastmail email synchronization via JMAP
-- Created: 2026-02-15

-- Email accounts table - stores Fastmail credentials
CREATE TABLE IF NOT EXISTS email_accounts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_address TEXT NOT NULL,
    jmap_server_url TEXT NOT NULL DEFAULT 'https://api.fastmail.com/jmap/session',
    app_password_encrypted TEXT,
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_status TEXT DEFAULT 'active' CHECK (sync_status IN ('active', 'error', 'disabled')),
    jmap_state TEXT, -- JMAP state token for incremental sync
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, email_address)
);

COMMENT ON TABLE email_accounts IS 'Stores Fastmail account credentials for email synchronization';
COMMENT ON COLUMN email_accounts.jmap_server_url IS 'JMAP server endpoint URL';
COMMENT ON COLUMN email_accounts.app_password_encrypted IS 'Encrypted app password for authentication';
COMMENT ON COLUMN email_accounts.is_active IS 'Whether the account is actively syncing';
COMMENT ON COLUMN email_accounts.sync_status IS 'Current sync status: active, error, or disabled';
COMMENT ON COLUMN email_accounts.jmap_state IS 'JMAP state token for incremental synchronization';

-- Emails table - stores synced emails
CREATE TABLE IF NOT EXISTS emails (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_account_id INT REFERENCES email_accounts(id) ON DELETE SET NULL,
    message_id TEXT NOT NULL, -- JMAP message ID
    thread_id TEXT, -- JMAP thread ID
    subject TEXT,
    from_address TEXT,
    from_name TEXT,
    to_addresses TEXT,
    body_text TEXT,
    body_html TEXT,
    received_at TIMESTAMP WITH TIME ZONE,
    folder TEXT DEFAULT 'Inbox',
    status TEXT DEFAULT 'unprocessed' CHECK (status IN ('unprocessed', 'triaged', 'reviewed', 'archived', 'deleted', 'converted')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);

COMMENT ON TABLE emails IS 'Stores synchronized email messages from Fastmail';
COMMENT ON COLUMN emails.message_id IS 'JMAP message identifier';
COMMENT ON COLUMN emails.thread_id IS 'JMAP thread identifier for conversation tracking';
COMMENT ON COLUMN emails.folder IS 'Email folder/category (e.g., Inbox, Archive)';
COMMENT ON COLUMN emails.status IS 'Processing status: unprocessed, triaged, reviewed, archived, deleted, or converted';

-- Email triage decisions table - AI recommendations
CREATE TABLE IF NOT EXISTS email_triage_decisions (
    id SERIAL PRIMARY KEY,
    email_id INT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    decision TEXT NOT NULL CHECK (decision IN ('archive', 'delete', 'keep', 'convert_to_card')),
    confidence FLOAT NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    reasoning TEXT,
    is_auto_executed BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE email_triage_decisions IS 'Stores AI-powered triage recommendations for emails';
COMMENT ON COLUMN email_triage_decisions.decision IS 'Recommended action: archive, delete, keep, or convert_to_card';
COMMENT ON COLUMN email_triage_decisions.confidence IS 'Confidence score between 0 and 1';
COMMENT ON COLUMN email_triage_decisions.reasoning IS 'AI-generated explanation for the decision';
COMMENT ON COLUMN email_triage_decisions.is_auto_executed IS 'Whether the decision was automatically applied';

-- Email card links table - links between emails and converted cards
CREATE TABLE IF NOT EXISTS email_card_links (
    id SERIAL PRIMARY KEY,
    email_id INT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    card_id INT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(email_id, card_id)
);

COMMENT ON TABLE email_card_links IS 'Links emails to cards created from email content';
COMMENT ON COLUMN email_card_links.email_id IS 'Reference to the source email';
COMMENT ON COLUMN email_card_links.card_id IS 'Reference to the created card';

-- Indexes for performance

-- Email account lookups
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_id ON email_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_is_active ON email_accounts(is_active);
CREATE INDEX IF NOT EXISTS idx_email_accounts_active_user ON email_accounts(user_id) WHERE is_active = true;

-- Email query performance
CREATE INDEX IF NOT EXISTS idx_emails_user_account ON emails(user_id, email_account_id);
CREATE INDEX IF NOT EXISTS idx_emails_user_status ON emails(user_id, status);
CREATE INDEX IF NOT EXISTS idx_emails_user_folder ON emails(user_id, folder);
CREATE INDEX IF NOT EXISTS idx_emails_received_at ON emails(received_at DESC);

-- Triage decision lookups
CREATE INDEX IF NOT EXISTS idx_email_triage_decisions_email_id ON email_triage_decisions(email_id);

-- Card link lookups
CREATE INDEX IF NOT EXISTS idx_email_card_links_email_id ON email_card_links(email_id);
CREATE INDEX IF NOT EXISTS idx_email_card_links_card_id ON email_card_links(card_id);
