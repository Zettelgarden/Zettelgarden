CREATE TABLE IF NOT EXISTS flashcard_reviews (
    id INT,
    card_pk INT,
    user_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    rating INT,
    FOREIGN KEY (card_pk) REFERENCES cards(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
