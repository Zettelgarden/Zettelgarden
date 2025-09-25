package services

import (
	"context"
	"go-backend/bootstrap"
	"go-backend/models"
	"log"
	"os"
	"strconv"
)

// upsertCardToTypesense adds or updates a card document in Typesense
func UpsertCardToTypesense(card models.Card) {
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}

	// Fetch tags for this card
	var tags []string
	// For now, we'll use an empty array. In a production system, we'd need database access here
	// or modify the function signature to accept tags as a parameter
	tags = []string{}

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
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

	client := bootstrap.GetTypesenseClient()
	_, err := client.Collection(collectionName).
		Documents().Upsert(context.Background(), doc)
	if err != nil {
		log.Printf("failed to upsert card ID %d: %v", card.ID, err)
	}
}

func deleteCardTypesense(cardPK int) {
	if os.Getenv("ZETTEL_IS_TESTING") == "true" {
		return
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	client := bootstrap.GetTypesenseClient()
	_, err := client.Collection(collectionName).
		Document("card-" + strconv.Itoa(cardPK)).Delete(context.Background())
	if err != nil {
		log.Printf("failed to delete card ID %d: %v", cardPK, err)
	}
}
