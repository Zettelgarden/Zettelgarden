-- Document the semantics of card_pk NULL in summarizations table
--
-- When card_pk IS NULL, the summarization is a manual/standalone analysis
-- created by the user without linking to a specific card. These summaries:
--   - Only appear in the list view (all summaries)
--   - Are excluded from card-specific queries
--   - Cannot be queried via GetSummariesByCardRoute
--
-- When card_pk IS NOT NULL, the summarization is linked to a specific card
-- and will appear in both the list view and card-specific queries.

COMMENT ON COLUMN summarizations.card_pk IS 'Reference to the card being analyzed. NULL indicates a manual/standalone summarization created without a card, which is excluded from card-specific queries.';
