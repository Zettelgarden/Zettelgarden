-- Add schema_definitions table for structured data feature
CREATE TABLE IF NOT EXISTS schema_definitions (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) <= 255),
    owner_id INT NOT NULL,
    fields JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- Add card_schema_id and structured_data columns to cards table
ALTER TABLE cards ADD COLUMN IF NOT EXISTS card_schema_id INT;
ALTER TABLE cards ADD COLUMN IF NOT EXISTS structured_data JSONB;

-- Add foreign key constraint for card_schema_id
ALTER TABLE cards ADD CONSTRAINT cards_card_schema_id_fkey
    FOREIGN KEY (card_schema_id) REFERENCES schema_definitions(id);

-- Create index on schema_definitions owner_id for performance
CREATE INDEX IF NOT EXISTS idx_schema_definitions_owner_id ON schema_definitions(owner_id);

-- Create index on cards card_schema_id for performance
CREATE INDEX IF NOT EXISTS idx_cards_card_schema_id ON cards(card_schema_id);
