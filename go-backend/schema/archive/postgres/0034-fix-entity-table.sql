ALTER TABLE entity_card_junction 
ALTER COLUMN card_pk TYPE INTEGER USING (card_pk::INTEGER);
