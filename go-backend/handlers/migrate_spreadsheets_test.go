package handlers

import (
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrateCardSpreadsheets_Success tests successful migration of spreadsheet blocks
func TestMigrateCardSpreadsheets_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test user
	user := createTestUser(s, t)

	// Create a test card with spreadsheet blocks in body
	cardBody := "# Test Card\n\nSome text here.\n\n```spreadsheet:budget\n{\n  \"rows\": 3,\n  \"cols\": 3,\n  \"data\": {\n    \"A1\": {\"value\": \"Item\"},\n    \"B1\": {\"value\": \"Cost\"},\n    \"C1\": {\"value\": \"Total\"},\n    \"A2\": {\"value\": \"Apple\"},\n    \"B2\": {\"value\": \"1.50\"},\n    \"C2\": {\"formula\": \"=SUM(B2)\"}\n  }\n}\n```\n\nMore text.\n\n```spreadsheet:inventory\n{\n  \"rows\": 2,\n  \"cols\": 2,\n  \"data\": {\n    \"A1\": {\"value\": \"Item\"},\n    \"B1\": {\"value\": \"Count\"},\n    \"A2\": {\"value\": \"Widget\"},\n    \"B2\": {\"value\": \"42\"}\n  }\n}\n```\n\nFinal text."

	params := models.EditCardParams{
		Title:  "Test Card with Spreadsheets",
		Body:   cardBody,
		CardID: "test-card-migration",
	}
	card, err := services.CreateCard(s.GetDB(), user.ID, params)
	require.NoError(t, err, "Failed to create test card")

	// Run migration directly on the card
	migrated, errors := s.migrateCardSpreadsheets(card)

	assert.Equal(t, 2, migrated, "Expected 2 spreadsheets migrated")
	assert.Empty(t, errors, "Expected no errors")

	// Verify spreadsheets were created in database
	spreadsheets, err := models.GetSpreadsheetsByCardID(s.GetDB(), card.ID, user.ID)
	require.NoError(t, err, "Failed to get spreadsheets")
	assert.Len(t, spreadsheets, 2, "Expected 2 spreadsheets in database")

	// Verify spreadsheet names
	spreadsheetNames := make(map[string]bool)
	for _, sheet := range spreadsheets {
		spreadsheetNames[sheet.Name] = true
	}
	assert.True(t, spreadsheetNames["budget"], "Expected 'budget' spreadsheet")
	assert.True(t, spreadsheetNames["inventory"], "Expected 'inventory' spreadsheet")

	// Verify card body was updated with references
	updatedCard, err := services.GetFullCard(s.GetDB(), user.ID, card.ID)
	require.NoError(t, err, "Failed to get updated card")

	assert.Contains(t, updatedCard.Body, "{{spreadsheet:", "Card body should contain spreadsheet references")
	assert.NotContains(t, updatedCard.Body, "```spreadsheet:budget", "Card body should not contain old spreadsheet block format")
	assert.NotContains(t, updatedCard.Body, "```spreadsheet:inventory", "Card body should not contain old spreadsheet block format")
}

// TestMigrateCardSpreadsheets_InvalidJSON tests handling of invalid JSON in spreadsheet blocks
func TestMigrateCardSpreadsheets_InvalidJSON(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	user := createTestUser(s, t)

	// Create a card with invalid JSON in one spreadsheet block
	cardBody := "# Test Card\n\n```spreadsheet:valid\n{\n  \"rows\": 2,\n  \"cols\": 2,\n  \"data\": {}\n}\n```\n\n```spreadsheet:invalid\n{invalid json here}\n```\n"

	params := models.EditCardParams{
		Title:  "Test Card with Invalid JSON",
		Body:   cardBody,
		CardID: "test-card-invalid-json",
	}
	card, err := services.CreateCard(s.GetDB(), user.ID, params)
	require.NoError(t, err, "Failed to create test card")

	// Run migration
	migrated, errors := s.migrateCardSpreadsheets(card)

	assert.Equal(t, 1, migrated, "Expected 1 spreadsheet migrated (valid one)")
	assert.Len(t, errors, 1, "Expected 1 error for invalid JSON")
	assert.Contains(t, errors[0].Error, "Failed to parse spreadsheet JSON")
}

// TestMigrateCardSpreadsheets_NoSpreadsheets tests handling of cards without spreadsheets
func TestMigrateCardSpreadsheets_NoSpreadsheets(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	user := createTestUser(s, t)

	// Create a card without any spreadsheet blocks
	cardBody := "# Test Card\n\nJust regular text here.\nNo spreadsheet blocks.\n"

	params := models.EditCardParams{
		Title:  "Test Card without Spreadsheets",
		Body:   cardBody,
		CardID: "test-card-no-spreadsheets",
	}
	card, err := services.CreateCard(s.GetDB(), user.ID, params)
	require.NoError(t, err, "Failed to create test card")

	// Run migration
	migrated, errors := s.migrateCardSpreadsheets(card)

	assert.Equal(t, 0, migrated, "Expected 0 spreadsheets migrated")
	assert.Empty(t, errors, "Expected no errors")
}

// TestMigrateCardSpreadsheets_DeletedCard tests that deleted cards are skipped
func TestMigrateCardSpreadsheets_DeletedCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	user := createTestUser(s, t)

	// Create a deleted card with spreadsheet blocks
	cardBody := "```spreadsheet:test\n{\"rows\": 2, \"cols\": 2, \"data\": {}}\n```\n"

	params := models.EditCardParams{
		Title:     "Deleted Test Card",
		Body:      cardBody,
		CardID:    "test-deleted-card",
	}
	card, err := services.CreateCard(s.GetDB(), user.ID, params)
	require.NoError(t, err, "Failed to create test card")

	// Mark card as deleted
	_, err = s.GetDB().Exec("UPDATE cards SET is_deleted = true WHERE id = $1", card.ID)
	require.NoError(t, err, "Failed to mark card as deleted")

	// Re-fetch the card to get the updated is_deleted status (including deleted cards)
	var title string
	var body string
	var isDeleted bool
	err = s.GetDB().QueryRow("SELECT title, body, is_deleted FROM cards WHERE id = $1", card.ID).Scan(&title, &body, &isDeleted)
	require.NoError(t, err, "Failed to re-fetch card")

	updatedCard := card
	updatedCard.Title = title
	updatedCard.Body = body
	updatedCard.IsDeleted = isDeleted

	// Run migration - deleted cards should be skipped
	migrated, errors := s.migrateCardSpreadsheets(updatedCard)

	assert.Equal(t, 0, migrated, "Expected 0 spreadsheets migrated (deleted card)")
	assert.Empty(t, errors, "Expected no errors for deleted card")
}

// Helper function to create a test user with admin privileges
func createTestUser(s *Handler, t *testing.T) *models.User {
	params := models.CreateUserParams{
		Username:        "testadmin",
		Email:           "admin@example.com",
		Password:        "hashedpassword",
		ConfirmPassword:  "hashedpassword",
	}
	userID, err := s.CreateUser(params)
	require.NoError(t, err, "Failed to create test user")

	// Mark user as admin
	_, err = s.GetDB().Exec("UPDATE users SET is_admin = true WHERE id = $1", userID)
	require.NoError(t, err, "Failed to mark user as admin")

	// Get the updated user
	user, err := s.QueryUser(userID)
	require.NoError(t, err, "Failed to query test user")
	return &user
}
