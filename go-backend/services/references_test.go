package services

import (
	"go-backend/models"
	"testing"
)

// TestGetUniqueCards tests the getUniqueCards helper function
func TestGetUniqueCards(t *testing.T) {
	tests := []struct {
		name     string
		input    []models.PartialCard
		expected int
	}{
		{
			name: "no duplicates",
			input: []models.PartialCard{
				{ID: 1, CardID: "card1"},
				{ID: 2, CardID: "card2"},
				{ID: 3, CardID: "card3"},
			},
			expected: 3,
		},
		{
			name: "with duplicates",
			input: []models.PartialCard{
				{ID: 1, CardID: "card1"},
				{ID: 2, CardID: "card2"},
				{ID: 1, CardID: "card1"},
				{ID: 3, CardID: "card3"},
				{ID: 2, CardID: "card2"},
			},
			expected: 3,
		},
		{
			name:     "empty slice",
			input:    []models.PartialCard{},
			expected: 0,
		},
		{
			name: "all same card",
			input: []models.PartialCard{
				{ID: 1, CardID: "card1"},
				{ID: 1, CardID: "card1"},
				{ID: 1, CardID: "card1"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueCards(tt.input)
			if len(result) != tt.expected {
				t.Errorf("getUniqueCards() returned %d cards, expected %d", len(result), tt.expected)
			}

			// Verify uniqueness by CardID
			seen := make(map[string]bool)
			for _, card := range result {
				if seen[card.CardID] {
					t.Errorf("getUniqueCards() returned duplicate card_id: %s", card.CardID)
				}
				seen[card.CardID] = true
			}
		})
	}
}

// TestCategorizeReferences tests the CategorizeReferences function
func TestCategorizeReferences(t *testing.T) {
	card1 := models.PartialCard{ID: 1, CardID: "card1", Title: "Card 1"}
	card2 := models.PartialCard{ID: 2, CardID: "card2", Title: "Card 2"}
	card3 := models.PartialCard{ID: 3, CardID: "card3", Title: "Card 3"}
	card4 := models.PartialCard{ID: 4, CardID: "card4", Title: "Card 4"}

	tests := []struct {
		name        string
		directLinks []models.PartialCard
		backlinks   []models.PartialCard
		wantBidir   int
		wantOut     int
		wantIn      int
	}{
		{
			name:        "empty",
			directLinks: []models.PartialCard{},
			backlinks:   []models.PartialCard{},
			wantBidir:   0,
			wantOut:     0,
			wantIn:      0,
		},
		{
			name:        "only outgoing",
			directLinks: []models.PartialCard{card1, card2},
			backlinks:   []models.PartialCard{},
			wantBidir:   0,
			wantOut:     2,
			wantIn:      0,
		},
		{
			name:        "only incoming",
			directLinks: []models.PartialCard{},
			backlinks:   []models.PartialCard{card3, card4},
			wantBidir:   0,
			wantOut:     0,
			wantIn:      2,
		},
		{
			name:        "bidirectional only",
			directLinks: []models.PartialCard{card1, card2},
			backlinks:   []models.PartialCard{card1, card2},
			wantBidir:   2,
			wantOut:     0,
			wantIn:      0,
		},
		{
			name:        "mixed",
			directLinks: []models.PartialCard{card1, card2},
			backlinks:   []models.PartialCard{card1, card3},
			wantBidir:   1, // card1 is in both
			wantOut:     1, // card2 is only in direct
			wantIn:      1, // card3 is only in backlinks
		},
		{
			name:        "complex mixed",
			directLinks: []models.PartialCard{card1, card2},
			backlinks:   []models.PartialCard{card1, card2, card3, card4},
			wantBidir:   2, // card1 and card2 are in both
			wantOut:     0,
			wantIn:      2, // card3 and card4 are only in backlinks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeReferences(tt.directLinks, tt.backlinks)

			if len(result.Bidirectional) != tt.wantBidir {
				t.Errorf("CategorizeReferences() Bidirectional count = %d, want %d", len(result.Bidirectional), tt.wantBidir)
			}
			if len(result.Outgoing) != tt.wantOut {
				t.Errorf("CategorizeReferences() Outgoing count = %d, want %d", len(result.Outgoing), tt.wantOut)
			}
			if len(result.Incoming) != tt.wantIn {
				t.Errorf("CategorizeReferences() Incoming count = %d, want %d", len(result.Incoming), tt.wantIn)
			}

			// Verify cards are in correct categories
			for _, card := range result.Bidirectional {
				if !containsCard(tt.directLinks, card) || !containsCard(tt.backlinks, card) {
					t.Errorf("Card %s is in Bidirectional but not in both direct and backlinks", card.CardID)
				}
			}
			for _, card := range result.Outgoing {
				if !containsCard(tt.directLinks, card) {
					t.Errorf("Card %s is in Outgoing but not in direct links", card.CardID)
				}
				if containsCard(tt.backlinks, card) {
					t.Errorf("Card %s is in Outgoing but also in backlinks", card.CardID)
				}
			}
			for _, card := range result.Incoming {
				if !containsCard(tt.backlinks, card) {
					t.Errorf("Card %s is in Incoming but not in backlinks", card.CardID)
				}
				if containsCard(tt.directLinks, card) {
					t.Errorf("Card %s is in Incoming but also in direct links", card.CardID)
				}
			}

			// Verify all categories are sorted by CardID descending
			if !isSortedByCardID(result.Bidirectional) {
				t.Errorf("Bidirectional is not sorted by CardID descending")
			}
			if !isSortedByCardID(result.Outgoing) {
				t.Errorf("Outgoing is not sorted by CardID descending")
			}
			if !isSortedByCardID(result.Incoming) {
				t.Errorf("Incoming is not sorted by CardID descending")
			}
		})
	}
}

// Helper function to check if a card exists in a slice
func containsCard(cards []models.PartialCard, card models.PartialCard) bool {
	for _, c := range cards {
		if c.ID == card.ID {
			return true
		}
	}
	return false
}

// Helper function to check if cards are sorted by CardID descending
func isSortedByCardID(cards []models.PartialCard) bool {
	for i := 1; i < len(cards); i++ {
		if cards[i-1].CardID < cards[i].CardID {
			return false
		}
	}
	return true
}
