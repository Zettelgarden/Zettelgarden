// Package fact provides fact-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The fact domain contains tools for managing and searching facts.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// The fact domain package demonstrates the pattern for splitting tools into
// separate domain packages. The registration is handled in services/fact_tools.go
// to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions (GetCardFacts, GetEntityFacts, GetFactCards, etc.)
// 2. Domain-specific business logic for fact search
// 3. Fact-card and fact-entity relationship management
package fact

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"go-backend/models"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// PartialCard represents a minimal card structure for fact-card relationships.
type PartialCard struct {
	ID        int
	CardID    string
	UserID    int
	Title     string
	ParentID  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetCardFacts retrieves all facts associated with a specific card.
// This is the domain data access function for fact-card relationships.
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

// GetEntityFacts retrieves all facts linked to a specific entity.
// This is the domain data access function for fact-entity relationships.
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

// GetFactCards retrieves all cards that are linked to a specific fact.
// This is the domain data access function for fact-card relationships.
func GetFactCards(db *sql.DB, userID int, factID int) ([]PartialCard, error) {
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

	var cards []PartialCard
	for rows.Next() {
		var c PartialCard
		// Handle nullable parent_id
		var parentID sql.NullInt32
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &parentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = int(parentID.Int32)
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// ExecuteFactTextSearch performs text-based search for facts.
// This is the domain business logic for fact text search.
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

// executeFactTextSearchFallback is the fallback SQL-based text search when Typesense fails.
func executeFactTextSearchFallback(db *sql.DB, userID int, query string, limit int) ([]map[string]interface{}, error) {
	searchQuery := `
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.title, c.parent_id
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE f.user_id = $1 AND f.fact ILIKE $2
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
		var id, cardPK, cardParentID int
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
		fact["linked_card_parent_id"] = cardParentID

		facts = append(facts, fact)
	}

	return facts, nil
}

// ExecuteFactSemanticSearch performs semantic similarity search for facts using embeddings.
// This is the domain business logic for fact semantic search.
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
