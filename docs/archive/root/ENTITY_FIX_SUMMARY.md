> **ARCHIVED** — Historical document moved to `docs/archive/` on 2026-08-08 during the documentation audit (Zettelgarden-0ui). Does not describe the current app; kept for the record.

# Entity Extraction Fix - Summary

## Problem
After disabling the "fact" feature, entities stopped being recorded during the summarization process.

## Root Cause
Entity extraction was **tied to fact processing**:
1. Facts were extracted from cards
2. Entities were extracted FROM those facts (via `ExtractSaveFactEntities`)
3. When facts were disabled, entity extraction was also disabled

The commented-out code in `ProcessEntitiesAndFacts`:
```go
// Facts processing disabled
// if len(facts) > 0 {
//     factObjs, err := h.ExtractSaveCardFacts(userID, card.ID, facts)
//     if err != nil {
//         log.Printf("Failed to save card facts: %v", err)
//     } else {
//         if err := h.ExtractSaveFactEntities(userID, card, factObjs); err != nil {
//             log.Printf("Failed to extract/save fact entities: %v", err)
//         }
//     }
// }
```

## Solution
Added **direct entity extraction from cards**, independent of facts:

### New Function: `ExtractSaveCardEntities`
- Location: `go-backend/handlers/summarize.go`
- Purpose: Extract entities directly from card title and body
- Uses: `services.FindEntities(client, card.Title, card.Body)`

### Implementation Details
1. **Extracts entities** from card using LLM (via `services.FindEntities`)
2. **Validates** entity names and descriptions
3. **Creates or updates** entities in the database:
   - If entity doesn't exist: Insert new entity with `card_pk` pointing to source card
   - If entity exists: Update description and type
4. **Links entities to cards** via `entity_card_junction` table

### Code Flow
```
ProcessEntitiesAndFacts
  ↓
Extract theses and arguments
  ↓
SaveAnalysis (theses/arguments)
  ↓
runSummarizationJobViaQueue
  ↓
ExtractSaveCardEntities ← NEW!
  ↓
LinkCardToEntityIfPossible
```

## What Changed

### File: `go-backend/handlers/summarize.go`

**Added:**
- `ExtractSaveCardEntities(userID int, card models.Card) error` - New method to extract entities from cards

**Modified:**
- `ProcessEntitiesAndFacts` - Added call to `ExtractSaveCardEntities` after summarization

## Testing
- Code compiles successfully
- Follows existing patterns from `extractSaveFactEntitiesSync`
- Uses existing validation functions from `entity.go`
- Logs extraction progress for debugging

## Entity Extraction Now Works
✅ Entities are extracted from cards during summarization
✅ Entities are created/updated in the database
✅ Entities are linked to cards
✅ Independent of fact extraction
✅ Facts feature remains disabled as requested

## Future Considerations
- Monitor entity extraction performance in logs
- Consider adding metrics for entity extraction success rate
- May want to add batch entity extraction for multiple cards
- Could add deduplication logic for similar entities
