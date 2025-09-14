-- Rename pinned_cards table to starred_cards
ALTER TABLE pinned_cards RENAME TO starred_cards;

-- Rename the index to match
ALTER INDEX pinned_cards_user_id_idx RENAME TO starred_cards_user_id_idx;

-- Rename pinned_searches table to starred_searches
ALTER TABLE pinned_searches RENAME TO starred_searches;

-- Rename the index to match
ALTER INDEX pinned_searches_user_id_idx RENAME TO starred_searches_user_id_idx;