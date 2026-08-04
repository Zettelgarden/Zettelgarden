ALTER TABLE files ADD COLUMN user_id int;
UPDATE files
SET user_id = cards.user_id
FROM cards
WHERE files.card_pk = cards.id;
