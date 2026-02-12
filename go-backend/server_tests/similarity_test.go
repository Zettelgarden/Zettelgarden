package server_tests

import (
	"context"
	"go-backend/models"
	"go-backend/tests"
	"testing"
)

func TestFindSimilarCards(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Given: A source card
	sourceCard := models.Card{
		ID:     1,
		Title:  "Test Card",
		Body:   "This is about programming",
		UserID: 1,
	}

	// When: Search for similar cards
	results, err := s.FindSimilarCards(context.Background(), sourceCard, 5)

	// Then: Should not error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should return a slice (may be empty if Typesense not configured)
	if results == nil {
		t.Error("expected results slice, got nil")
	}

	// Verify scores are between 0 and 1 if results exist
	for _, r := range results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("expected score between 0-1, got %f", r.Score)
		}
	}
}

func TestFindSimilarCards_NilTypesenseClient(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Given: Server with nil TypesenseClient
	s.TypesenseClient = nil

	sourceCard := models.Card{
		ID:     1,
		Title:  "Test Card",
		Body:   "This is about programming",
		UserID: 1,
	}

	// When: Search for similar cards
	results, err := s.FindSimilarCards(context.Background(), sourceCard, 5)

	// Then: Should return empty slice, nil (graceful degradation)
	if err != nil {
		t.Fatalf("expected no error with nil TypesenseClient, got %v", err)
	}

	if results == nil {
		t.Error("expected empty slice results when TypesenseClient is nil, got nil")
	}

	if len(results) != 0 {
		t.Errorf("expected empty results when TypesenseClient is nil, got %d results", len(results))
	}
}

func TestFindSimilarCards_ExcludesCurrentCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Given: A source card with ID 1
	sourceCard := models.Card{
		ID:     1,
		Title:  "Test Card",
		Body:   "This is about programming",
		UserID: 1,
	}

	// When: Search for similar cards
	results, err := s.FindSimilarCards(context.Background(), sourceCard, 10)

	// Then: Should not error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify current card is not in results
	if results != nil {
		for _, r := range results {
			if r.ID == sourceCard.ID {
				t.Errorf("expected results to exclude current card (ID %d), but found it in results", sourceCard.ID)
			}
		}
	}
}
