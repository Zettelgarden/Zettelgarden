-- Create admin_audit_log table for security auditing of admin actions
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id SERIAL PRIMARY KEY,
    admin_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL, -- e.g., "user.update", "user.delete", "mailing_list.unsubscribe"
    target_type TEXT NOT NULL, -- e.g., "user", "mailing_list", "subscription"
    target_id INTEGER, -- ID of the affected entity (can be NULL for actions without specific target)
    details JSONB NOT NULL DEFAULT '{}', -- Additional context (before/after values, reasons, etc.)
    ip_address INET, -- IP address of the admin for security investigations
    user_agent TEXT, -- User agent string for additional context
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for querying by admin user
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_admin_user_id ON admin_audit_log(admin_user_id);

-- Create index for querying by action type
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_action ON admin_audit_log(action);

-- Create index for querying by target
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_target ON admin_audit_log(target_type, target_id);

-- Create index for time-based queries
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at ON admin_audit_log(created_at DESC);

-- Add comment for documentation
COMMENT ON TABLE admin_audit_log IS 'Audit log of all admin actions for security and compliance';
COMMENT ON COLUMN admin_audit_log.admin_user_id IS 'The admin user who performed the action';
COMMENT ON COLUMN admin_audit_log.action IS 'The action performed (e.g., user.update, mailing_list.unsubscribe)';
COMMENT ON COLUMN admin_audit_log.target_type IS 'Type of entity affected (e.g., user, mailing_list)';
COMMENT ON COLUMN admin_audit_log.target_id IS 'ID of the affected entity';
COMMENT ON COLUMN admin_audit_log.details IS 'Additional context about the action (JSON)';
COMMENT ON COLUMN admin_audit_log.ip_address IS 'IP address of the admin for security investigations';
