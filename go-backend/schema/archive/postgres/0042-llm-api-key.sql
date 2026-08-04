ALTER TABLE llm_providers ADD COLUMN user_id INTEGER REFERENCES users(id);
ALTER TABLE llm_providers ADD COLUMN api_key TEXT;
