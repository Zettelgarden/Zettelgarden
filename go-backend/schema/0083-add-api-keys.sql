-- Create API keys table for user-generated API keys
-- These provide long-lived access for programmatic use beyond JWT token expiration
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,           -- User-defined key name (e.g., "My Script", "Integration")
    key_hash VARCHAR(255) NOT NULL, -- Bcrypt hash of the actual API key
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE NULL,
    revoked_at TIMESTAMP WITH TIME ZONE NULL,
    is_active BOOLEAN DEFAULT true,
    description TEXT,                     -- Optional description/note

    -- Ensure revocation timestamp is set when deactivated
    CONSTRAINT revoked_at_required_when_inactive CHECK (
        (is_active = true AND revoked_at IS NULL) OR
        (is_active = false AND revoked_at IS NOT NULL)
    )
);

-- Create index for efficient API key lookups by user
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- Create partial unique index to ensure unique active keys per user
-- This allows revoked keys to have the same name (since they won't be active)
CREATE UNIQUE INDEX idx_unique_active_key_name_per_user
ON api_keys (user_id, name) WHERE is_active = true;

-- Create index for efficient API key lookups during authentication
-- Note: We're not indexing key_hash since that's used for bcrypt comparison,
-- which would be more efficient as a sequential scan anyway