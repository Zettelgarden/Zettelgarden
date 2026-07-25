package services

import (
	"context"
	"database/sql"
	"go-backend/models"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

func GetCardFacts(db *sql.DB, userID int, cardPK int) ([]models.Fact, error) {
	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.card_pk, f.fact, f.created_at, f.updated_at
		FROM facts f
		JOIN fact_card_junction fcj ON f.id = fcj.fact_id
		WHERE fcj.card_pk = $1 AND fcj.user_id = $2
		ORDER BY f.created_at DESC
	`, cardPK, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		if err := rows.Scan(&f.ID, &f.UserID, &f.CardPK, &f.Fact, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, f)
	}
	return facts, nil
}

func GetEntityFacts(db *sql.DB, userID int, entityID int) ([]models.Fact, error) {
	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.card_pk, f.fact, f.created_at, f.updated_at
		FROM facts f
		JOIN entity_fact_junction efj ON f.id = efj.fact_id
		WHERE efj.entity_id = $1 AND efj.user_id = $2
		ORDER BY f.created_at DESC
	`, entityID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		if err := rows.Scan(&f.ID, &f.UserID, &f.CardPK, &f.Fact, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, f)
	}
	return facts, nil
}

func GetFactCards(db *sql.DB, userID int, factID int) ([]models.PartialCard, error) {
	rows, err := db.Query(`
		SELECT c.id, c.card_id, c.user_id, c.title, c.parent_id, c.created_at, c.updated_at
		FROM cards c
		JOIN fact_card_junction fcj ON c.id = fcj.card_pk
		WHERE fcj.fact_id = $1 AND fcj.user_id = $2
	`, factID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var c models.PartialCard
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &parentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = new(int)
			*c.ParentID = int(parentID.Int64)
		}
		cards = append(cards, c)
	}
	return cards, nil
}

func ExecuteFactTextSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=fact"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "title",
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense fact text search error: %v", err)
		return executeFactTextSearchFallback(db, userID, query, limit)
	}

	var facts []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "fact" {
				fact := map[string]interface{}{
					"id":                    int(doc["fact_pk"].(float64)),
					"fact":                  doc["title"].(string),
					"created_at":            time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at":            time.Unix(int64(doc["updated_at"].(float64)), 0),
					"linked_card_id":        doc["linked_card_id"].(string),
					"linked_card_pk":        int(doc["linked_card_pk"].(float64)),
					"linked_card_title":     doc["linked_card_title"].(string),
					"linked_card_parent_id": int(doc["linked_card_parent_id"].(float64)),
				}
				facts = append(facts, fact)
			}
		}
	}

	return facts, nil
}

func executeFactTextSearchFallback(db *sql.DB, userID int, query string, limit int) ([]map[string]interface{}, error) {
	searchQuery := `
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.title, c.parent_id
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE f.user_id = $1 AND f.fact LIKE $2
		ORDER BY f.updated_at DESC
		LIMIT $3
	`

	searchPattern := "%" + query + "%"
	rows, err := db.Query(searchQuery, userID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []map[string]interface{}
	for rows.Next() {
		fact := make(map[string]interface{})
		var id, cardPK int
		var cardParentID sql.NullInt64
		var factText, cardID, cardTitle string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &factText, &createdAt, &updatedAt, &cardPK, &cardID, &cardTitle, &cardParentID)
		if err != nil {
			continue
		}

		fact["id"] = id
		fact["fact"] = factText
		fact["created_at"] = createdAt
		fact["updated_at"] = updatedAt
		fact["linked_card_id"] = cardID
		fact["linked_card_pk"] = cardPK
		fact["linked_card_title"] = cardTitle
		if cardParentID.Valid {
			fact["linked_card_parent_id"] = int(cardParentID.Int64)
		} else {
			fact["linked_card_parent_id"] = nil
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

func ExecuteFactSemanticSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=fact"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "title, embedding",
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense fact semantic search error: %v", err)
		return ExecuteFactTextSearch(db, userID, query, limit, typesenseClient)
	}

	var facts []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if docType, ok := doc["type"].(string); ok && docType == "fact" {
				fact := map[string]interface{}{
					"id":         int(doc["fact_pk"].(float64)),
					"fact":       doc["title"].(string),
					"created_at": time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at": time.Unix(int64(doc["updated_at"].(float64)), 0),
				}

				if linkedCardID, ok := doc["linked_card_id"].(string); ok {
					fact["linked_card_id"] = linkedCardID
				}
				if linkedCardPK, ok := doc["linked_card_pk"].(float64); ok {
					fact["linked_card_pk"] = int(linkedCardPK)
				}
				if linkedCardTitle, ok := doc["linked_card_title"].(string); ok {
					fact["linked_card_title"] = linkedCardTitle
				}
				if linkedCardParentID, ok := doc["linked_card_parent_id"].(float64); ok {
					fact["linked_card_parent_id"] = int(linkedCardParentID)
				}

				facts = append(facts, fact)
			}
		}
	}

	if len(facts) == 0 {
		log.Printf("No facts found in semantic search, trying text search fallback")
		return ExecuteFactTextSearch(db, userID, query, limit, typesenseClient)
	}

	return facts, nil
}
