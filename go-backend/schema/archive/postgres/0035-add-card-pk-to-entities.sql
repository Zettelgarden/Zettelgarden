ALTER TABLE entities
ADD COLUMN card_pk INTEGER,
ADD CONSTRAINT fk_card_pk FOREIGN KEY (card_pk) REFERENCES cards(id); 