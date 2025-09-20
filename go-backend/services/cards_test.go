package services

import (
	"go-backend/models"
	"go-backend/tests"
	"reflect"
	"testing"
)

func TestExtractBacklinks(t *testing.T) {
	text := "This is a sample text with [link1] and [another link]."
	expected := []string{"link1", "another link"}
	result := ExtractBacklinks(text)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestGetParentCardId(t *testing.T) {
	// Test cases for new dual-format function
	testCases := []struct {
		name     string
		cardID   string
		expected string
	}{
		// Old format (alternating separators) test cases
		{"Old format - complex hierarchy", "SP24/P.19", "SP24/P"},
		{"Old format - simple hierarchy", "1957/A.135", "1957/A"},
		{"Old format - deep hierarchy", "1957/A.135/B.2", "1957/A.135/B"},
		{"Old format - very deep", "SP170/A.1/A.1/A.1/A.1", "SP170/A.1/A.1/A.1/A"},
		{"Old format - root card", "SP24", "SP24"},
		{"Old format - root card numeric", "1957", "1957"},
		{"Old format - single level", "1", "1"},

		// New format test cases
		{"New format - dot separators", "cardA.1.2", "cardA.1"},
		{"New format - slash separators", "cardA/1/2", "cardA/1"},
		{"New format - dash separators", "cardA-1-2", "cardA-1"},
		{"New format - mixed separators", "cardA.1/2", "cardA.1"},
		{"New format - mixed separators 2", "cardA/1.2", "cardA/1"},
		{"New format - mixed separators 3", "cardA-1.2", "cardA-1"},
		{"New format - single level", "cardA.1", "cardA"},
		{"New format - single level slash", "cardA/1", "cardA"},
		{"New format - root card", "cardA", "cardA"},
		{"New format - complex name", "myProject.v2.1", "myProject.v2"},
		{"New format - deep hierarchy", "project.1.2.3", "project.1.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DiscoverParentId(tc.cardID)
			if result != tc.expected {
				t.Errorf("function returned wrong result for %q, got %v want %v", tc.cardID, result, tc.expected)
			}
		})
	}
}

func TestCreateCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body with [1] reference",
		CardID: "test123",
		Link:   "http://example.com",
	}

	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("CreateCard failed: %v", err)
	}

	if card.Title != params.Title {
		t.Errorf("Expected title %v, got %v", params.Title, card.Title)
	}
	if card.Body != params.Body {
		t.Errorf("Expected body %v, got %v", params.Body, card.Body)
	}
	if card.CardID != params.CardID {
		t.Errorf("Expected card_id %v, got %v", params.CardID, card.CardID)
	}
	if card.UserID != userID {
		t.Errorf("Expected user_id %v, got %v", userID, card.UserID)
	}
}

func TestCreateCardWithParent(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card first
	parentParams := models.EditCardParams{
		Title:  "Parent Card",
		Body:   "Parent Body",
		CardID: "parent",
		Link:   "",
	}
	parentCard, err := CreateCard(s.DB, userID, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent card: %v", err)
	}

	// Create child card
	childParams := models.EditCardParams{
		Title:  "Child Card",
		Body:   "Child Body",
		CardID: "parent/child",
		Link:   "",
	}
	childCard, err := CreateCard(s.DB, userID, childParams)
	if err != nil {
		t.Fatalf("Failed to create child card: %v", err)
	}

	if childCard.ParentID != parentCard.ID {
		t.Errorf("Expected parent_id %v, got %v", parentCard.ID, childCard.ParentID)
	}
}

func TestCreateCardDuplicateCardID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "duplicate",
		Link:   "",
	}

	// Create first card
	_, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("First CreateCard failed: %v", err)
	}

	// Try to create second card with same CardID
	_, err = CreateCard(s.DB, userID, params)
	if err == nil {
		t.Error("Expected error for duplicate card_id, but got none")
	}
}

func TestUpdateCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card first
	createParams := models.EditCardParams{
		Title:  "Original Title",
		Body:   "Original Body",
		CardID: "update_test",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, createParams)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Update the card
	updateParams := models.EditCardParams{
		Title:  "Updated Title",
		Body:   "Updated Body with [test] reference",
		CardID: "update_test",
		Link:   "http://updated.com",
	}

	updatedCard, err := UpdateCard(s.DB, userID, card.ID, updateParams)
	if err != nil {
		t.Fatalf("UpdateCard failed: %v", err)
	}

	if updatedCard.Title != updateParams.Title {
		t.Errorf("Expected updated title %v, got %v", updateParams.Title, updatedCard.Title)
	}
	if updatedCard.Body != updateParams.Body {
		t.Errorf("Expected updated body %v, got %v", updateParams.Body, updatedCard.Body)
	}
	if updatedCard.Link != updateParams.Link {
		t.Errorf("Expected updated link %v, got %v", updateParams.Link, updatedCard.Link)
	}
}

func TestGetFullCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "get_test",
		Link:   "http://example.com",
	}
	createdCard, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Get the card
	retrievedCard, err := GetFullCard(s.DB, userID, createdCard.ID)
	if err != nil {
		t.Fatalf("GetFullCard failed: %v", err)
	}

	if retrievedCard.ID != createdCard.ID {
		t.Errorf("Expected ID %v, got %v", createdCard.ID, retrievedCard.ID)
	}
	if retrievedCard.Title != params.Title {
		t.Errorf("Expected title %v, got %v", params.Title, retrievedCard.Title)
	}
	if retrievedCard.Body != params.Body {
		t.Errorf("Expected body %v, got %v", params.Body, retrievedCard.Body)
	}
	if retrievedCard.CardID != params.CardID {
		t.Errorf("Expected card_id %v, got %v", params.CardID, retrievedCard.CardID)
	}
}

func TestGetFullCardNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	nonExistentID := 99999

	_, err := GetFullCard(s.DB, userID, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent card, but got none")
	}
	if err.Error() != "card not found" {
		t.Errorf("Expected 'card not found' error, got %v", err.Error())
	}
}

func TestGetPartialCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "partial_test",
		Link:   "http://example.com",
	}
	createdCard, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Get the partial card by ID
	partialCard, err := GetPartialCard(s.DB, userID, createdCard.ID)
	if err != nil {
		t.Fatalf("GetPartialCard failed: %v", err)
	}

	if partialCard.ID != createdCard.ID {
		t.Errorf("Expected ID %v, got %v", createdCard.ID, partialCard.ID)
	}
	if partialCard.Title != params.Title {
		t.Errorf("Expected title %v, got %v", params.Title, partialCard.Title)
	}
	if partialCard.CardID != params.CardID {
		t.Errorf("Expected card_id %v, got %v", params.CardID, partialCard.CardID)
	}

	// Get the partial card by CardID
	partialCardByCardID, err := GetPartialCardByCardID(s.DB, userID, params.CardID)
	if err != nil {
		t.Fatalf("GetPartialCardByCardID failed: %v", err)
	}

	if partialCardByCardID.ID != createdCard.ID {
		t.Errorf("Expected ID %v, got %v", createdCard.ID, partialCardByCardID.ID)
	}
	if partialCardByCardID.CardID != params.CardID {
		t.Errorf("Expected card_id %v, got %v", params.CardID, partialCardByCardID.CardID)
	}
}

func TestGetChildCards(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card",
		Body:   "Parent Body",
		CardID: "parent_children",
		Link:   "",
	}
	parentCard, err := CreateCard(s.DB, userID, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent card: %v", err)
	}

	// Create child cards
	child1Params := models.EditCardParams{
		Title:  "Child 1",
		Body:   "Child 1 Body",
		CardID: "parent_children/child1",
		Link:   "",
	}
	child1, err := CreateCard(s.DB, userID, child1Params)
	if err != nil {
		t.Fatalf("Failed to create child1: %v", err)
	}

	child2Params := models.EditCardParams{
		Title:  "Child 2",
		Body:   "Child 2 Body",
		CardID: "parent_children/child2",
		Link:   "",
	}
	child2, err := CreateCard(s.DB, userID, child2Params)
	if err != nil {
		t.Fatalf("Failed to create child2: %v", err)
	}

	// Get child cards
	children, err := GetChildCards(s.DB, userID, parentCard.ID)
	if err != nil {
		t.Fatalf("GetChildCards failed: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %v", len(children))
	}

	// Check that both children are in the result
	foundChild1, foundChild2 := false, false
	for _, child := range children {
		if child.ID == child1.ID {
			foundChild1 = true
		}
		if child.ID == child2.ID {
			foundChild2 = true
		}
	}

	if !foundChild1 {
		t.Error("Child 1 not found in children")
	}
	if !foundChild2 {
		t.Error("Child 2 not found in children")
	}
}

func TestDeleteCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card to delete
	params := models.EditCardParams{
		Title:  "Card to Delete",
		Body:   "Delete me",
		CardID: "delete_test",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Delete the card
	err = DeleteCard(s.DB, userID, card.ID)
	if err != nil {
		t.Fatalf("DeleteCard failed: %v", err)
	}

	// Verify card is deleted (should not be found)
	_, err = GetFullCard(s.DB, userID, card.ID)
	if err == nil {
		t.Error("Expected error when getting deleted card, but got none")
	}
	if err.Error() != "card not found" {
		t.Errorf("Expected 'card not found' error, got %v", err.Error())
	}
}

func TestDeleteCardWithBacklinks(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target card
	targetParams := models.EditCardParams{
		Title:  "Target Card",
		Body:   "Target Body",
		CardID: "target_card",
		Link:   "",
	}
	targetCard, err := CreateCard(s.DB, userID, targetParams)
	if err != nil {
		t.Fatalf("Failed to create target card: %v", err)
	}

	// Create source card that links to target
	sourceParams := models.EditCardParams{
		Title:  "Source Card",
		Body:   "This links to [target_card]",
		CardID: "source_card",
		Link:   "",
	}
	_, err = CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source card: %v", err)
	}

	// Try to delete target card (should fail due to backlinks)
	err = DeleteCard(s.DB, userID, targetCard.ID)
	if err == nil {
		t.Error("Expected error when deleting card with backlinks, but got none")
	}
	if err.Error() != "card has backlinks, cannot be deleted" {
		t.Errorf("Expected backlinks error, got %v", err.Error())
	}
}

func TestDeleteCardWithChildren(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card",
		Body:   "Parent Body",
		CardID: "parent_delete",
		Link:   "",
	}
	parentCard, err := CreateCard(s.DB, userID, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent card: %v", err)
	}

	// Create child card
	childParams := models.EditCardParams{
		Title:  "Child Card",
		Body:   "Child Body",
		CardID: "parent_delete/child",
		Link:   "",
	}
	_, err = CreateCard(s.DB, userID, childParams)
	if err != nil {
		t.Fatalf("Failed to create child card: %v", err)
	}

	// Try to delete parent card (should fail due to children)
	err = DeleteCard(s.DB, userID, parentCard.ID)
	if err == nil {
		t.Error("Expected error when deleting card with children, but got none")
	}
	if err.Error() != "card has children, cannot be deleted" {
		t.Errorf("Expected children error, got %v", err.Error())
	}
}

func TestGetBacklinks(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target card
	targetParams := models.EditCardParams{
		Title:  "Target Card",
		Body:   "Target Body",
		CardID: "target_backlinks",
		Link:   "",
	}
	targetCard, err := CreateCard(s.DB, userID, targetParams)
	if err != nil {
		t.Fatalf("Failed to create target card: %v", err)
	}

	// Create source card that links to target
	sourceParams := models.EditCardParams{
		Title:  "Source Card",
		Body:   "This links to [target_backlinks]",
		CardID: "source_backlinks",
		Link:   "",
	}
	sourceCard, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source card: %v", err)
	}

	// Get backlinks for target card
	backlinks, err := GetBacklinks(s.DB, userID, targetCard.CardID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}

	if len(backlinks) != 1 {
		t.Errorf("Expected 1 backlink, got %v", len(backlinks))
	}

	if len(backlinks) > 0 && backlinks[0].ID != sourceCard.ID {
		t.Errorf("Expected backlink to source card ID %v, got %v", sourceCard.ID, backlinks[0].ID)
	}
}

func TestUpdateBacklinks(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create target cards
	target1Params := models.EditCardParams{
		Title:  "Target 1",
		Body:   "Target 1 Body",
		CardID: "target1",
		Link:   "",
	}
	target1, err := CreateCard(s.DB, userID, target1Params)
	if err != nil {
		t.Fatalf("Failed to create target1: %v", err)
	}

	target2Params := models.EditCardParams{
		Title:  "Target 2",
		Body:   "Target 2 Body",
		CardID: "target2",
		Link:   "",
	}
	target2, err := CreateCard(s.DB, userID, target2Params)
	if err != nil {
		t.Fatalf("Failed to create target2: %v", err)
	}

	// Create source card with links
	sourceParams := models.EditCardParams{
		Title:  "Source Card",
		Body:   "This links to [target1] and [target2]",
		CardID: "source_update",
		Link:   "",
	}
	sourceCard, err := CreateCard(s.DB, userID, sourceParams)
	if err != nil {
		t.Fatalf("Failed to create source card: %v", err)
	}

	// Verify backlinks were created
	backlinks1, _ := GetBacklinks(s.DB, userID, target1.CardID)
	backlinks2, _ := GetBacklinks(s.DB, userID, target2.CardID)

	if len(backlinks1) != 1 || len(backlinks2) != 1 {
		t.Errorf("Expected 1 backlink for each target, got %v and %v", len(backlinks1), len(backlinks2))
	}

	// Update source card to only link to target1
	updateParams := models.EditCardParams{
		Title:  "Updated Source Card",
		Body:   "This only links to [target1]",
		CardID: "source_update",
		Link:   "",
	}
	_, err = UpdateCard(s.DB, userID, sourceCard.ID, updateParams)
	if err != nil {
		t.Fatalf("UpdateCard failed: %v", err)
	}

	// Verify backlinks updated correctly
	backlinks1After, _ := GetBacklinks(s.DB, userID, target1.CardID)
	backlinks2After, _ := GetBacklinks(s.DB, userID, target2.CardID)

	if len(backlinks1After) != 1 {
		t.Errorf("Expected 1 backlink for target1 after update, got %v", len(backlinks1After))
	}
	if len(backlinks2After) != 0 {
		t.Errorf("Expected 0 backlinks for target2 after update, got %v", len(backlinks2After))
	}
}
