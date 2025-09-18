package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

func GetChildCards(db *sql.DB, userID int, cardID int) ([]map[string]interface{}, error) {
	// Get the parent card's card_id first
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", cardID, userID).Scan(&parentCardID)
	if err != nil {
		return nil, err
	}

	// Find child cards based on card_id hierarchy
	query := `
		SELECT id, title, LEFT(body, 200) as body_preview, card_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND card_id LIKE $2 AND card_id != $3
		ORDER BY card_id
		LIMIT 50
	`

	pattern := parentCardID + "%"
	rows, err := db.Query(query, userID, pattern, parentCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []map[string]interface{}
	for rows.Next() {
		var card map[string]interface{} = make(map[string]interface{})
		var title, bodyPreview, cardIDStr sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &bodyPreview, &cardIDStr, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body_preview"] = bodyPreview.String + "..."
		card["card_id"] = cardIDStr.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

func GetParentCard(db *sql.DB, userID int, cardID int) ([]map[string]interface{}, error) {
	// Get the card's card_id first
	var currentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", cardID, userID).Scan(&currentCardID)
	if err != nil {
		return nil, err
	}

	// Find parent by removing last segment
	parts := strings.Split(currentCardID, "/")
	if len(parts) <= 1 {
		return []map[string]interface{}{}, nil // No parent (root card)
	}

	parentCardID := strings.Join(parts[:len(parts)-1], "/")

	query := `
		SELECT id, title, LEFT(body, 200) as body_preview, card_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND card_id = $2
	`

	var card map[string]interface{} = make(map[string]interface{})
	var title, bodyPreview, cardIDStr sql.NullString
	var id int
	var createdAt, updatedAt time.Time

	err = db.QueryRow(query, userID, parentCardID).Scan(&id, &title, &bodyPreview, &cardIDStr, &createdAt, &updatedAt)
	if err != nil {
		return []map[string]interface{}{}, nil // Parent not found
	}

	card["id"] = id
	card["title"] = title.String
	card["body_preview"] = bodyPreview.String + "..."
	card["card_id"] = cardIDStr.String
	card["created_at"] = createdAt
	card["updated_at"] = updatedAt

	return []map[string]interface{}{card}, nil
}

func ExecuteTextSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	// Use Typesense for text search
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=card"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "card_id, title, preview",
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense search error: %v", err)
		// Fallback to SQL search if Typesense fails
		return executeTextSearchFallback(db, userID, query, limit)
	}

	var cards []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "card" {
				card := map[string]interface{}{
					"id":           int(doc["card_pk"].(float64)),
					"title":        doc["title"].(string),
					"body_preview": doc["preview"].(string),
					"card_id":      doc["card_id"].(string),
					"created_at":   time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at":   time.Unix(int64(doc["updated_at"].(float64)), 0),
				}
				cards = append(cards, card)
			}
		}
	}

	return cards, nil
}

// executeTextSearchFallback provides SQL-based fallback when Typesense is unavailable
func executeTextSearchFallback(db *sql.DB, userID int, query string, limit int) ([]map[string]interface{}, error) {
	searchQuery := `
		SELECT id, title, LEFT(body, 200) as body_preview, created_at, updated_at, card_id
		FROM cards
		WHERE user_id = $1 AND (
			title ILIKE $2 OR
			body ILIKE $2 OR
			card_id ILIKE $2
		)
		ORDER BY
			CASE WHEN title ILIKE $2 THEN 1 ELSE 2 END,
			updated_at DESC
		LIMIT $3
	`

	searchPattern := "%" + query + "%"
	rows, err := db.Query(searchQuery, userID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []map[string]interface{}
	for rows.Next() {
		var card map[string]interface{} = make(map[string]interface{})
		var title, bodyPreview, cardID sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &bodyPreview, &createdAt, &updatedAt, &cardID)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body_preview"] = bodyPreview.String + "..."
		card["card_id"] = cardID.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

func ExecuteSemanticSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	// Use Typesense with embedding search for semantic search
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=card"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "card_id, title, embedding", // Include embedding for semantic search
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense semantic search error: %v", err)
		// Fallback to text search if semantic search fails
		return ExecuteTextSearch(db, userID, query, limit, typesenseClient)
	}

	var cards []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "card" {
				card := map[string]interface{}{
					"id":           int(doc["card_pk"].(float64)),
					"title":        doc["title"].(string),
					"body_preview": doc["preview"].(string),
					"card_id":      doc["card_id"].(string),
					"created_at":   time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at":   time.Unix(int64(doc["updated_at"].(float64)), 0),
				}
				cards = append(cards, card)
			}
		}
	}

	return cards, nil
}

func GetFullCard(db *sql.DB, userID int, cardPK int) (models.Card, error) {
	var card models.Card

	err := db.QueryRow(`
	SELECT 
	id, card_id, user_id, title, body, link, parent_id,
        created_at, updated_at
	FROM 
	cards
	WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, cardPK, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.Body,
		&card.Link,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Card{}, fmt.Errorf("card not found")
		}
		log.Printf("query error: %v", err)
		return models.Card{}, fmt.Errorf("unable to access card")
	}

	return card, nil
}
