package services

import (
	"go-backend/models"
	"go-backend/tests"
	"testing"
)

func TestGetCardFacts(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	card, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Test Card for Facts",
		Body:   "Test Body",
		CardID: "fact_test_card",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var fact1ID, fact2ID int
	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card.ID, userID, "Test fact 1").Scan(&fact1ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact 1: %v", err)
	}

	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card.ID, userID, "Test fact 2").Scan(&fact2ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact 2: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ($4, $5, $6, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, fact1ID, card.ID, userID, fact2ID, card.ID, userID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact_card_junction: %v", err)
	}

	tx.Commit()

	facts, err := GetCardFacts(s.DB, userID, card.ID)
	if err != nil {
		t.Fatalf("GetCardFacts failed: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("Expected 2 facts, got %v", len(facts))
	}

	factsFound := make(map[int]bool)
	for _, fact := range facts {
		factsFound[fact.ID] = true
		if fact.UserID != userID {
			t.Errorf("Expected user_id %v, got %v", userID, fact.UserID)
		}
		if fact.CardPK != card.ID {
			t.Errorf("Expected card_pk %v, got %v", card.ID, fact.CardPK)
		}
	}

	if !factsFound[fact1ID] {
		t.Error("Fact 1 not found in results")
	}
	if !factsFound[fact2ID] {
		t.Error("Fact 2 not found in results")
	}
}

func TestGetEntityFacts(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	card, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Test Card for Entity Facts",
		Body:   "Test Body",
		CardID: "entity_fact_test_card",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var entityID int
	err = tx.QueryRow(`
		INSERT INTO entities (user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, userID, "Test Entity Facts", "A test entity for facts", "person").Scan(&entityID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert entity: %v", err)
	}

	var fact1ID, fact2ID int
	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card.ID, userID, "Entity fact 1").Scan(&fact1ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact 1: %v", err)
	}

	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card.ID, userID, "Entity fact 2").Scan(&fact2ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact 2: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ($4, $5, $6, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, fact1ID, card.ID, userID, fact2ID, card.ID, userID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact_card_junction: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO entity_fact_junction (user_id, entity_id, fact_id, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ($4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, userID, entityID, fact1ID, userID, entityID, fact2ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert entity_fact_junction: %v", err)
	}

	tx.Commit()

	facts, err := GetEntityFacts(s.DB, userID, entityID)
	if err != nil {
		t.Fatalf("GetEntityFacts failed: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("Expected 2 facts, got %v", len(facts))
	}

	factsFound := make(map[int]bool)
	for _, fact := range facts {
		factsFound[fact.ID] = true
	}

	if !factsFound[fact1ID] {
		t.Error("Fact 1 not found in results")
	}
	if !factsFound[fact2ID] {
		t.Error("Fact 2 not found in results")
	}
}

func TestGetFactCards(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	card1, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Test Card 1 for Fact",
		Body:   "Test Body 1",
		CardID: "fact_card_test_1",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card 1: %v", err)
	}

	card2, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Test Card 2 for Fact",
		Body:   "Test Body 2",
		CardID: "fact_card_test_2",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card 2: %v", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var factID int
	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card1.ID, userID, "Shared fact").Scan(&factID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ($4, $5, $6, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, factID, card1.ID, userID, factID, card2.ID, userID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact_card_junction: %v", err)
	}

	tx.Commit()

	cards, err := GetFactCards(s.DB, userID, factID)
	if err != nil {
		t.Fatalf("GetFactCards failed: %v", err)
	}

	if len(cards) != 2 {
		t.Errorf("Expected 2 cards, got %v", len(cards))
	}

	cardsFound := make(map[int]bool)
	for _, card := range cards {
		cardsFound[card.ID] = true
		if card.UserID != userID {
			t.Errorf("Expected user_id %v, got %v", userID, card.UserID)
		}
	}

	if !cardsFound[card1.ID] {
		t.Error("Card 1 not found in results")
	}
	if !cardsFound[card2.ID] {
		t.Error("Card 2 not found in results")
	}
}

func TestGetCardFactsEmpty(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	card, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Card with no facts",
		Body:   "Test Body",
		CardID: "no_facts_card",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	facts, err := GetCardFacts(s.DB, userID, card.ID)
	if err != nil {
		t.Fatalf("GetCardFacts failed: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("Expected 0 facts, got %v", len(facts))
	}
}

func TestGetEntityFactsEmpty(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var entityID int
	err = tx.QueryRow(`
		INSERT INTO entities (user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, userID, "Entity with no facts test", "A test entity with no facts", "person").Scan(&entityID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert entity: %v", err)
	}
	tx.Commit()

	facts, err := GetEntityFacts(s.DB, userID, entityID)
	if err != nil {
		t.Fatalf("GetEntityFacts failed: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("Expected 0 facts, got %v", len(facts))
	}
}

func TestGetFactCardsEmpty(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	card, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Card for orphaned fact",
		Body:   "Test Body",
		CardID: "orphaned_fact_card",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var factID int
	err = tx.QueryRow(`
		INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, card.ID, userID, "Orphaned fact").Scan(&factID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert fact: %v", err)
	}
	tx.Commit()

	cards, err := GetFactCards(s.DB, userID, factID)
	if err != nil {
		t.Fatalf("GetFactCards failed: %v", err)
	}

	if len(cards) != 0 {
		t.Errorf("Expected 0 cards for orphaned fact, got %v", len(cards))
	}
}