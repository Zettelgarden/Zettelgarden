CREATE TABLE entity_fact_junction (
    user_id INT NOT NULL,
    entity_id INT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    fact_id INT NOT NULL REFERENCES facts(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(entity_id, fact_id)
);