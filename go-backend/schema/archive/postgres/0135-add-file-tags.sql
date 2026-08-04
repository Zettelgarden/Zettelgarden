-- Add description and extracted_text columns to files
ALTER TABLE files ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE files ADD COLUMN IF NOT EXISTS extracted_text TEXT;

-- Create file_tags table
CREATE TABLE IF NOT EXISTS file_tags (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_tag UNIQUE(user_id, name)
);

-- Create files_tags junction table
CREATE TABLE IF NOT EXISTS files_tags (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES file_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_file_tags_user_id ON file_tags(user_id);
CREATE INDEX IF NOT EXISTS idx_files_tags_file_id ON files_tags(file_id);
CREATE INDEX IF NOT EXISTS idx_files_tags_tag_id ON files_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_files_extracted_text ON files USING gin(to_tsvector('english', extracted_text));
