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
