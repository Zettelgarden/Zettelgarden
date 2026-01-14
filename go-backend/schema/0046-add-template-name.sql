-- Add name field to card_templates for user-friendly template display names
-- The name is shown in template lists/dropdowns, while title becomes the card title when applied

ALTER TABLE card_templates ADD COLUMN name TEXT NOT NULL DEFAULT '';

-- Populate name with existing title values for existing templates
UPDATE card_templates SET name = title WHERE name = '';
