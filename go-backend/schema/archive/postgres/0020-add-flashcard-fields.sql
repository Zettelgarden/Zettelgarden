ALTER TABLE cards ADD COLUMN flashcard_state text;
ALTER TABLE cards ADD COLUMN flashcard_reps int DEFAULT 0;
ALTER TABLE cards ADD COLUMN flashcard_lapses int DEFAULT 0;
ALTER TABLE cards ADD COLUMN flashcard_last_review TIMESTAMP;
ALTER TABLE cards ADD COLUMN flashcard_due TIMESTAMP;
