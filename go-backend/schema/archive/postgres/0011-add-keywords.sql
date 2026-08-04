CREATE TABLE IF NOT EXISTS keywords (
    id SERIAL PRIMARY KEY,
    card_pk INT,
    user_id INT,
    keyword TEXT,
    FOREIGN KEY (card_pk) REFERENCES cards(id)
);
