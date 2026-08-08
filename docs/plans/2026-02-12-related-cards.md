> **STATUS: HISTORICAL — pre-SQLite era.** This plan predates the PostgreSQL→SQLite cutover (2026-07-28, epic Zettelgarden-c7j) and the move to local on-disk file storage (epic Zettelgarden-yar). Zettelgarden now runs SQLite-only with local storage; this document is kept for design history.

# Related Cards Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "Related Cards" section to the ViewPage sidebar showing cards related by shared entities, shared tags, and semantic similarity.

**Architecture:** New backend endpoint `/api/cards/{id}/related` that queries entity/tag junctions and Typesense for semantic matches, merges results with scoring, and returns top cards. Frontend component displays results using existing CardItem.

**Tech Stack:** Go (backend), React/TypeScript (frontend), Typesense (vector search), PostgreSQL (junction tables)

---

## Task 1: Add RelatedCard Response Type to Backend Models

**Files:**
- Modify: `go-backend/models/card.go`

**Step 1: Add RelatedCard struct to models**

Add to `go-backend/models/card.go`:

```go
// RelatedCard represents a card with its relatedness score and reasons
type RelatedCard struct {
    Card   PartialCard
    Score   float64
    Reasons []string // e.g. entity/tag names that caused the match
}
```

**Step 2: Commit**

```bash
cd go-backend
git add models/card.go
git commit -m "feat(models): add RelatedCard type for related cards feature"
```

---

## Task 2: Add SQL Query for Shared Entity Cards

**Files:**
- Modify: `go-backend/services/cards.go`

**Step 1: Write test for finding cards by shared entities**

Create `go-backend/services/cards_test.go` (if not exists):

```go
package services

import (
    "testing"
    "go-backend/models"
)

func TestGetCardsBySharedEntities(t *testing.T) {
    // Setup: Create test cards with entities
    // This test assumes test database setup exists

    // Given: A card with entities "Python" and "Go"
    sourceCardID := 1

    // When: Query for cards sharing these entities
    cards, err := GetCardsBySharedEntities(db, 1, sourceCardID)

    // Then: Should return cards that have "Python" or "Go" entities
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    if len(cards) == 0 {
        t.Error("expected at least one card with shared entities")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./services -v -run TestGetCardsBySharedEntities
```
Expected: FAIL with "undefined: GetCardsBySharedEntities"

**Step 3: Implement GetCardsBySharedEntities**

Add to `go-backend/services/cards.go`:

```go
// GetCardsBySharedEntities finds cards that share entities with the source card
// Returns a map of cardID to score (higher = more shared entities)
func GetCardsBySharedEntities(db *sql.DB, userID int, sourceCardID int) (map[int]int, error) {
    // First, get all entity IDs for the source card
    entityQuery := `
        SELECT DISTINCT entity_id
        FROM entity_card_junction
        WHERE card_pk = $1 AND user_id = $2
    `

    rows, err := db.Query(entityQuery, sourceCardID, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get card entities: %w", err)
    }
    defer rows.Close()

    var entityIDs []int
    for rows.Next() {
        var entityID int
        if err := rows.Scan(&entityID); err != nil {
            return nil, fmt.Errorf("failed to scan entity ID: %w", err)
        }
        entityIDs = append(entityIDs, entityID)
    }

    if len(entityIDs) == 0 {
        return make(map[int]int), nil
    }

    // Now find all cards that share these entities
    query := `
        SELECT ecj.card_pk, COUNT(DISTINCT ecj.entity_id) as shared_count
        FROM entity_card_junction ecj
        JOIN cards c ON ecj.card_pk = c.id
        WHERE ecj.entity_id = ANY($1)
          AND ecj.user_id = $2
          AND ecj.card_pk != $3
          AND c.is_deleted = FALSE
        GROUP BY ecj.card_pk
    `

    rows, err = db.Query(query, pg.Array(entityIDs), userID, sourceCardID)
    if err != nil {
        return nil, fmt.Errorf("failed to find shared entity cards: %w", err)
    }
    defer rows.Close()

    scores := make(map[int]int)
    for rows.Next() {
        var cardID, sharedCount int
        if err := rows.Scan(&cardID, &sharedCount); err != nil {
            return nil, fmt.Errorf("failed to scan shared entity card: %w", err)
        }
        // Each shared entity adds 3 points
        scores[cardID] = sharedCount * 3
    }

    return scores, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./services -v -run TestGetCardsBySharedEntities
```
Expected: PASS (may need test database setup)

**Step 5: Commit**

```bash
cd go-backend
git add services/cards.go services/cards_test.go
git commit -m "feat(services): add GetCardsBySharedEntities for related cards"
```

---

## Task 3: Add SQL Query for Shared Tag Cards

**Files:**
- Modify: `go-backend/services/cards.go`

**Step 1: Write test for finding cards by shared tags**

Add to `go-backend/services/cards_test.go`:

```go
func TestGetCardsBySharedTags(t *testing.T) {
    // Given: A card with tags "programming" and "golang"
    sourceCardID := 1

    // When: Query for cards sharing these tags
    cards, err := GetCardsBySharedTags(db, 1, sourceCardID)

    // Then: Should return cards that have "programming" or "golang" tags
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    if len(cards) == 0 {
        t.Error("expected at least one card with shared tags")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./services -v -run TestGetCardsBySharedTags
```
Expected: FAIL with "undefined: GetCardsBySharedTags"

**Step 3: Implement GetCardsBySharedTags**

Add to `go-backend/services/cards.go`:

```go
// GetCardsBySharedTags finds cards that share tags with the source card
// Returns a map of cardID to score (higher = more shared tags)
func GetCardsBySharedTags(db *sql.DB, userID int, sourceCardID int) (map[int]int, error) {
    // First, get all tag IDs for the source card
    tagQuery := `
        SELECT DISTINCT tag_id
        FROM card_tags
        WHERE card_pk = $1 AND user_id = $2
    `

    rows, err := db.Query(tagQuery, sourceCardID, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get card tags: %w", err)
    }
    defer rows.Close()

    var tagIDs []int
    for rows.Next() {
        var tagID int
        if err := rows.Scan(&tagID); err != nil {
            return nil, fmt.Errorf("failed to scan tag ID: %w", err)
        }
        tagIDs = append(tagIDs, tagID)
    }

    if len(tagIDs) == 0 {
        return make(map[int]int), nil
    }

    // Now find all cards that share these tags
    query := `
        SELECT ct.card_pk, COUNT(DISTINCT ct.tag_id) as shared_count
        FROM card_tags ct
        JOIN cards c ON ct.card_pk = c.id
        JOIN tags t ON ct.tag_id = t.id
        WHERE ct.tag_id = ANY($1)
          AND ct.user_id = $2
          AND ct.card_pk != $3
          AND c.is_deleted = FALSE
          AND t.is_deleted = FALSE
        GROUP BY ct.card_pk
    `

    rows, err = db.Query(query, pg.Array(tagIDs), userID, sourceCardID)
    if err != nil {
        return nil, fmt.Errorf("failed to find shared tag cards: %w", err)
    }
    defer rows.Close()

    scores := make(map[int]int)
    for rows.Next() {
        var cardID, sharedCount int
        if err := rows.Scan(&cardID, &sharedCount); err != nil {
            return nil, fmt.Errorf("failed to scan shared tag card: %w", err)
        }
        // Each shared tag adds 1 point
        scores[cardID] = sharedCount * 1
    }

    return scores, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./services -v -run TestGetCardsBySharedTags
```
Expected: PASS

**Step 5: Commit**

```bash
cd go-backend
git add services/cards.go services/cards_test.go
git commit -m "feat(services): add GetCardsBySharedTags for related cards"
```

---

## Task 4: Add Semantic Search via Typesense

**Files:**
- Modify: `go-backend/server/similarity.go`

**Step 1: Write test for semantic card search**

Create `go-backend/server/similarity_test.go`:

```go
package server

import (
    "testing"
    "go-backend/models"
)

func TestFindSimilarCards(t *testing.T) {
    // Given: A source card
    sourceCard := models.Card{
        ID:    1,
        Title:  "Test Card",
        Body:   "This is about programming",
        UserID: 1,
    }

    // When: Search for similar cards
    results, err := testServer.FindSimilarCards(context.Background(), sourceCard, 5)

    // Then: Should return cards with similarity scores
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    if len(results) == 0 {
        t.Error("expected at least one similar card")
    }

    // Verify scores are between 0 and 1
    for _, r := range results {
        if r.Score < 0 || r.Score > 1 {
            t.Errorf("expected score between 0-1, got %f", r.Score)
        }
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./server -v -run TestFindSimilarCards
```
Expected: FAIL with "undefined: FindSimilarCards"

**Step 3: Implement FindSimilarCards**

Add to `go-backend/server/similarity.go`:

```go
// SimilarCard represents a card ID with its similarity score
type SimilarCard struct {
    ID    int
    Score float64
}

// FindSimilarCards finds cards semantically similar to the given card using Typesense
func (s *Server) FindSimilarCards(ctx context.Context, card models.Card, limit int) ([]SimilarCard, error) {
    if s.TypesenseClient == nil {
        log.Printf("Typesense client not available")
        return nil, nil // Return empty, will use entity/tag matches only
    }

    collectionName := os.Getenv("TYPESENSE_COLLECTION")
    if collectionName == "" {
        log.Printf("TYPESENSE_COLLECTION env var not set")
        return nil, nil
    }

    // Build search query from title and body
    searchQuery := fmt.Sprintf("%s %s", card.Title, card.Body)

    // Filter for cards only, excluding current card
    filter := fmt.Sprintf("user_id:=%d && type:=card && card_pk:!=%d", card.UserID, card.ID)

    perPage := limit
    searchParams := &api.SearchCollectionParams{
        Q:        searchQuery,
        QueryBy:  "title,preview,embedding", // Include embedding for vector search
        FilterBy: &filter,
        PerPage:  &perPage,
    }

    searchResult, err := s.TypesenseClient.Collection(collectionName).Documents().Search(ctx, searchParams)
    if err != nil {
        log.Printf("Typesense semantic search failed: %v", err)
        return nil, nil // Return empty to trigger fallback
    }

    var similarCards []SimilarCard
    if searchResult.Hits != nil {
        for _, hit := range *searchResult.Hits {
            if hit.Document != nil {
                doc := *hit.Document
                if pk, ok := doc["card_pk"].(float64); ok {
                    score := 0.0
                    // Extract vector distance if available
                    if hit.VectorDistance != nil {
                        distance := float64(*hit.VectorDistance)
                        // Convert cosine distance to similarity (0-1 range)
                        score = 1.0 - (distance / 2.0)
                        // Clamp to 0-1
                        if score < 0 {
                            score = 0
                        } else if score > 1 {
                            score = 1
                        }
                    }
                    similarCards = append(similarCards, SimilarCard{
                        ID:    int(pk),
                        Score: score,
                    })
                }
            }
        }
    }

    return similarCards, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./server -v -run TestFindSimilarCards
```
Expected: PASS (may need mock Typesense)

**Step 5: Commit**

```bash
cd go-backend
git add server/similarity.go server/similarity_test.go
git commit -m "feat(server): add FindSimilarCards for semantic related cards"
```

---

## Task 5: Create Backend Handler for Related Cards

**Files:**
- Modify: `go-backend/handlers/cards.go`

**Step 1: Write handler integration test**

Add to `go-backend/handlers/cards_test.go`:

```go
func TestGetRelatedCardsHandler(t *testing.T) {
    // Setup test DB with cards, entities, tags
    // Create test server

    req := httptest.NewRequest("GET", "/api/cards/1/related", nil)
    rr := httptest.NewRecorder()

    handler := testHandler.GetRelatedCards
    handler.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", rr.Code)
    }

    var response []models.RelatedCard
    json.Unmarshal(rr.Body.Bytes(), &response)

    if len(response) == 0 {
        t.Error("expected at least one related card")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./handlers -v -run TestGetRelatedCardsHandler
```
Expected: FAIL with handler not found

**Step 3: Implement GetRelatedCards handler**

Add to `go-backend/handlers/cards.go`:

```go
// GetRelatedCards returns cards related to the given card ID
// Relatedness is determined by shared entities, tags, and semantic similarity
func (h *Handler) GetRelatedCards(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    cardIDStr := mux.Vars(r)["id"]

    cardID, err := strconv.Atoi(cardIDStr)
    if err != nil {
        http.Error(w, "invalid card ID", http.StatusBadRequest)
        return
    }

    // Get the source card
    var card models.Card
    err = h.DB.QueryRow(`
        SELECT id, card_id, user_id, title, body,
               parent_id, created_at, updated_at
        FROM cards
        WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
    `, cardID, userID).Scan(
        &card.ID, &card.CardID, &card.UserID, &card.Title,
        &card.Body, &card.ParentID, &card.CreatedAt, &card.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "card not found", http.StatusNotFound)
            return
        }
        log.Printf("error getting card: %v", err)
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    // Collect scores from all signals
    entityScores, err := services.GetCardsBySharedEntities(h.DB, userID, cardID)
    if err != nil {
        log.Printf("error getting shared entity cards: %v", err)
        // Continue without entity scores
    }

    tagScores, err := services.GetCardsBySharedTags(h.DB, userID, cardID)
    if err != nil {
        log.Printf("error getting shared tag cards: %v", err)
        // Continue without tag scores
    }

    semanticResults, err := h.Server.FindSimilarCards(r.Context(), card, 20)
    if err != nil {
        log.Printf("error getting semantic cards: %v", err)
        // Continue without semantic scores
    }

    // Merge all scores
    combinedScores := make(map[int]float64)
    cardReasons := make(map[int][]string)

    for cardID, score := range entityScores {
        combinedScores[cardID] += float64(score)
        cardReasons[cardID] = append(cardReasons[cardID], "entities")
    }

    for cardID, score := range tagScores {
        combinedScores[cardID] += float64(score)
        cardReasons[cardID] = append(cardReasons[cardID], "tags")
    }

    for _, result := range semanticResults {
        combinedScores[result.ID] += result.Score
        cardReasons[result.ID] = append(cardReasons[result.ID], "similarity")
    }

    // Exclude: current card, parent, direct children, siblings (already visible in UI)
    excludeIDs := make(map[int]bool)
    excludeIDs[cardID] = true
    if card.ParentID != 0 {
        excludeIDs[card.ParentID] = true

        // Get siblings
        siblingRows, _ := h.DB.Query(`
            SELECT id FROM cards WHERE parent_id = $1 AND user_id = $2 AND id != $3
        `, card.ParentID, userID, cardID)
        if siblingRows != nil {
            defer siblingRows.Close()
            for siblingRows.Next() {
                var siblingID int
                if siblingRows.Scan(&siblingID) == nil {
                    excludeIDs[siblingID] = true
                }
            }
        }
    }

    // Get children
    childRows, _ := h.DB.Query(`
        SELECT id FROM cards WHERE parent_id = $1 AND user_id = $2
    `, cardID, userID)
    if childRows != nil {
        defer childRows.Close()
        for childRows.Next() {
            var childID int
            if childRows.Scan(&childID) == nil {
                excludeIDs[childID] = true
            }
        }
    }

    // Build final results, excluding filtered cards
    var results []models.RelatedCard
    for cardID, score := range combinedScores {
        if excludeIDs[cardID] {
            continue
        }

        // Get the card details
        var partialCard models.PartialCard
        err := h.DB.QueryRow(`
            SELECT id, card_id, user_id, title,
                   parent_id, created_at, updated_at
            FROM cards
            WHERE id = $1
        `, cardID).Scan(
            &partialCard.ID, &partialCard.CardID, &partialCard.UserID,
            &partialCard.Title, &partialCard.ParentID,
            &partialCard.CreatedAt, &partialCard.UpdatedAt,
        )
        if err != nil {
            continue
        }

        // Get tags for the card
        partialCard.Tags, _ = services.QueryTagsForCard(h.DB, userID, cardID)

        results = append(results, models.RelatedCard{
            Card:   partialCard,
            Score:   score,
            Reasons: cardReasons[cardID],
        })
    }

    // Sort by score descending
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    // Limit to top 10
    maxResults := 10
    if len(results) > maxResults {
        results = results[:maxResults]
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./handlers -v -run TestGetRelatedCardsHandler
```
Expected: PASS

**Step 5: Commit**

```bash
cd go-backend
git add handlers/cards.go handlers/cards_test.go
git commit -m "feat(handlers): add GetRelatedCards handler"
```

---

## Task 6: Add Backend Route

**Files:**
- Modify: `go-backend/routes/cards.go`

**Step 1: Add route**

Add the new route to `go-backend/routes/cards.go`:

```go
router.HandleFunc("/cards/{id}/related", handlers.GetAuthHandler(h.GetRelatedCards)).Methods("GET")
```

Place it after the existing card routes.

**Step 2: Commit**

```bash
cd go-backend
git add routes/cards.go
git commit -m "feat(routes): add /cards/{id}/related endpoint"
```

---

## Task 7: Add TypeScript Types for Related Cards

**Files:**
- Modify: `zettelkasten-front/src/models/Card.ts`

**Step 1: Add RelatedCard interface**

Add to `zettelkasten-front/src/models/Card.ts`:

```typescript
export interface RelatedCard {
  card: PartialCard;
  score: number;
  reasons: string[];
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/models/Card.ts
git commit -m "feat(types): add RelatedCard interface"
```

---

## Task 8: Add API Function for Related Cards

**Files:**
- Modify: `zettelkasten-front/src/api/cards.ts`

**Step 1: Add getRelatedCards function**

Add to `zettelkasten-front/src/api/cards.ts`:

```typescript
import { RelatedCard } from "../models/Card";

export async function getRelatedCards(cardId: string): Promise<RelatedCard[]> {
  const response = await authenticatedFetch(`/api/cards/${cardId}/related`);
  if (!response.ok) {
    throw new Error(`Failed to fetch related cards: ${response.statusText}`);
  }
  return response.json();
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/api/cards.ts
git commit -m "feat(api): add getRelatedCards function"
```

---

## Task 9: Create RelatedCards Component

**Files:**
- Create: `zettelkasten-front/src/components/cards/RelatedCards.tsx`

**Step 1: Write component**

Create `zettelkasten-front/src/components/cards/RelatedCards.tsx`:

```tsx
import React from "react";
import { RelatedCard } from "../../models/Card";
import { HeaderSubSection } from "../Header";
import { CardItem } from "./CardItem";

interface RelatedCardsProps {
  relatedCards: RelatedCard[];
  onCardClick: (cardId: number) => void;
}

export function RelatedCards({ relatedCards, onCardClick }: RelatedCardsProps) {
  if (relatedCards.length === 0) {
    return null;
  }

  return (
    <div>
      <HeaderSubSection text="Related Cards" />
      <ul className="mt-2 space-y-1">
        {relatedCards.map((rc) => (
          <li
            key={rc.card.id}
            onClick={() => onCardClick(rc.card.id)}
            className="cursor-pointer"
          >
            <CardItem card={rc.card} />
            {rc.score > 0 && (
              <span className="text-xs text-gray-400 ml-2">
                {rc.score.toFixed(1)}
              </span>
            )}
          </li>
        ))}
      </ul>
      <hr className="my-4" />
    </div>
  );
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/components/cards/RelatedCards.tsx
git commit -m "feat(components): add RelatedCards component"
```

---

## Task 10: Update ViewPageContainer to Fetch Related Cards

**Files:**
- Modify: `zettelkasten-front/src/pages/cards/ViewPageContainer.tsx`

**Step 1: Add relatedCards to state and data**

Find the data state in ViewPageContainer and add `relatedCards`:

```typescript
// In the data object, add:
relatedCards: RelatedCard[] | null;
```

**Step 2: Fetch related cards when card loads**

Add to the effect or data fetching logic:

```typescript
// After fetching the card, fetch related cards
if (viewingCard && !relatedCards) {
  getRelatedCards(viewingCard.id.toString())
    .then(setRelatedCards)
    .catch(err => console.error("Failed to fetch related cards:", err));
}
```

**Step 3: Commit**

```bash
cd zettelkasten-front
git add src/pages/cards/ViewPageContainer.tsx
git commit -m "feat(container): fetch related cards in ViewPageContainer"
```

---

## Task 11: Update ViewPageSidePanels to Show Related Cards

**Files:**
- Modify: `zettelkasten-front/src/components/cards/ViewPageSidePanels.tsx`

**Step 1: Add RelatedCards component import**

Add import:
```tsx
import { RelatedCards } from "./RelatedCards";
```

**Step 2: Add relatedCards prop and render RelatedCards**

Add to props interface and component:

```tsx
interface ViewPageSidePanelsProps {
  // ... existing props
  relatedCards?: RelatedCard[];
  onRelatedCardClick?: (cardId: number) => void;
}

export function ViewPageSidePanels({
  // ... existing props
  relatedCards,
  onRelatedCardClick
}: ViewPageSidePanelsProps) {
  // ... existing code

  return (
    <div className="md:w-1/3 bg-white rounded-lg p-4 shadow-sm space-y-4">
      {/* ... existing sections */}

      {/* Related Cards Section */}
      {relatedCards && onRelatedCardClick && (
        <RelatedCards
          relatedCards={relatedCards}
          onCardClick={onRelatedCardClick}
        />
      )}

      {/* ... rest of sections */}
    </div>
  );
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/components/cards/ViewPageSidePanels.tsx
git commit -m "feat(panels): add RelatedCards to ViewPageSidePanels"
```

---

## Task 12: Wire Up Related Cards in ViewPage

**Files:**
- Modify: `zettelkasten-front/src/pages/cards/ViewPage.tsx`

**Step 1: Pass relatedCards props to ViewPageSidePanels**

Update the ViewPageSidePanels usage:

```tsx
<ViewPageSidePanels
  // ... existing props
  relatedCards={data.relatedCards}
  onRelatedCardClick={(cardId) => navigate(`/app/card/${cardId}`)}
/>
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/pages/cards/ViewPage.tsx
git commit -m "feat(page): wire up related cards in ViewPage"
```

---

## Task 13: Test End-to-End

**Files:**
- No file changes

**Step 1: Start both services**

```bash
# Terminal 1 - Backend
cd go-backend
source .env-bash
go run main.go

# Terminal 2 - Frontend
cd zettelkasten-front
npm start
```

**Step 2: Manual testing checklist**

1. Create two cards with shared entity "Python"
2. View one card - verify the other appears in Related Cards
3. Create two cards with shared tag "programming"
4. Verify related cards appear
5. Create card with semantically similar content
6. Verify semantic matches appear
7. Verify current card's parent/children/siblings are NOT shown
8. Verify clicking related card navigates correctly

**Step 3: Commit**

```bash
git add docs/plans/2026-02-12-related-cards.md
git commit -m "docs: complete related cards implementation plan"
```

---

## Summary

This plan implements the Related Cards feature in 13 bite-sized tasks:

1. Backend models and queries for entity/tag sharing
2. Typesense semantic search integration
3. Handler combining all signals with scoring
4. Frontend types, API, and components
5. Integration with existing ViewPage

**Total estimated time**: 2-3 hours for full implementation

**Testing approach**: TDD throughout with unit tests for backend logic
