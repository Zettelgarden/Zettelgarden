package services

import (
	"fmt"
	"go-backend/models"
	"go-backend/tests"
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

// Integration tests for references service

// TestGetDirectLinks_Integration tests GetDirectLinks with real database
func TestGetDirectLinks_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target cards that will be referenced
	target1Params := models.EditCardParams{
		Title:  "Target Card 1",
		Body:   "Target body 1",
		CardID: "target1",
		Link:   "",
	}
	target1, err := CreateCard(s.DB, userID, target1Params)
	if err != nil {
		t.Fatalf("Failed to create target1: %v", err)
	}

	target2Params := models.EditCardParams{
		Title:  "Target Card 2",
		Body:   "Target body 2",
		CardID: "target2",
		Link:   "",
	}
	target2, err := CreateCard(s.DB, userID, target2Params)
	if err != nil {
		t.Fatalf("Failed to create target2: %v", err)
	}

	// Create a source card that references the targets
	sourceParams := models.EditCardParams{
		Title:  "Source Card",
		Body:   "This card references [target1] and [target2]",
		CardID: "source",
		Link:   "",
	}
	source, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get direct links from the source card
	directLinks, err := GetDirectLinks(s.DB, userID, source)
	if err != nil {
		t.Fatalf("GetDirectLinks failed: %v", err)
	}

	// Verify we got 2 direct links
	if len(directLinks) != 2 {
		t.Errorf("Expected 2 direct links, got %d", len(directLinks))
	}

	// Verify the direct links are the target cards
	foundTarget1, foundTarget2 := false, false
	for _, link := range directLinks {
		if link.CardID == "target1" {
			foundTarget1 = true
			if link.ID != target1.ID {
				t.Errorf("Target1 ID mismatch: expected %d, got %d", target1.ID, link.ID)
			}
		}
		if link.CardID == "target2" {
			foundTarget2 = true
			if link.ID != target2.ID {
				t.Errorf("Target2 ID mismatch: expected %d, got %d", target2.ID, link.ID)
			}
		}
	}

	if !foundTarget1 {
		t.Error("Target1 not found in direct links")
	}
	if !foundTarget2 {
		t.Error("Target2 not found in direct links")
	}
}

// TestGetDirectLinks_NonExistentTarget_Integration tests GetDirectLinks with non-existent referenced cards
func TestGetDirectLinks_NonExistentTarget_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card that references non-existent cards
	sourceParams := models.EditCardParams{
		Title:  "Source Card",
		Body:   "This references [nonexistent1] and [nonexistent2]",
		CardID: "source_nonexist",
		Link:   "",
	}
	source, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get direct links - should return empty slice, not error
	directLinks, err := GetDirectLinks(s.DB, userID, source)
	if err != nil {
		t.Fatalf("GetDirectLinks failed: %v", err)
	}

	// Verify we got 0 direct links (non-existent cards are skipped)
	if len(directLinks) != 0 {
		t.Errorf("Expected 0 direct links for non-existent cards, got %d", len(directLinks))
	}
}

// TestGetDirectLinks_NoReferences_Integration tests GetDirectLinks with card that has no references
func TestGetDirectLinks_NoReferences_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card with no references
	params := models.EditCardParams{
		Title:  "Card with no refs",
		Body:   "Just some plain text",
		CardID: "norefs_card",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Get direct links
	directLinks, err := GetDirectLinks(s.DB, userID, card)
	if err != nil {
		t.Fatalf("GetDirectLinks failed: %v", err)
	}

	if len(directLinks) != 0 {
		t.Errorf("Expected 0 direct links, got %d", len(directLinks))
	}
}

// TestGetReferences_Integration tests GetReferences with real database
func TestGetReferences_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target cards first
	target1Params := models.EditCardParams{
		Title:  "Target 1",
		Body:   "Target body 1",
		CardID: "ref_target1",
		Link:   "",
	}
	target1, err := CreateCard(s.DB, userID, target1Params)
	if err != nil {
		t.Fatalf("Failed to create target1: %v", err)
	}

	target2Params := models.EditCardParams{
		Title:  "Target 2",
		Body:   "Target body 2",
		CardID: "ref_target2",
		Link:   "",
	}
	target2, err := CreateCard(s.DB, userID, target2Params)
	if err != nil {
		t.Fatalf("Failed to create target2: %v", err)
	}

	// Create main card with direct links (must be created before backlink_source)
	mainParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "References [ref_target1] and [ref_target2]",
		CardID: "main_card",
		Link:   "",
	}
	mainCard, err := CreateCard(s.DB, userID, mainParams)
	if err != nil {
		t.Fatalf("Failed to create main card: %v", err)
	}

	// Now create backlink source (must come after main_card so backlink is established)
	backlinkSourceParams := models.EditCardParams{
		Title:  "Backlink Source",
		Body:   "This links to [main_card]",
		CardID: "backlink_source",
		Link:   "",
	}
	backlinkSource, err := CreateCard(s.DB, userID, backlinkSourceParams)
	if err != nil {
		t.Fatalf("Failed to create backlink source: %v", err)
	}

	// Get references for the main card
	references, err := GetReferences(s.DB, userID, mainCard)
	if err != nil {
		t.Fatalf("GetReferences failed: %v", err)
	}

	// Should have 3 references: 2 direct links + 1 backlink
	if len(references) != 3 {
		t.Errorf("Expected 3 references (2 direct + 1 backlink), got %d", len(references))
	}

	// Verify the cards are present
	foundTarget1, foundTarget2, foundBacklink := false, false, false
	for _, ref := range references {
		if ref.CardID == "ref_target1" {
			foundTarget1 = true
			if ref.ID != target1.ID {
				t.Errorf("Target1 ID mismatch: expected %d, got %d", target1.ID, ref.ID)
			}
		}
		if ref.CardID == "ref_target2" {
			foundTarget2 = true
			if ref.ID != target2.ID {
				t.Errorf("Target2 ID mismatch: expected %d, got %d", target2.ID, ref.ID)
			}
		}
		if ref.CardID == "backlink_source" {
			foundBacklink = true
			if ref.ID != backlinkSource.ID {
				t.Errorf("Backlink source ID mismatch: expected %d, got %d", backlinkSource.ID, ref.ID)
			}
		}
	}

	if !foundTarget1 {
		t.Error("Target1 not found in references")
	}
	if !foundTarget2 {
		t.Error("Target2 not found in references")
	}
	if !foundBacklink {
		t.Error("Backlink source not found in references")
	}

	// Verify references are sorted by CardID descending
	if !isSortedByCardID(references) {
		t.Error("References are not sorted by CardID descending")
	}

	// Verify no duplicates
	seen := make(map[int]bool)
	for _, ref := range references {
		if seen[ref.ID] {
			t.Errorf("Duplicate card found in references: ID %d", ref.ID)
		}
		seen[ref.ID] = true
	}
}

// TestGetReferences_Empty_Integration tests GetReferences with no references
func TestGetReferences_Empty_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	params := models.EditCardParams{
		Title:  "Isolated Card",
		Body:   "No references at all",
		CardID: "isolated_refs",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	references, err := GetReferences(s.DB, userID, card)
	if err != nil {
		t.Fatalf("GetReferences failed: %v", err)
	}

	if len(references) != 0 {
		t.Errorf("Expected 0 references, got %d", len(references))
	}
}

// TestGetCategorizedReferences_Integration tests GetCategorizedReferences with real database
func TestGetCategorizedReferences_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create bidirectional card - main will link to it, and it will link back to main
	bidirParams := models.EditCardParams{
		Title:  "Bidirectional Card",
		Body:   "Will link to [main]",
		CardID: "bidir",
		Link:   "",
	}
	bidir, err := CreateCard(s.DB, userID, bidirParams)
	if err != nil {
		t.Fatalf("Failed to create bidir: %v", err)
	}

	// Create outgoing-only card (referenced by main but doesn't link back)
	outgoingParams := models.EditCardParams{
		Title:  "Outgoing Only",
		Body:   "Doesn't link back",
		CardID: "outgoing",
		Link:   "",
	}
	outgoing, err := CreateCard(s.DB, userID, outgoingParams)
	if err != nil {
		t.Fatalf("Failed to create outgoing: %v", err)
	}

	// Create incoming-only card placeholder (will update after main_card exists)
	incomingParams := models.EditCardParams{
		Title:  "Incoming Only",
		Body:   "Will link to [main]",
		CardID: "incoming",
		Link:   "",
	}
	incoming, err := CreateCard(s.DB, userID, incomingParams)
	if err != nil {
		t.Fatalf("Failed to create incoming: %v", err)
	}

	// Create main card - links to bidir and outgoing
	mainParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "Links to [bidir] and [outgoing]",
		CardID: "main",
		Link:   "",
	}
	mainCard, err := CreateCard(s.DB, userID, mainParams)
	if err != nil {
		t.Fatalf("Failed to create main card: %v", err)
	}

	// Now update bidir to link back to main (creating true bidirectional relationship)
	bidirUpdateParams := models.EditCardParams{
		Title:  "Bidirectional Card",
		Body:   "Links to [main]",
		CardID: "bidir",
		Link:   "",
	}
	_, err = UpdateCard(s.DB, userID, bidir.ID, bidirUpdateParams)
	if err != nil {
		t.Fatalf("Failed to update bidir: %v", err)
	}

	// Update incoming to link to main (creating incoming backlink)
	incomingUpdateParams := models.EditCardParams{
		Title:  "Incoming Only",
		Body:   "Links to [main]",
		CardID: "incoming",
		Link:   "",
	}
	_, err = UpdateCard(s.DB, userID, incoming.ID, incomingUpdateParams)
	if err != nil {
		t.Fatalf("Failed to update incoming: %v", err)
	}

	// Get categorized references for main card
	categorized, err := GetCategorizedReferences(s.DB, userID, mainCard)
	if err != nil {
		t.Fatalf("GetCategorizedReferences failed: %v", err)
	}

	// Verify counts
	// Bidirectional: bidir (main -> bidir, and bidir -> main)
	// Outgoing: outgoing (main -> outgoing, but outgoing doesn't link back)
	// Incoming: incoming (incoming -> main, but main doesn't link to incoming)
	t.Logf("Bidirectional count: %d", len(categorized.Bidirectional))
	t.Logf("Outgoing count: %d", len(categorized.Outgoing))
	t.Logf("Incoming count: %d", len(categorized.Incoming))

	if len(categorized.Bidirectional) != 1 {
		t.Errorf("Expected 1 bidirectional reference, got %d", len(categorized.Bidirectional))
	}
	if len(categorized.Outgoing) != 1 {
		t.Errorf("Expected 1 outgoing reference, got %d", len(categorized.Outgoing))
	}
	if len(categorized.Incoming) != 1 {
		t.Errorf("Expected 1 incoming reference, got %d", len(categorized.Incoming))
	}

	// Verify bidirectional contains bidir card
	if len(categorized.Bidirectional) > 0 {
		if categorized.Bidirectional[0].ID != bidir.ID {
			t.Errorf("Bidirectional card ID mismatch: expected %d, got %d", bidir.ID, categorized.Bidirectional[0].ID)
		}
		if categorized.Bidirectional[0].CardID != "bidir" {
			t.Errorf("Bidirectional card CardID mismatch: expected 'bidir', got %s", categorized.Bidirectional[0].CardID)
		}
	}

	// Verify outgoing contains outgoing card
	foundOutgoing := false
	for _, card := range categorized.Outgoing {
		if card.CardID == "outgoing" {
			foundOutgoing = true
			if card.ID != outgoing.ID {
				t.Errorf("Outgoing card ID mismatch: expected %d, got %d", outgoing.ID, card.ID)
			}
		}
	}
	if !foundOutgoing {
		t.Error("Outgoing card not found in Outgoing category")
	}

	// Verify incoming contains incoming card
	foundIncoming := false
	for _, card := range categorized.Incoming {
		if card.CardID == "incoming" {
			foundIncoming = true
			if card.ID != incoming.ID {
				t.Errorf("Incoming card ID mismatch: expected %d, got %d", incoming.ID, card.ID)
			}
		}
	}
	if !foundIncoming {
		t.Error("Incoming card not found in Incoming category")
	}

	// Verify all categories are sorted by CardID descending
	if !isSortedByCardID(categorized.Bidirectional) {
		t.Error("Bidirectional is not sorted by CardID descending")
	}
	if !isSortedByCardID(categorized.Outgoing) {
		t.Error("Outgoing is not sorted by CardID descending")
	}
	if !isSortedByCardID(categorized.Incoming) {
		t.Error("Incoming is not sorted by CardID descending")
	}
}

// TestGetCategorizedReferences_Empty_Integration tests GetCategorizedReferences with no references
func TestGetCategorizedReferences_Empty_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	params := models.EditCardParams{
		Title:  "Isolated Card",
		Body:   "No references",
		CardID: "isolated_cat",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	categorized, err := GetCategorizedReferences(s.DB, userID, card)
	if err != nil {
		t.Fatalf("GetCategorizedReferences failed: %v", err)
	}

	if len(categorized.Bidirectional) != 0 {
		t.Errorf("Expected 0 bidirectional references, got %d", len(categorized.Bidirectional))
	}
	if len(categorized.Outgoing) != 0 {
		t.Errorf("Expected 0 outgoing references, got %d", len(categorized.Outgoing))
	}
	if len(categorized.Incoming) != 0 {
		t.Errorf("Expected 0 incoming references, got %d", len(categorized.Incoming))
	}
}

// TestGetCategorizedReferences_OnlyOutgoing_Integration tests with only outgoing references
func TestGetCategorizedReferences_OnlyOutgoing_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create cards that main will link to
	target1Params := models.EditCardParams{
		Title:  "Target 1",
		Body:   "Target",
		CardID: "onlyout_target1",
		Link:   "",
	}
	target1, err := CreateCard(s.DB, userID, target1Params)
	if err != nil {
		t.Fatalf("Failed to create target1: %v", err)
	}

	target2Params := models.EditCardParams{
		Title:  "Target 2",
		Body:   "Target",
		CardID: "onlyout_target2",
		Link:   "",
	}
	target2, err := CreateCard(s.DB, userID, target2Params)
	if err != nil {
		t.Fatalf("Failed to create target2: %v", err)
	}

	// Create main card with only outgoing links
	mainParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "Links to [onlyout_target1] and [onlyout_target2]",
		CardID: "onlyout_main",
		Link:   "",
	}
	mainCard, err := CreateCard(s.DB, userID, mainParams)
	if err != nil {
		t.Fatalf("Failed to create main card: %v", err)
	}

	categorized, err := GetCategorizedReferences(s.DB, userID, mainCard)
	if err != nil {
		t.Fatalf("GetCategorizedReferences failed: %v", err)
	}

	if len(categorized.Bidirectional) != 0 {
		t.Errorf("Expected 0 bidirectional references, got %d", len(categorized.Bidirectional))
	}
	if len(categorized.Outgoing) != 2 {
		t.Errorf("Expected 2 outgoing references, got %d", len(categorized.Outgoing))
	}
	if len(categorized.Incoming) != 0 {
		t.Errorf("Expected 0 incoming references, got %d", len(categorized.Incoming))
	}

	// Verify the correct cards are in outgoing
	foundTarget1, foundTarget2 := false, false
	for _, card := range categorized.Outgoing {
		if card.ID == target1.ID {
			foundTarget1 = true
		}
		if card.ID == target2.ID {
			foundTarget2 = true
		}
	}
	if !foundTarget1 {
		t.Error("Target1 not found in Outgoing")
	}
	if !foundTarget2 {
		t.Error("Target2 not found in Outgoing")
	}
}

// TestGetCategorizedReferences_OnlyIncoming_Integration tests with only incoming references
func TestGetCategorizedReferences_OnlyIncoming_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create main card (no outgoing links)
	mainParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "No outgoing links",
		CardID: "onlyin_main",
		Link:   "",
	}
	mainCard, err := CreateCard(s.DB, userID, mainParams)
	if err != nil {
		t.Fatalf("Failed to create main card: %v", err)
	}

	// Create cards that link to main
	linker1Params := models.EditCardParams{
		Title:  "Linker 1",
		Body:   "Links to [onlyin_main]",
		CardID: "onlyin_linker1",
		Link:   "",
	}
	linker1, err := CreateCard(s.DB, userID, linker1Params)
	if err != nil {
		t.Fatalf("Failed to create linker1: %v", err)
	}

	linker2Params := models.EditCardParams{
		Title:  "Linker 2",
		Body:   "Also links to [onlyin_main]",
		CardID: "onlyin_linker2",
		Link:   "",
	}
	linker2, err := CreateCard(s.DB, userID, linker2Params)
	if err != nil {
		t.Fatalf("Failed to create linker2: %v", err)
	}

	categorized, err := GetCategorizedReferences(s.DB, userID, mainCard)
	if err != nil {
		t.Fatalf("GetCategorizedReferences failed: %v", err)
	}

	if len(categorized.Bidirectional) != 0 {
		t.Errorf("Expected 0 bidirectional references, got %d", len(categorized.Bidirectional))
	}
	if len(categorized.Outgoing) != 0 {
		t.Errorf("Expected 0 outgoing references, got %d", len(categorized.Outgoing))
	}
	if len(categorized.Incoming) != 2 {
		t.Errorf("Expected 2 incoming references, got %d", len(categorized.Incoming))
	}

	// Verify the correct cards are in incoming
	foundLinker1, foundLinker2 := false, false
	for _, card := range categorized.Incoming {
		if card.ID == linker1.ID {
			foundLinker1 = true
		}
		if card.ID == linker2.ID {
			foundLinker2 = true
		}
	}
	if !foundLinker1 {
		t.Error("Linker1 not found in Incoming")
	}
	if !foundLinker2 {
		t.Error("Linker2 not found in Incoming")
	}
}

// TestGetReferences_UserIsolation_Integration tests that users can't see other users' referenced cards
func TestGetReferences_UserIsolation_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	user1ID := 1
	user2ID := 2

	// User 1 creates a card
	user1CardParams := models.EditCardParams{
		Title:  "User 1 Card",
		Body:   "User 1 content",
		CardID: "user1_card_isolation",
		Link:   "",
	}
	user1Card, err := CreateCard(s.DB, user1ID, user1CardParams)
	if err != nil {
		t.Fatalf("Failed to create user1 card: %v", err)
	}

	// User 2 creates a card that references user1's card (by card_id)
	user2Params := models.EditCardParams{
		Title:  "User 2 Card",
		Body:   "References [user1_card_isolation]",
		CardID: "user2_card_isolation",
		Link:   "",
	}
	user2Card, err := CreateCard(s.DB, user2ID, user2Params)
	if err != nil {
		t.Fatalf("Failed to create user2 card: %v", err)
	}

	// User 2 tries to get references - should not see user1's card
	references, err := GetReferences(s.DB, user2ID, user2Card)
	if err != nil {
		t.Fatalf("GetReferences failed: %v", err)
	}

	// User 2 should get 0 references since user1_card belongs to user1
	if len(references) != 0 {
		t.Errorf("Expected 0 references (user isolation), got %d", len(references))
	}

	// User 1 getting references for their card should also work
	references1, err := GetReferences(s.DB, user1ID, user1Card)
	if err != nil {
		t.Fatalf("GetReferences failed for user1: %v", err)
	}
	// User1's card doesn't reference anything
	if len(references1) != 0 {
		t.Errorf("Expected 0 references for user1 card, got %d", len(references1))
	}
}

// Edge case tests for reference resolution

// TestGetDirectLinks_DuplicateReferences_Integration tests handling of duplicate backlink IDs in card body
func TestGetDirectLinks_DuplicateReferences_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target card
	targetParams := models.EditCardParams{
		Title:  "Target Card",
		Body:   "Target body",
		CardID: "dup_target",
		Link:   "",
	}
	target, err := CreateCard(s.DB, userID, targetParams)
	if err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Create source card with duplicate references to same card
	sourceParams := models.EditCardParams{
		Title:  "Source with Duplicates",
		Body:   "This references [dup_target] once, and [dup_target] again, and [dup_target] one more time!",
		CardID: "dup_source",
		Link:   "",
	}
	source, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get direct links - should return only 1 link (duplicates removed)
	directLinks, err := GetDirectLinks(s.DB, userID, source)
	if err != nil {
		t.Fatalf("GetDirectLinks failed: %v", err)
	}

	if len(directLinks) != 1 {
		t.Errorf("Expected 1 direct link (duplicates removed), got %d", len(directLinks))
	}

	if len(directLinks) > 0 && directLinks[0].ID != target.ID {
		t.Errorf("Expected target card ID %d, got %d", target.ID, directLinks[0].ID)
	}
}

// TestGetReferences_EmptyBody_Integration tests GetReferences with empty card body
func TestGetReferences_EmptyBody_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	params := models.EditCardParams{
		Title:  "Empty Body Card",
		Body:   "",
		CardID: "empty_body_card",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	references, err := GetReferences(s.DB, userID, card)
	if err != nil {
		t.Fatalf("GetReferences failed: %v", err)
	}

	if len(references) != 0 {
		t.Errorf("Expected 0 references for empty body, got %d", len(references))
	}
}

// TestGetCategorizedReferences_EmptyBody_Integration tests GetCategorizedReferences with empty card body
func TestGetCategorizedReferences_EmptyBody_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	params := models.EditCardParams{
		Title:  "Empty Body Card",
		Body:   "",
		CardID: "empty_body_cat",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	categorized, err := GetCategorizedReferences(s.DB, userID, card)
	if err != nil {
		t.Fatalf("GetCategorizedReferences failed: %v", err)
	}

	if len(categorized.Bidirectional) != 0 {
		t.Errorf("Expected 0 bidirectional references, got %d", len(categorized.Bidirectional))
	}
	if len(categorized.Outgoing) != 0 {
		t.Errorf("Expected 0 outgoing references, got %d", len(categorized.Outgoing))
	}
	if len(categorized.Incoming) != 0 {
		t.Errorf("Expected 0 incoming references, got %d", len(categorized.Incoming))
	}
}

// TestExtractBacklinks_Duplicates tests that ExtractBacklinks handles duplicate IDs in body
func TestExtractBacklinks_Duplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    "This has [link1] and [link2]",
			expected: []string{"link1", "link2"},
		},
		{
			name:     "with duplicates",
			input:    "This has [link1] and [link1] again",
			expected: []string{"link1", "link1"}, // ExtractBacklinks doesn't dedupe
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "no backlinks",
			input:    "Just plain text with no links",
			expected: []string{},
		},
		{
			name:     "multiple duplicates",
			input:    "[a] [b] [a] [c] [b] [a]",
			expected: []string{"a", "b", "a", "c", "b", "a"},
		},
		{
			name:     "whitespace in brackets",
			input:    "This has [  link with spaces  ] and [another]",
			expected: []string{"  link with spaces  ", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBacklinks(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractBacklinks() returned %d items, expected %d", len(result), len(tt.expected))
			}
			// Compare results
			for i, r := range result {
				if i >= len(tt.expected) {
					break
				}
				if r != tt.expected[i] {
					t.Errorf("ExtractBacklinks()[%d] = %q, want %q", i, r, tt.expected[i])
				}
			}
		})
	}
}

// TestGetDirectLinks_MixedExistence_Integration tests GetDirectLinks with mix of existing and non-existing cards
func TestGetDirectLinks_MixedExistence_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create only one target card
	targetParams := models.EditCardParams{
		Title:  "Existing Target",
		Body:   "Target body",
		CardID: "existing_target",
		Link:   "",
	}
	_, err := CreateCard(s.DB, userID, targetParams)
	if err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Create source card that references existing and non-existing cards
	sourceParams := models.EditCardParams{
		Title:  "Mixed Source",
		Body:   "References [existing_target], [nonexistent1], and [nonexistent2]",
		CardID: "mixed_source",
		Link:   "",
	}
	source, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get direct links - should return only the existing card
	directLinks, err := GetDirectLinks(s.DB, userID, source)
	if err != nil {
		t.Fatalf("GetDirectLinks failed: %v", err)
	}

	if len(directLinks) != 1 {
		t.Errorf("Expected 1 direct link (only existing card), got %d", len(directLinks))
	}

	if len(directLinks) > 0 && directLinks[0].CardID != "existing_target" {
		t.Errorf("Expected existing_target, got %s", directLinks[0].CardID)
	}
}

// TestGetReferences_MalformedBacklinks_Integration tests handling of malformed backlink syntax
func TestGetReferences_MalformedBacklinks_Integration(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Test various malformed backlink patterns
	testCases := []struct {
		name     string
		body     string
		expected int // expected number of backlinks extracted
	}{
		{
			name:     "unclosed bracket",
			body:     "This has [unclosed",
			expected: 0,
		},
		{
			name:     "unopened bracket",
			body:     "This has unclosed]",
			expected: 0,
		},
		{
			name:     "empty brackets",
			body:     "This has []",
			expected: 1, // Empty string is extracted
		},
		{
			name:     "nested brackets",
			body:     "This has [outer [inner]]",
			expected: 2, // Both "outer [inner" and "inner" are extracted
		},
		{
			name:     "markdown link syntax",
			body:     "This has [text](url)",
			expected: 0, // Markdown links should be excluded
		},
		{
			name:     "valid backlinks only",
			body:     "This has [valid1] and [valid2]",
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := models.EditCardParams{
				Title:  fmt.Sprintf("Test %s", tc.name),
				Body:   tc.body,
				CardID: fmt.Sprintf("malformed_%s", tc.name),
				Link:   "",
			}
			card, err := CreateCard(s.DB, userID, params)
			if err != nil {
				t.Fatalf("Failed to create card: %v", err)
			}

			// Extract backlinks to verify syntax parsing
			backlinks := ExtractBacklinks(tc.body)
			if len(backlinks) != tc.expected {
				t.Errorf("ExtractBacklinks returned %d items, expected %d for body: %s", len(backlinks), tc.expected, tc.body)
			}

			// GetDirectLinks should handle whatever ExtractBacklinks returns
			directLinks, err := GetDirectLinks(s.DB, userID, card)
			if err != nil {
				t.Fatalf("GetDirectLinks failed: %v", err)
			}
			// DirectLinks filters out non-existent cards, so count may differ
			// The important thing is it doesn't error
			_ = directLinks
		})
	}
}
