package services

import (
	"go-backend/tests"
	"testing"
)

// TestGetCardsBySharedEntities tests finding cards that share entities with the source card
func TestGetCardsBySharedEntities(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Given: A card with entities "Test Entity 1" and "Test Entity 2"
	// From test data: Card 1 has entities 1 and 2, Card 2 has entities 1 and 2
	sourceCardID := 1

	// When: Query for cards sharing these entities
	scores, err := GetCardsBySharedEntities(s.DB, userID, sourceCardID)

	// Then: Should return cards that share entities with the source card
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(scores) == 0 {
		t.Error("expected at least one card with shared entities")
	}

	// From test data: Card 2 shares both entities with Card 1
	// So card 2 should have a score of 6 (2 entities * 3 points each)
	expectedScore := 6
	if score, ok := scores[2]; !ok {
		t.Error("expected card 2 to be in results (shares both entities)")
	} else if score != expectedScore {
		t.Errorf("expected card 2 to have score %d, got %d", expectedScore, score)
	}

	// Source card should not be in its own results
	if _, ok := scores[sourceCardID]; ok {
		t.Error("source card should not be in its own results")
	}
}

// TestGetCardsBySharedEntities_NoEntities tests with a card that has no entities
func TestGetCardsBySharedEntities_NoEntities(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card with no entities
	var newCardID int
	err := s.DB.QueryRow(`
		INSERT INTO cards (user_id, title, body, card_id, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, userID, "No Entities Card", "This card has no entities", "NOENT001", 1).Scan(&newCardID)
	if err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}

	// Query for cards sharing entities
	scores, err := GetCardsBySharedEntities(s.DB, userID, newCardID)

	// Should return empty map (no error)
	if err != nil {
		t.Fatalf("expected no error for card with no entities, got %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("expected empty map for card with no entities, got %d entries", len(scores))
	}
}

// TestGetCardsBySharedEntities_NoMatches tests when no other cards share entities
func TestGetCardsBySharedEntities_NoMatches(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a new card with a unique entity
	var newCardID int
	err := s.DB.QueryRow(`
		INSERT INTO cards (user_id, title, body, card_id, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, userID, "Unique Entity Card", "This card has unique entities", "UNIQUE001", 1).Scan(&newCardID)
	if err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}

	// Create a unique entity for this card
	var uniqueEntityID int
	err = s.DB.QueryRow(`
		INSERT INTO entities (user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, userID, "Unique Entity", "A unique entity only on this card", "concept").Scan(&uniqueEntityID)
	if err != nil {
		t.Fatalf("failed to create test entity: %v", err)
	}

	// Link the unique entity to the new card
	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, userID, uniqueEntityID, newCardID)
	if err != nil {
		t.Fatalf("failed to link entity to card: %v", err)
	}

	// Query for cards sharing entities - should find no matches
	scores, err := GetCardsBySharedEntities(s.DB, userID, newCardID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("expected no matches for card with unique entity, got %d matches", len(scores))
	}
}

// TestGetCardsBySharedEntities_WrongUser tests user isolation
func TestGetCardsBySharedEntities_WrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// From test data, entity 3 belongs to user 2
	// Query as user 1 for cards sharing entities with card 1
	scores, err := GetCardsBySharedEntities(s.DB, 1, 1)

	// Should succeed but only include user 1's cards
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Card 13 belongs to user 2, so it should not be in results
	// even if it shares entities (it shouldn't based on user_id in junction)
	if _, ok := scores[13]; ok {
		t.Error("user 2's card should not be in user 1's results")
	}
}

// TestGetCardsBySharedEntities_SingleSharedEntity tests scoring with one shared entity
func TestGetCardsBySharedEntities_SingleSharedEntity(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create test cards with controlled entity sharing
	// Card A: Has Entity 1
	var cardAID int
	err := s.DB.QueryRow(`
		INSERT INTO cards (user_id, title, body, card_id, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, userID, "Card A", "Content A", "TESTA", 1).Scan(&cardAID)
	if err != nil {
		t.Fatalf("failed to create card A: %v", err)
	}

	// Card B: Has Entity 1 and Entity 2
	var cardBID int
	err = s.DB.QueryRow(`
		INSERT INTO cards (user_id, title, body, card_id, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, userID, "Card B", "Content B", "TESTB", 1).Scan(&cardBID)
	if err != nil {
		t.Fatalf("failed to create card B: %v", err)
	}

	// Create Entity 1 and Entity 2 (using high IDs to avoid conflicts)
	var entity1ID, entity2ID int
	err = s.DB.QueryRow(`
		INSERT INTO entities (id, user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, 9999, userID, "Test Entity A", "Entity A description", "concept").Scan(&entity1ID)
	if err != nil {
		t.Fatalf("failed to create entity 1: %v", err)
	}

	err = s.DB.QueryRow(`
		INSERT INTO entities (id, user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, 9998, userID, "Test Entity B", "Entity B description", "concept").Scan(&entity2ID)
	if err != nil {
		t.Fatalf("failed to create entity 2: %v", err)
	}

	// Link Entity 1 to both cards
	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, userID, entity1ID, cardAID)
	if err != nil {
		t.Fatalf("failed to link entity 1 to card A: %v", err)
	}

	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, userID, entity1ID, cardBID)
	if err != nil {
		t.Fatalf("failed to link entity 1 to card B: %v", err)
	}

	// Link Entity 2 only to card B
	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, userID, entity2ID, cardBID)
	if err != nil {
		t.Fatalf("failed to link entity 2 to card B: %v", err)
	}

	// Query for cards sharing entities with card A
	scores, err := GetCardsBySharedEntities(s.DB, userID, cardAID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Card B should be in results with score 3 (1 shared entity)
	expectedScore := 3 // 1 shared entity * 3 points
	if score, ok := scores[cardBID]; !ok {
		t.Error("expected card B to be in results (shares 1 entity)")
	} else if score != expectedScore {
		t.Errorf("expected card B to have score %d, got %d", expectedScore, score)
	}

	// If we query from card B, card A should have score 3 (shares only entity 1)
	scoresFromB, err := GetCardsBySharedEntities(s.DB, userID, cardBID)
	if err != nil {
		t.Fatalf("expected no error querying from card B, got %v", err)
	}

	if score, ok := scoresFromB[cardAID]; !ok {
		t.Error("expected card A to be in results when querying from card B")
	} else if score != expectedScore {
		t.Errorf("expected card A to have score %d when querying from B, got %d", expectedScore, score)
	}
}

// TestGetCardsBySharedEntities_DeletedCard tests that deleted cards are excluded
func TestGetCardsBySharedEntities_DeletedCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a new card and link it to share entities with card 1
	var newCardID int
	err := s.DB.QueryRow(`
		INSERT INTO cards (user_id, title, body, card_id, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, userID, "Shared Entity Card", "This card shares entities", "SHARED001", 1).Scan(&newCardID)
	if err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}

	// Link to existing entity 1
	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, userID, 1, newCardID)
	if err != nil {
		t.Fatalf("failed to link entity to card: %v", err)
	}

	// Verify the card appears in results
	scores, err := GetCardsBySharedEntities(s.DB, userID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := scores[newCardID]; !ok {
		t.Error("expected new card to be in results before deletion")
	}

	// Soft delete the card
	_, err = s.DB.Exec(`
		UPDATE cards SET is_deleted = TRUE WHERE id = $1
	`, newCardID)
	if err != nil {
		t.Fatalf("failed to delete card: %v", err)
	}

	// Query again - deleted card should not appear
	scoresAfterDelete, err := GetCardsBySharedEntities(s.DB, userID, 1)
	if err != nil {
		t.Fatalf("expected no error after deletion, got %v", err)
	}

	if _, ok := scoresAfterDelete[newCardID]; ok {
		t.Error("deleted card should not be in results")
	}
}
