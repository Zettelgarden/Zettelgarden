package services

import (
	"database/sql"
	"go-backend/models"
	"go-backend/server"
	"go-backend/tests"
	"testing"
	"time"
)

type Handler struct {
	DB     *sql.DB
	Server *server.Server
}

func setup() *Handler {
	S := tests.Setup()
	s := &Handler{
		DB:     S.DB,
		Server: S,
	}

	return s

}

func TestCreateAuditEvent(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	t.Run("Create audit event for card creation", func(t *testing.T) {
		card := models.Card{
			Title:  "Test Card",
			Body:   "Test Body",
			CardID: "test.1",
			UserID: 1,
		}

		err := CreateAuditEvent(s.DB, 1, 1, "card", "create", nil, card)
		if err != nil {
			t.Errorf("Failed to create audit event: %v", err)
		}

		events, err := GetAuditEvents(s.DB, "card", 1)
		if err != nil {
			t.Errorf("Failed to get audit events: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		if events[0].Action != "create" {
			t.Errorf("Expected action 'create', got %s", events[0].Action)
		}
		if events[0].EntityType != "card" {
			t.Errorf("Expected entity_type 'card', got %s", events[0].EntityType)
		}
		if events[0].Details.CustomData["initial_state"] == nil {
			t.Error("Expected initial_state in custom_data")
		}
	})

	t.Run("Create audit event for card update", func(t *testing.T) {
		oldCard := models.Card{
			Title:  "Old Title",
			Body:   "Old Body",
			CardID: "test.1",
			UserID: 1,
		}
		newCard := models.Card{
			Title:  "New Title",
			Body:   "New Body",
			CardID: "test.1",
			UserID: 1,
		}

		err := CreateAuditEvent(s.DB, 1, 1, "card", "update", oldCard, newCard)
		if err != nil {
			t.Errorf("Failed to create audit event: %v", err)
		}

		events, err := GetAuditEvents(s.DB, "card", 1)
		if err != nil {
			t.Errorf("Failed to get audit events: %v", err)
		}
		if len(events) == 0 {
			t.Error("Expected events, got none")
		}

		lastEvent := events[0]
		if lastEvent.Action != "update" {
			t.Errorf("Expected action 'update', got %s", lastEvent.Action)
		}
		if _, ok := lastEvent.Details.Changes["Title"]; !ok {
			t.Error("Expected Title in changes")
		}
		if _, ok := lastEvent.Details.Changes["Body"]; !ok {
			t.Error("Expected Body in changes")
		}
		if lastEvent.Details.Changes["Title"].From != "Old Title" {
			t.Errorf("Expected old title 'Old Title', got %v", lastEvent.Details.Changes["Title"].From)
		}
		if lastEvent.Details.Changes["Title"].To != "New Title" {
			t.Errorf("Expected new title 'New Title', got %v", lastEvent.Details.Changes["Title"].To)
		}
	})

	t.Run("Test field exclusions", func(t *testing.T) {
		oldCard := models.Card{
			Title:     "Same Title",
			Body:      "Old Body",
			CardID:    "test.1",
			UserID:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		newCard := models.Card{
			Title:     "Same Title",
			Body:      "New Body",
			CardID:    "test.1",
			UserID:    1,
			CreatedAt: time.Now().Add(time.Hour),
			UpdatedAt: time.Now().Add(time.Hour),
		}

		err := CreateAuditEvent(s.DB, 1, 1, "card", "update", oldCard, newCard)
		if err != nil {
			t.Errorf("Failed to create audit event: %v", err)
		}

		events, err := GetAuditEvents(s.DB, "card", 1)
		if err != nil {
			t.Errorf("Failed to get audit events: %v", err)
		}
		if len(events) == 0 {
			t.Error("Expected events, got none")
		}

		lastEvent := events[0]
		// Should only contain Body change, not Title (unchanged) or CreatedAt/UpdatedAt (excluded)
		if _, ok := lastEvent.Details.Changes["Title"]; ok {
			t.Error("Title should not be in changes (unchanged)")
		}
		if _, ok := lastEvent.Details.Changes["CreatedAt"]; ok {
			t.Error("CreatedAt should not be in changes (excluded)")
		}
		if _, ok := lastEvent.Details.Changes["UpdatedAt"]; ok {
			t.Error("UpdatedAt should not be in changes (excluded)")
		}
		if _, ok := lastEvent.Details.Changes["Body"]; !ok {
			t.Error("Expected Body in changes")
		}
	})
}

// func TestAuditEventIntegration(t *testing.T) {
// 	s := setup()
// 	defer tests.Teardown()

// 	t.Run("Full card lifecycle audit", func(t *testing.T) {
// 		// Create a card
// 		params := models.EditCardParams{
// 			Title:  "Test Card",
// 			Body:   "Initial Body",
// 			CardID: "test.1",
// 		}
// 		card, err := s.CreateCard(1, params)
// 		if err != nil {
// 			t.Fatalf("Failed to create card: %v", err)
// 		}

// 		// Update the card
// 		params.Body = "Updated Body"
// 		_, err = s.UpdateCard(1, card.ID, params)
// 		if err != nil {
// 			t.Fatalf("Failed to update card: %v", err)
// 		}

// 		// Delete the card
// 		err = s.DeleteCard(1, card.ID)
// 		if err != nil {
// 			t.Fatalf("Failed to delete card: %v", err)
// 		}

// 		// Verify audit trail
// 		events, err := s.GetAuditEvents("card", card.ID)
// 		if err != nil {
// 			t.Fatalf("Failed to get audit events: %v", err)
// 		}
// 		if len(events) != 3 {
// 			t.Errorf("Expected 3 events, got %d", len(events))
// 		}

// 		// Verify event order and types
// 		if events[0].Action != "delete" {
// 			t.Errorf("Expected first event to be delete, got %s", events[0].Action)
// 		}
// 		if events[1].Action != "update" {
// 			t.Errorf("Expected second event to be update, got %s", events[1].Action)
// 		}
// 		if events[2].Action != "create" {
// 			t.Errorf("Expected third event to be create, got %s", events[2].Action)
// 		}

// 		// Verify update content
// 		updateEvent := events[1]
// 		if _, ok := updateEvent.Details.Changes["Body"]; !ok {
// 			t.Error("Expected Body in changes")
// 		}
// 		if updateEvent.Details.Changes["Body"].From != "Initial Body" {
// 			t.Errorf("Expected old body 'Initial Body', got %v", updateEvent.Details.Changes["Body"].From)
// 		}
// 		if updateEvent.Details.Changes["Body"].To != "Updated Body" {
// 			t.Errorf("Expected new body 'Updated Body', got %v", updateEvent.Details.Changes["Body"].To)
// 		}
// 	})
// }
