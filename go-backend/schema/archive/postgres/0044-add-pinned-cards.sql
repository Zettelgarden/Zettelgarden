CREATE TABLE pinned_cards (
    id SERIAL PRIMARY KEY,
    card_pk INTEGER REFERENCES cards(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(card_pk, user_id)
);

-- Add index for faster lookups by user_id
CREATE INDEX pinned_cards_user_id_idx ON pinned_cards(user_id);
