package handlers

import (
	"go-backend/models"
	"go-backend/services"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a test card for audit testing
func createTestCard(t *testing.T, h *Handler, userID int) models.Card {
	params := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "test-1",
	}
	card, err := services.CreateCard(h.DB, userID, params)
	assert.NoError(t, err)
	return card
}

// Helper function to verify audit event
func verifyAuditEvent(t *testing.T, event models.AuditEvent, expectedUserID int, expectedEntityID int, expectedEntityType string, expectedAction string) {
	assert.Equal(t, expectedUserID, event.UserID)
	assert.Equal(t, expectedEntityID, event.EntityID)
	assert.Equal(t, expectedEntityType, event.EntityType)
	assert.Equal(t, expectedAction, event.Action)
	assert.NotNil(t, event.Details)
	assert.False(t, event.CreatedAt.IsZero())
}
