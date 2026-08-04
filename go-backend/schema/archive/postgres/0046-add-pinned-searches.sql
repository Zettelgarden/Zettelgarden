-- Create the pinned_searches table
CREATE TABLE pinned_searches (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  search_term TEXT NOT NULL,
  search_config JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add index for faster lookups by user_id
CREATE INDEX pinned_searches_user_id_idx ON pinned_searches(user_id);

-- Add comment to describe the table
COMMENT ON TABLE pinned_searches IS 'Stores user-pinned searches with their configuration';
