CREATE TABLE IF NOT EXISTS entities (
    id SERIAL PRIMARY KEY,
    user_id int,
    name text,
    description text,
    type text,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, name)
);

CREATE TABLE IF NOT EXISTS entity_card_junction (
    id SERIAL PRIMARY KEY,
    user_id int,
    entity_id int,
    card_pk TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (entity_id) REFERENCES entities(id),
    UNIQUE(entity_id, card_pk)
);
