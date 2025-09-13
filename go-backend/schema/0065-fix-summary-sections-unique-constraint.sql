-- Drop the unique constraint that prevents multiple sections with same title
ALTER TABLE summary_sections DROP CONSTRAINT IF EXISTS summary_sections_summarization_id_section_title_key;

-- Add section_order column to maintain order of sections
ALTER TABLE summary_sections ADD COLUMN IF NOT EXISTS section_order INTEGER DEFAULT 0;