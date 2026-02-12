# Related Cards Feature Design

## Overview
Add a "Related Cards" section to the ViewPage sidebar that shows cards related to the currently viewed card. Relatedness is determined by combining multiple signals: shared entities, shared tags, and semantic similarity.

## Architecture

### Backend Endpoint
**New endpoint**: `GET /api/cards/{id}/related`

**Flow**:
1. Fetch source card to get its entities, tags, and content
2. Query for cards sharing entities (score +3 per entity)
3. Query for cards sharing tags (score +1 per tag)
4. Query Typesense for semantically similar cards (add cosine similarity)
5. Merge results, deduplicate, sort by combined score
6. Exclude: current card, parent, children, siblings (already visible)
7. Return top 5-10 cards

### Frontend Changes
- Add `relatedCards: RelatedCard[]` to ViewPageContainer data
- Add new `RelatedCards` component using `CardItem` for consistency
- Add `RelatedCards` section to `ViewPageSidePanels` (below Linked Entities)

## Scoring Formula

**Signals** (configurable via env vars):
- Shared entity: +3 points per entity (`RELATED_ENTITY_WEIGHT=3`)
- Shared tag: +1 per tag (`RELATED_TAG_WEIGHT=1`)
- Semantic similarity: add vector cosine similarity (0-1 range)
- Exclude: current card itself, parent/children, siblings (already visible)

**Response type**:
```typescript
interface RelatedCard {
  card: PartialCard;
  score: number;
  reasons: string[];  // e.g., ["Python", "Machine Learning"]
}
```

## Backend Implementation

### New Handler (`go-backend/handlers/cards.go`)
```go
type RelatedCard struct {
    Card       models.PartialCard
    Score       float64
    Reasons     []string  // e.g., ["Python", "Machine Learning"]
}

func (h *Handler) GetRelatedCards(w http.ResponseWriter, r *http.Request) {
    cardID := mux.Vars(r)["id"]
    userID := r.Context().Value("current_user").(int)

    // 1. Get source card with entities/tags
    // 2. Find cards with shared entities (score +3 each)
    // 3. Find cards with shared tags (score +1 each)
    // 4. Query Typesense for semantic similarity (add cosine score)
    // 5. Merge, dedupe, exclude current/parent/children/siblings
    // 6. Return top 5-10 sorted by score
}
```

### SQL Queries
- **Entity overlap**: `JOIN entity_card_junction` to find cards sharing any of source card's entities
- **Tag overlap**: `JOIN card_tags` to find cards sharing any of source card's tags

### New Route (`go-backend/routes/cards.go`)
```go
router.HandleFunc("/cards/{id}/related", handlers.GetAuthHandler(h.GetRelatedCards)).Methods("GET")
```

## Frontend Implementation

### New Component (`zettelkasten-front/src/components/cards/RelatedCards.tsx`)
```tsx
interface RelatedCardsProps {
  relatedCards: RelatedCard[];
  onCardClick: (cardId: string) => void;
}

export function RelatedCards({ relatedCards, onCardClick }: RelatedCardsProps) {
  if (relatedCards.length === 0) return null;

  return (
    <div>
      <HeaderSubSection text="Related Cards" />
      <ul className="mt-2 space-y-1">
        {relatedCards.map(rc => (
          <li key={rc.card.id} onClick={() => onCardClick(rc.card.id)}>
            <CardItem card={rc.card} />
            {rc.score && (
              <span className="text-xs text-gray-400 ml-2">{rc.score.toFixed(1)}</span>
            )}
          </li>
        ))}
      </ul>
      <hr className="my-4" />
    </div>
  );
}
```

### API Function (`zettelkasten-front/src/api/cards.ts`)
```typescript
export async function getRelatedCards(cardId: string): Promise<RelatedCard[]> {
  const response = await fetch(`/api/cards/${cardId}/related`);
  return response.json();
}
```

### Container Updates
- **ViewPageContainer**: Fetch related cards when card loads, add to state
- **ViewPageSidePanels**: Include RelatedCards component below Linked Entities

## Data Flow

1. ViewPageContainer fetches card data (existing behavior)
2. After card loads, fetch `/api/cards/{id}/related`
3. Cache results in state (no refetch on re-renders)
4. Pass to ViewPageSidePanels → RelatedCards component

## Error Handling

- **Backend**: If Typesense unavailable, skip semantic search (still return entity/tag matches)
- **Frontend**: If API fails, silently hide RelatedCards section (non-critical feature)
- **Edge cases**: Empty results → don't render section

## Future Configuration Options
- Score weights via env vars: `RELATED_ENTITY_WEIGHT=3`, `RELATED_TAG_WEIGHT=1`
- Max results: `RELATED_MAX_RESULTS=10`

## Testing

- Unit tests for scoring logic
- Integration test with mock Typesense responses
- Test cards with no entities/tags (should only get semantic results)
