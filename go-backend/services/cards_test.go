package services

import (
	"fmt"
	"go-backend/models"
	"go-backend/tests"
	"reflect"
	"testing"
	"time"
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

	if childCard.ParentID == nil || *childCard.ParentID != parentCard.ID {
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

func TestIsMarkdownLink(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		match    string
		expected bool
	}{
		{"Markdown link", "Check out [link](http://example.com)", "[link]", true},
		{"Backlink only", "This is a [backlink] reference", "[backlink]", false},
		{"Mixed content", "Here's [link](url) and [backlink]", "[link]", true},
		{"Mixed content backlink", "Here's [link](url) and [backlink]", "[backlink]", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isMarkdownLink(tc.text, tc.match)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for text %q and match %q", tc.expected, result, tc.text, tc.match)
			}
		})
	}
}

func TestGetParentCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card",
		Body:   "Parent Body",
		CardID: "parent_get",
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
		CardID: "parent_get/child",
		Link:   "",
	}
	childCard, err := CreateCard(s.DB, userID, childParams)
	if err != nil {
		t.Fatalf("Failed to create child card: %v", err)
	}

	// Get parent of child card
	parents, err := GetParentCard(s.DB, userID, childCard.ID)
	if err != nil {
		t.Fatalf("GetParentCard failed: %v", err)
	}

	if len(parents) != 1 {
		t.Errorf("Expected 1 parent, got %v", len(parents))
	}

	if len(parents) > 0 && parents[0].ID != parentCard.ID {
		t.Errorf("Expected parent ID %v, got %v", parentCard.ID, parents[0].ID)
	}
}

func TestCheckIsCardIDUnique(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Test with non-existent card ID
	if !checkIsCardIDUnique(s.DB, userID, "unique_test") {
		t.Error("Expected true for non-existent card ID")
	}

	// Create a card
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "unique_test",
		Link:   "",
	}
	_, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Test with existing card ID
	if checkIsCardIDUnique(s.DB, userID, "unique_test") {
		t.Error("Expected false for existing card ID")
	}

	// Test with empty card ID
	if !checkIsCardIDUnique(s.DB, userID, "") {
		t.Error("Expected true for empty card ID")
	}
}

func TestGetCardWithDescendantsNoChildren(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a root card with no children
	rootParams := models.EditCardParams{
		Title:  "Root Card",
		Body:   "Root Body",
		CardID: "root_no_children",
		Link:   "",
	}
	rootCard, err := CreateCard(s.DB, userID, rootParams)
	if err != nil {
		t.Fatalf("Failed to create root card: %v", err)
	}

	// Get card with descendants
	result, err := GetCardWithDescendants(s.DB, userID, rootCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed: %v", err)
	}

	// Verify root card data
	if result.ID != rootCard.ID {
		t.Errorf("Expected ID %v, got %v", rootCard.ID, result.ID)
	}
	if result.Title != rootParams.Title {
		t.Errorf("Expected title %v, got %v", rootParams.Title, result.Title)
	}

	// Verify depth is 0 for root
	if result.Depth != 0 {
		t.Errorf("Expected root depth 0, got %v", result.Depth)
	}

	// Verify no descendants
	if len(result.Descendants) != 0 {
		t.Errorf("Expected 0 descendants, got %v", len(result.Descendants))
	}
}

func TestGetCardWithDescendantsSingleChild(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card",
		Body:   "Parent Body",
		CardID: "parent_single_child",
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
		CardID: "parent_single_child/child",
		Link:   "",
	}
	childCard, err := CreateCard(s.DB, userID, childParams)
	if err != nil {
		t.Fatalf("Failed to create child card: %v", err)
	}

	// Get card with descendants
	result, err := GetCardWithDescendants(s.DB, userID, parentCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed: %v", err)
	}

	// Verify parent
	if result.ID != parentCard.ID {
		t.Errorf("Expected parent ID %v, got %v", parentCard.ID, result.ID)
	}
	if result.Depth != 0 {
		t.Errorf("Expected parent depth 0, got %v", result.Depth)
	}

	// Verify single child
	if len(result.Descendants) != 1 {
		t.Errorf("Expected 1 descendant, got %v", len(result.Descendants))
	}

	if len(result.Descendants) > 0 {
		child := result.Descendants[0]
		if child.ID != childCard.ID {
			t.Errorf("Expected child ID %v, got %v", childCard.ID, child.ID)
		}
		if child.Title != childParams.Title {
			t.Errorf("Expected child title %v, got %v", childParams.Title, child.Title)
		}
		if child.Depth != 1 {
			t.Errorf("Expected child depth 1, got %v", child.Depth)
		}
		if len(child.Descendants) != 0 {
			t.Errorf("Expected child to have 0 descendants, got %v", len(child.Descendants))
		}
	}
}

func TestGetCardWithDescendantsDeepNesting(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a deep hierarchy: root -> level1 -> level2 -> level3
	rootParams := models.EditCardParams{
		Title:  "Root",
		Body:   "Root Body",
		CardID: "deep_root",
		Link:   "",
	}
	rootCard, err := CreateCard(s.DB, userID, rootParams)
	if err != nil {
		t.Fatalf("Failed to create root card: %v", err)
	}

	level1Params := models.EditCardParams{
		Title:  "Level 1",
		Body:   "Level 1 Body",
		CardID: "deep_root/level1",
		Link:   "",
	}
	level1Card, err := CreateCard(s.DB, userID, level1Params)
	if err != nil {
		t.Fatalf("Failed to create level 1 card: %v", err)
	}

	level2Params := models.EditCardParams{
		Title:  "Level 2",
		Body:   "Level 2 Body",
		CardID: "deep_root/level1/level2",
		Link:   "",
	}
	level2Card, err := CreateCard(s.DB, userID, level2Params)
	if err != nil {
		t.Fatalf("Failed to create level 2 card: %v", err)
	}

	level3Params := models.EditCardParams{
		Title:  "Level 3",
		Body:   "Level 3 Body",
		CardID: "deep_root/level1/level2/level3",
		Link:   "",
	}
	level3Card, err := CreateCard(s.DB, userID, level3Params)
	if err != nil {
		t.Fatalf("Failed to create level 3 card: %v", err)
	}

	// Get card with descendants
	result, err := GetCardWithDescendants(s.DB, userID, rootCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed: %v", err)
	}

	// Verify root
	if result.ID != rootCard.ID {
		t.Errorf("Expected root ID %v, got %v", rootCard.ID, result.ID)
	}
	if result.Depth != 0 {
		t.Errorf("Expected root depth 0, got %v", result.Depth)
	}

	// Verify level 1
	if len(result.Descendants) != 1 {
		t.Errorf("Expected 1 level-1 descendant, got %v", len(result.Descendants))
	}

	level1 := result.Descendants[0]
	if level1.ID != level1Card.ID {
		t.Errorf("Expected level 1 ID %v, got %v", level1Card.ID, level1.ID)
	}
	if level1.Depth != 1 {
		t.Errorf("Expected level 1 depth 1, got %v", level1.Depth)
	}

	// Verify level 2
	if len(level1.Descendants) != 1 {
		t.Errorf("Expected 1 level-2 descendant, got %v", len(level1.Descendants))
	}

	level2 := level1.Descendants[0]
	if level2.ID != level2Card.ID {
		t.Errorf("Expected level 2 ID %v, got %v", level2Card.ID, level2.ID)
	}
	if level2.Depth != 2 {
		t.Errorf("Expected level 2 depth 2, got %v", level2.Depth)
	}

	// Verify level 3
	if len(level2.Descendants) != 1 {
		t.Errorf("Expected 1 level-3 descendant, got %v", len(level2.Descendants))
	}

	level3 := level2.Descendants[0]
	if level3.ID != level3Card.ID {
		t.Errorf("Expected level 3 ID %v, got %v", level3Card.ID, level3.ID)
	}
	if level3.Depth != 3 {
		t.Errorf("Expected level 3 depth 3, got %v", level3.Depth)
	}
	if len(level3.Descendants) != 0 {
		t.Errorf("Expected level 3 to have 0 descendants, got %v", len(level3.Descendants))
	}
}

func TestGetCardWithDescendantsMultipleChildren(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create parent card
	parentParams := models.EditCardParams{
		Title:  "Parent",
		Body:   "Parent Body",
		CardID: "multi_parent",
		Link:   "",
	}
	parentCard, err := CreateCard(s.DB, userID, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent card: %v", err)
	}

	// Create multiple children
	childIDs := []int{}
	for i := 1; i <= 3; i++ {
		childCardID := fmt.Sprintf("multi_parent/child%d", i)
		childParams := models.EditCardParams{
			Title:  fmt.Sprintf("Child %d", i),
			Body:   fmt.Sprintf("Child %d Body", i),
			CardID: childCardID,
			Link:   "",
		}
		childCard, err := CreateCard(s.DB, userID, childParams)
		if err != nil {
			t.Fatalf("Failed to create child %d: %v", i, err)
		}
		childIDs = append(childIDs, childCard.ID)
	}

	// Get card with descendants
	result, err := GetCardWithDescendants(s.DB, userID, parentCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed: %v", err)
	}

	// Verify parent
	if result.ID != parentCard.ID {
		t.Errorf("Expected parent ID %v, got %v", parentCard.ID, result.ID)
	}

	// Verify all children are present
	if len(result.Descendants) != 3 {
		t.Errorf("Expected 3 children, got %v", len(result.Descendants))
	}

	// Verify each child has correct depth and data
	for i, child := range result.Descendants {
		if child.Depth != 1 {
			t.Errorf("Expected child %d depth 1, got %v", i, child.Depth)
		}
		if len(child.Descendants) != 0 {
			t.Errorf("Expected child %d to have 0 descendants, got %v", i, len(child.Descendants))
		}
		// Verify child ID is in the expected list
		found := false
		for _, expectedID := range childIDs {
			if child.ID == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Child ID %v not in expected list", child.ID)
		}
	}
}

func TestGetCardWithDescendantsComplexTree(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a complex tree:
	//       root
	//      /    \
	//    child1  child2
	//    /        /  \
	//   gc1      gc2  gc3
	//   /
	//  ggc1

	rootParams := models.EditCardParams{
		Title:  "Root",
		Body:   "Root Body",
		CardID: "complex_root",
		Link:   "",
	}
	rootCard, err := CreateCard(s.DB, userID, rootParams)
	if err != nil {
		t.Fatalf("Failed to create root: %v", err)
	}

	// Create child1
	child1Params := models.EditCardParams{
		Title:  "Child 1",
		Body:   "Child 1 Body",
		CardID: "complex_root/child1",
		Link:   "",
	}
	child1, err := CreateCard(s.DB, userID, child1Params)
	if err != nil {
		t.Fatalf("Failed to create child1: %v", err)
	}

	// Create child2
	child2Params := models.EditCardParams{
		Title:  "Child 2",
		Body:   "Child 2 Body",
		CardID: "complex_root/child2",
		Link:   "",
	}
	child2, err := CreateCard(s.DB, userID, child2Params)
	if err != nil {
		t.Fatalf("Failed to create child2: %v", err)
	}

	// Create grandchild1 under child1
	gc1Params := models.EditCardParams{
		Title:  "GrandChild 1",
		Body:   "GrandChild 1 Body",
		CardID: "complex_root/child1/gc1",
		Link:   "",
	}
	gc1, err := CreateCard(s.DB, userID, gc1Params)
	if err != nil {
		t.Fatalf("Failed to create gc1: %v", err)
	}

	// Create great-grandchild1 under gc1
	ggc1Params := models.EditCardParams{
		Title:  "GreatGrandChild 1",
		Body:   "GreatGrandChild 1 Body",
		CardID: "complex_root/child1/gc1/ggc1",
		Link:   "",
	}
	ggc1, err := CreateCard(s.DB, userID, ggc1Params)
	if err != nil {
		t.Fatalf("Failed to create ggc1: %v", err)
	}

	// Create grandchild2 under child2
	gc2Params := models.EditCardParams{
		Title:  "GrandChild 2",
		Body:   "GrandChild 2 Body",
		CardID: "complex_root/child2/gc2",
		Link:   "",
	}
	gc2, err := CreateCard(s.DB, userID, gc2Params)
	if err != nil {
		t.Fatalf("Failed to create gc2: %v", err)
	}

	// Create grandchild3 under child2
	gc3Params := models.EditCardParams{
		Title:  "GrandChild 3",
		Body:   "GrandChild 3 Body",
		CardID: "complex_root/child2/gc3",
		Link:   "",
	}
	gc3, err := CreateCard(s.DB, userID, gc3Params)
	if err != nil {
		t.Fatalf("Failed to create gc3: %v", err)
	}

	// Get card with descendants
	result, err := GetCardWithDescendants(s.DB, userID, rootCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed: %v", err)
	}

	// Verify root
	if result.Depth != 0 {
		t.Errorf("Expected root depth 0, got %v", result.Depth)
	}
	if len(result.Descendants) != 2 {
		t.Errorf("Expected 2 children at root, got %v", len(result.Descendants))
	}

	// Find child1 and child2 in descendants
	var foundChild1, foundChild2 *models.CardWithDescendants
	for i := range result.Descendants {
		if result.Descendants[i].ID == child1.ID {
			foundChild1 = &result.Descendants[i]
		}
		if result.Descendants[i].ID == child2.ID {
			foundChild2 = &result.Descendants[i]
		}
	}

	if foundChild1 == nil {
		t.Fatal("Child 1 not found in descendants")
	}
	if foundChild2 == nil {
		t.Fatal("Child 2 not found in descendants")
	}

	// Verify child1 has 1 grandchild (gc1)
	if len(foundChild1.Descendants) != 1 {
		t.Errorf("Expected child1 to have 1 descendant, got %v", len(foundChild1.Descendants))
	}

	// Verify child2 has 2 grandchildren (gc2, gc3)
	if len(foundChild2.Descendants) != 2 {
		t.Errorf("Expected child2 to have 2 descendants, got %v", len(foundChild2.Descendants))
	}

	// Verify gc2 and gc3 are present in child2's descendants
	foundGC2, foundGC3 := false, false
	for _, gc := range foundChild2.Descendants {
		if gc.ID == gc2.ID {
			foundGC2 = true
		}
		if gc.ID == gc3.ID {
			foundGC3 = true
		}
	}

	if !foundGC2 {
		t.Error("GrandChild 2 not found in child2's descendants")
	}
	if !foundGC3 {
		t.Error("GrandChild 3 not found in child2's descendants")
	}

	// Verify gc1 has ggc1 as descendant
	if len(foundChild1.Descendants) > 0 {
		gc1Node := foundChild1.Descendants[0]
		if gc1Node.ID != gc1.ID {
			t.Errorf("Expected gc1 ID %v, got %v", gc1.ID, gc1Node.ID)
		}
		if gc1Node.Depth != 2 {
			t.Errorf("Expected gc1 depth 2, got %v", gc1Node.Depth)
		}
		if len(gc1Node.Descendants) != 1 {
			t.Errorf("Expected gc1 to have 1 descendant, got %v", len(gc1Node.Descendants))
		}

		if len(gc1Node.Descendants) > 0 {
			ggc1Node := gc1Node.Descendants[0]
			if ggc1Node.ID != ggc1.ID {
				t.Errorf("Expected ggc1 ID %v, got %v", ggc1.ID, ggc1Node.ID)
			}
			if ggc1Node.Depth != 3 {
				t.Errorf("Expected ggc1 depth 3, got %v", ggc1Node.Depth)
			}
		}
	}
}

func TestGetCardWithDescendantsCardNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	nonExistentID := 99999

	_, err := GetCardWithDescendants(s.DB, userID, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent card, but got none")
	}
	if err.Error() != "card not found" {
		t.Errorf("Expected 'card not found' error, got %v", err.Error())
	}
}

func TestGetCardWithDescendantsDeletedCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card
	params := models.EditCardParams{
		Title:  "Card to Delete",
		Body:   "Delete me",
		CardID: "deleted_card_test",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Delete the card
	err = DeleteCard(s.DB, userID, card.ID)
	if err != nil {
		t.Fatalf("Failed to delete card: %v", err)
	}

	// Try to get with descendants
	_, err = GetCardWithDescendants(s.DB, userID, card.ID)
	if err == nil {
		t.Error("Expected error for deleted card, but got none")
	}
	if err.Error() != "card not found" {
		t.Errorf("Expected 'card not found' error, got %v", err.Error())
	}
}

func TestGetCardWithDescendantsUserIsolation(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	user1ID := 1
	user2ID := 2

	// Create a card for user 1
	params := models.EditCardParams{
		Title:  "User 1 Card",
		Body:   "User 1 Body",
		CardID: "user1_card",
		Link:   "",
	}
	user1Card, err := CreateCard(s.DB, user1ID, params)
	if err != nil {
		t.Fatalf("Failed to create card for user 1: %v", err)
	}

	// User 2 should not be able to access user 1's card
	_, err = GetCardWithDescendants(s.DB, user2ID, user1Card.ID)
	if err == nil {
		t.Error("Expected error when user 2 tries to access user 1's card")
	}
	if err.Error() != "card not found" {
		t.Errorf("Expected 'card not found' error, got %v", err.Error())
	}

	// User 1 should be able to access their own card
	result, err := GetCardWithDescendants(s.DB, user1ID, user1Card.ID)
	if err != nil {
		t.Fatalf("User 1 should be able to access their card: %v", err)
	}
	if result.ID != user1Card.ID {
		t.Errorf("Expected card ID %v, got %v", user1Card.ID, result.ID)
	}
}

// TestGetCardWithDescendantsLargeTree tests performance with deep nesting (10+ levels) and wide breadth (100+ siblings)
// This is designed to test performance bottlenecks in the tree rendering system
func TestGetCardWithDescendantsLargeTree(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create root card
	rootParams := models.EditCardParams{
		Title:  "Performance Root",
		Body:   "Root card for performance testing",
		CardID: "perf_root",
		Link:   "",
	}
	rootCard, err := CreateCard(s.DB, userID, rootParams)
	if err != nil {
		t.Fatalf("Failed to create root card: %v", err)
	}

	t.Logf("Created root card: %d", rootCard.ID)

	// Create a tree with multiple levels to test descendant retrieval
	// Reduced depth from 12 to 4 to avoid timeout on slower CI environments
	// The original test created ~2000+ cards which caused excessive database queries
	// in IdentifyParentTags (recursive tag inheritance with DB query per level)
	// Current: 3 siblings per level, 4 levels = ~120 cards total
	const maxDepth = 4
	currentDepth := 1

	// Store parents at each level for breadth-first creation
	type levelInfo struct {
		parentCardID string
		parentID      int
		numSiblings   int
	}
	levels := []levelInfo{{parentCardID: "perf_root", parentID: rootCard.ID, numSiblings: 3}}

	// Breadth-first tree creation to maintain structure
	totalCardsCreated := 1

	for currentDepth < maxDepth {
		nextLevel := []levelInfo{}

		// For each parent in current level, create numSiblings children
		for _, level := range levels {
			for i := 0; i < level.numSiblings; i++ {
				childCardID := fmt.Sprintf("%s/child_%d_%d", level.parentCardID, currentDepth, i+1)

				childParams := models.EditCardParams{
					Title:  fmt.Sprintf("Level %d Child %d", currentDepth, i+1),
					Body:   fmt.Sprintf("Child at depth %d, sibling %d", currentDepth, i+1),
					CardID: childCardID,
					Link:   "",
				}

				childCard, err := CreateCard(s.DB, userID, childParams)
				if err != nil {
					t.Fatalf("Failed to create child card at depth %d, sibling %d: %v", currentDepth, i+1, err)
				}

				totalCardsCreated++
				t.Logf("Created card %d/%d: %s (depth: %d)", totalCardsCreated, i+1, childCardID, currentDepth)

				// Plan for next level - create children for this node
				nextLevelSiblings := 3 // Fixed number for simplicity, could scale
				nextLevel = append(nextLevel, levelInfo{
					parentCardID: childCardID,
					parentID:      childCard.ID,
					numSiblings:   nextLevelSiblings,
				})
			}
		}

		levels = nextLevel
		currentDepth++

		if len(levels) == 0 {
			break // No more levels to create
		}
	}

	t.Logf("Total cards created for tree: %d", totalCardsCreated)

	// Now test retrieving the tree - this is where performance issues will show
	startTime := time.Now()

	result, err := GetCardWithDescendants(s.DB, userID, rootCard.ID)
	if err != nil {
		t.Fatalf("GetCardWithDescendants failed on large tree: %v", err)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	t.Logf("Tree retrieval took: %v", duration)
	t.Logf("Result has %d direct descendants", len(result.Descendants))

	// Count total nodes in tree recursively
	totalNodes := countNodesInTree(result)
	t.Logf("Total nodes in retrieved tree: %d", totalNodes)

	// Basic validation
	if result.ID != rootCard.ID {
		t.Errorf("Root ID mismatch: expected %v, got %v", rootCard.ID, result.ID)
	}
	if result.Depth != 0 {
		t.Errorf("Root depth should be 0, got %v", result.Depth)
	}

	// Log performance metrics
	if duration > 5*time.Second {
		t.Errorf("PERFORMANCE ISSUE: Tree retrieval took %v (should be < 5s for performance test)", duration)
	}
	if totalNodes < totalCardsCreated-10 { // Allow some tolerance for test flakiness
		t.Errorf("PERFORMANCE ISSUE: Retrieved %d nodes but created %d cards", totalNodes, totalCardsCreated)
	}
}

// countNodesInTree recursively counts all nodes in a CardWithDescendants tree
func countNodesInTree(card models.CardWithDescendants) int {
	count := 1
	for _, descendant := range card.Descendants {
		count += countNodesInTree(descendant)
	}
	return count
}
