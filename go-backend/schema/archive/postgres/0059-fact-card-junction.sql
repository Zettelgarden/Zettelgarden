CREATE TABLE fact_card_junction (
    fact_id   INT NOT NULL REFERENCES facts(id) ON DELETE CASCADE,
    card_pk   INT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id   INT NOT NULL,
    is_origin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (fact_id, card_pk)
);

-- Seed existing facts into the junction table
INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
SELECT id, card_pk, user_id, TRUE, created_at, updated_at
FROM facts
ON CONFLICT (fact_id, card_pk) DO NOTHING;