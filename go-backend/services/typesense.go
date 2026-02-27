package services

import (
	"context"
	"go-backend/bootstrap"
	"go-backend/models"
	"go-backend/pkg/config"
	"log"
	"os"
	"strconv"
)

// upsertCardToTypesense adds or updates a card document in Typesense
func UpsertCardToTypesense(db models.Database, card models.Card) {
	cfg := config.GetConfig()
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}
	if cfg == nil || cfg.Server.DevMode {
		return
	}

	// Fetch tags for this card
	cardTags, err := QueryTagsForCard(db, card.UserID, card.ID)
	if err != nil {
		log.Printf("failed to fetch tags for card ID %d: %v", card.ID, err)
		cardTags = []models.Tag{}
	}

	// Convert tags to string array for Typesense
	tags := make([]string, 0)
	for _, tag := range cardTags {
		tags = append(tags, tag.Name)
	}

	collectionName := cfg.Services.Search.Collection
	doc := map[string]interface{}{
		"id":                    "card-" + strconv.Itoa(card.ID),
		"fact_pk":               -1,
		"card_id":               card.CardID,
		"card_pk":               card.ID,
		"entity_pk":             -1,
		"user_id":               card.UserID,
		"type":                  "card",
		"title":                 card.Title,
		"preview":               card.Body,
		"parent_id":             card.ParentID,
		"created_at":            card.CreatedAt.Unix(),
		"updated_at":            card.UpdatedAt.Unix(),
		"linked_card_id":        "",
		"linked_card_pk":        -1,
		"linked_card_title":     "",
		"linked_card_parent_id": -1,
		"tags":                  tags,
	}

	client := bootstrap.GetTypesenseClient(cfg.Services.Search)
	_, err = client.Collection(collectionName).
		Documents().Upsert(context.Background(), doc)
	if err != nil {
		log.Printf("failed to upsert card ID %d: %v", card.ID, err)
	}
}

func deleteCardTypesense(cardPK int) {
	cfg := config.GetConfig()
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}
	if cfg == nil || cfg.Server.DevMode {
		return
	}
	collectionName := cfg.Services.Search.Collection
	client := bootstrap.GetTypesenseClient(cfg.Services.Search)
	_, err := client.Collection(collectionName).
		Document("card-" + strconv.Itoa(cardPK)).Delete(context.Background())
	if err != nil {
		log.Printf("failed to delete card ID %d: %v", cardPK, err)
	}
}

// UpsertEmailToTypesense adds or updates an email document in Typesense
func UpsertEmailToTypesense(db models.Database, email models.Email) {
	cfg := config.GetConfig()
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}
	if cfg == nil || cfg.Server.DevMode {
		return
	}

	collectionName := cfg.Services.Search.Collection

	// Build a searchable text from email body
	bodyText := ""
	if email.BodyText != nil {
		bodyText = *email.BodyText
	} else if email.BodyHTML != nil {
		// Strip HTML tags for searching
		bodyText = *email.BodyHTML
	}

	// Build sender display name
	sender := ""
	if email.FromName != nil && *email.FromName != "" {
		sender = *email.FromName + " <" + *email.FromAddress + ">"
	} else if email.FromAddress != nil {
		sender = *email.FromAddress
	}

	// Handle received_at timestamp
	var receivedAt int64
	if email.ReceivedAt != nil {
		receivedAt = email.ReceivedAt.Unix()
	} else {
		receivedAt = email.CreatedAt.Unix()
	}

	// Handle nullable fields for Typesense
	var subject interface{} = nil
	if email.Subject != nil {
		subject = *email.Subject
	}
	var fromAddress interface{} = nil
	if email.FromAddress != nil {
		fromAddress = *email.FromAddress
	}
	var folder interface{} = nil
	if email.Folder != nil {
		folder = *email.Folder
	}

	doc := map[string]interface{}{
		"id":                    "email-" + strconv.Itoa(email.ID),
		"fact_pk":               -1,
		"card_id":               "",
		"card_pk":               -1,
		"entity_pk":             -1,
		"email_pk":              email.ID,
		"user_id":               email.UserID,
		"type":                  "email",
		"title":                 subject,
		"preview":               bodyText,
		"parent_id":             -1,
		"created_at":            email.CreatedAt.Unix(),
		"updated_at":            email.UpdatedAt.Unix(),
		"linked_card_id":        "",
		"linked_card_pk":        -1,
		"linked_card_title":     "",
		"linked_card_parent_id": -1,
		"tags":                  []string{},
		"email_id":              email.ID,
		"email_sender":          sender,
		"email_from_address":    fromAddress,
		"email_subject":         subject,
		"email_received_at":     receivedAt,
		"email_status":          email.Status,
		"email_folder":          folder,
		"email_is_read":         email.IsRead,
	}

	client := bootstrap.GetTypesenseClient(cfg.Services.Search)
	_, err := client.Collection(collectionName).
		Documents().Upsert(context.Background(), doc)
	if err != nil {
		log.Printf("failed to upsert email ID %d: %v", email.ID, err)
	}
}

// DeleteEmailFromTypesense removes an email document from Typesense
func DeleteEmailFromTypesense(emailPK int) {
	cfg := config.GetConfig()
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}
	if cfg == nil || cfg.Server.DevMode {
		return
	}
	collectionName := cfg.Services.Search.Collection
	client := bootstrap.GetTypesenseClient(cfg.Services.Search)
	_, err := client.Collection(collectionName).
		Document("email-" + strconv.Itoa(emailPK)).Delete(context.Background())
	if err != nil {
		log.Printf("failed to delete email ID %d: %v", emailPK, err)
	}
}
