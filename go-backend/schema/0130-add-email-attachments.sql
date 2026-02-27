-- Migration: Add email attachments table
-- Description: Table for storing email attachments
-- Created: 2026-02-26

-- Email attachments table - stores attachment metadata
CREATE TABLE IF NOT EXISTS email_attachments (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    file_id INT REFERENCES files(id) ON DELETE SET NULL,
    filename TEXT NOT NULL,
    content_type TEXT,
    size BIGINT,
    s3_key TEXT,
    thumbnail_path TEXT,
    content_id TEXT, -- Content-ID for inline images (CID)
    is_inline BOOLEAN DEFAULT false, -- Whether the attachment is inline (like embedded images)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE email_attachments IS 'Stores email attachment metadata and links to files in S3';
COMMENT ON COLUMN email_attachments.email_id IS 'Reference to the email this attachment belongs to';
COMMENT ON COLUMN email_attachments.file_id IS 'Reference to the file record if saved to file vault';
COMMENT ON COLUMN email_attachments.filename IS 'Original filename of the attachment';
COMMENT ON COLUMN email_attachments.content_type IS 'MIME content type (e.g., image/jpeg, application/pdf)';
COMMENT ON COLUMN email_attachments.size IS 'Size of the attachment in bytes';
COMMENT ON COLUMN email_attachments.s3_key IS 'S3 key where the attachment is stored';
COMMENT ON COLUMN email_attachments.thumbnail_path IS 'S3 key for thumbnail (for images)';
COMMENT ON COLUMN email_attachments.content_id IS 'Content-ID header for inline attachments';
COMMENT ON COLUMN email_attachments.is_inline IS 'Whether this is an inline attachment (e.g., embedded image)';

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_email_attachments_email_id ON email_attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_user_id ON email_attachments(user_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_file_id ON email_attachments(file_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_inline ON email_attachments(email_id, is_inline);
