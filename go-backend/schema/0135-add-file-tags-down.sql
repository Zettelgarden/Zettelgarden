-- Rollback migration for file tags

DROP TABLE IF EXISTS files_tags;
DROP TABLE IF EXISTS file_tags;
ALTER TABLE files DROP COLUMN IF EXISTS extracted_text;
ALTER TABLE files DROP COLUMN IF EXISTS description;
