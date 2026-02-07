package entity

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"time"

	"github.com/lib/pq"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// UpdateEntityParams holds the parameters for updating an entity
type UpdateEntityParams struct {
	Name        string
	Description string
	Type        string
	CardPK      *int
}

// EntityListResponse represents the paginated response for entity queries
type EntityListResponse struct {
	Entities   []models.Entity `json:"entities"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// GetEntityByName retrieves a specific entity by its name for a given user
func GetEntityByName(db *sql.DB, userID int, entityName string) (models.Entity, error) {
	query := `
        SELECT
            e.id,
            e.user_id,
            e.name,
            e.description,
            e.type,
            e.created_at,
            e.updated_at,
            e.card_pk,
            COUNT(DISTINCT ecj.card_pk) as card_count,
            c.id as linked_card_id,
            c.card_id as linked_card_card_id,
            c.title as linked_card_title,
            c.user_id as linked_card_user_id,
            c.parent_id as linked_card_parent_id,
            c.created_at as linked_card_created_at,
            c.updated_at as linked_card_updated_at
        FROM
            entities e
            LEFT JOIN entity_card_junction ecj ON e.id = ecj.entity_id
            LEFT JOIN cards c ON e.card_pk = c.id AND c.is_deleted = FALSE
        WHERE
            e.user_id = $1 AND e.name = $2
        GROUP BY
            e.id, e.user_id, e.name, e.description, e.type, e.created_at, e.updated_at, e.card_pk,
            c.id, c.card_id, c.title, c.user_id, c.parent_id, c.created_at, c.updated_at
    `

	var entity models.Entity
	var cardID sql.NullInt64
	var cardCardID, cardTitle sql.NullString
	var cardUserID, cardParentID sql.NullInt64
	var cardCreatedAt, cardUpdatedAt sql.NullTime

	err := db.QueryRow(query, userID, entityName).Scan(
		&entity.ID,
		&entity.UserID,
		&entity.Name,
		&entity.Description,
		&entity.Type,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&entity.CardPK,
		&entity.CardCount,
		&cardID,
		&cardCardID,
		&cardTitle,
		&cardUserID,
		&cardParentID,
		&cardCreatedAt,
		&cardUpdatedAt,
	)
	if err != nil {
		return models.Entity{}, err
	}

	if cardID.Valid {
		entity.Card = &models.PartialCard{
			ID:        int(cardID.Int64),
			CardID:    cardCardID.String,
			Title:     cardTitle.String,
			UserID:    int(cardUserID.Int64),
			ParentID:  int(cardParentID.Int64),
			CreatedAt: cardCreatedAt.Time,
			UpdatedAt: cardUpdatedAt.Time,
			Tags:      []models.Tag{},
		}
	}

	return entity, nil
}

// SearchEntities searches for entities using Typesense only
func SearchEntities(db *sql.DB, typesenseClient *typesense.Client, userID int, query string, limit int) ([]models.Entity, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		return nil, fmt.Errorf("TYPESENSE_COLLECTION environment variable not set")
	}

	// Build filter for entities only
	filter := fmt.Sprintf("user_id:=%d && type:=entity", userID)

	searchParams := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "title,preview", // Search in name and description
		FilterBy: &filter,
		PerPage:  &limit,
	}

	searchResult, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), searchParams)
	if err != nil {
		return nil, fmt.Errorf("typesense search error: %w", err)
	}

	var entityIDs []int
	var entityMap = make(map[int]*models.Entity)

	// Extract entity data from Typesense results
	if searchResult.Hits != nil {
		for _, hit := range *searchResult.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				if entityPK, ok := doc["entity_pk"].(float64); ok {
					entityID := int(entityPK)
					entityIDs = append(entityIDs, entityID)

					entity := &models.Entity{
						ID:          entityID,
						UserID:      userID,
						Name:        doc["title"].(string),
						Description: doc["preview"].(string),
						Type:        "entity",
						CreatedAt:   time.Unix(int64(doc["created_at"].(float64)), 0),
						UpdatedAt:   time.Unix(int64(doc["updated_at"].(float64)), 0),
						CardCount:   0, // Will be filled in below
					}

					// Handle linked card data if available
					if linkedCardPK, ok := doc["linked_card_pk"].(float64); ok && linkedCardPK > 0 {
						entity.CardPK = new(int)
						*entity.CardPK = int(linkedCardPK)

						if linkedCardID, ok := doc["linked_card_id"].(string); ok && linkedCardID != "" {
							entity.Card = &models.PartialCard{
								ID:        int(linkedCardPK),
								CardID:    linkedCardID,
								Title:     doc["linked_card_title"].(string),
								UserID:    userID,
								ParentID:  int(doc["linked_card_parent_id"].(float64)),
								CreatedAt: entity.CreatedAt,
								UpdatedAt: entity.UpdatedAt,
								Tags:      []models.Tag{},
							}
						}
					}

					entityMap[entityID] = entity
				}
			}
		}
	}

	// Get card counts from database in a single query if we have entities
	if len(entityIDs) > 0 {
		cardCountQuery := `
			SELECT entity_id, COUNT(DISTINCT card_pk) as card_count
			FROM entity_card_junction
			WHERE entity_id = ANY($1) AND user_id = $2
			GROUP BY entity_id
		`

		rows, err := db.Query(cardCountQuery, pq.Array(entityIDs), userID)
		if err != nil {
			log.Printf("error querying entity card counts: %v", err)
			// Continue without card counts rather than failing completely
		} else {
			defer rows.Close()
			for rows.Next() {
				var entityID int
				var cardCount int
				if err := rows.Scan(&entityID, &cardCount); err != nil {
					log.Printf("error scanning card count: %v", err)
					continue
				}
				if entity, exists := entityMap[entityID]; exists {
					entity.CardCount = cardCount
				}
			}
		}
	}

	// Convert map to slice, maintaining the order from Typesense
	var entities []models.Entity
	for _, entityID := range entityIDs {
		if entity, exists := entityMap[entityID]; exists {
			entities = append(entities, *entity)
		}
	}

	return entities, nil
}

// GetCardsByEntity retrieves all cards linked to a specific entity
func GetCardsByEntity(db models.Database, userID int, entityID int) ([]models.PartialCard, error) {
	rows, err := db.Query(`
		SELECT c.id, c.card_id, c.user_id, c.title, c.parent_id, c.created_at, c.updated_at
		FROM cards c
		JOIN entity_card_junction ecj ON c.id = ecj.card_pk
		WHERE ecj.entity_id = $1 AND ecj.user_id = $2 AND c.is_deleted = FALSE
		ORDER BY c.updated_at DESC
	`, entityID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var c models.PartialCard
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// GetEntityByID retrieves a specific entity by its ID for a given user
func GetEntityByID(db *sql.DB, userID int, entityID int) (models.Entity, error) {
	query := `
        SELECT
            e.id,
            e.user_id,
            e.name,
            e.description,
            e.type,
            e.created_at,
            e.updated_at,
            e.card_pk,
            COUNT(DISTINCT ecj.card_pk) as card_count,
            c.id as linked_card_id,
            c.card_id as linked_card_card_id,
            c.title as linked_card_title,
            c.user_id as linked_card_user_id,
            c.parent_id as linked_card_parent_id,
            c.created_at as linked_card_created_at,
            c.updated_at as linked_card_updated_at
        FROM
            entities e
            LEFT JOIN entity_card_junction ecj ON e.id = ecj.entity_id
            LEFT JOIN cards c ON e.card_pk = c.id AND c.is_deleted = FALSE
        WHERE
            e.user_id = $1 AND e.id = $2
        GROUP BY
            e.id, e.user_id, e.name, e.description, e.type, e.created_at, e.updated_at, e.card_pk,
            c.id, c.card_id, c.title, c.user_id, c.parent_id, c.created_at, c.updated_at
    `

	var entity models.Entity
	var cardID sql.NullInt64
	var cardCardID, cardTitle sql.NullString
	var cardUserID, cardParentID sql.NullInt64
	var cardCreatedAt, cardUpdatedAt sql.NullTime

	err := db.QueryRow(query, userID, entityID).Scan(
		&entity.ID,
		&entity.UserID,
		&entity.Name,
		&entity.Description,
		&entity.Type,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&entity.CardPK,
		&entity.CardCount,
		&cardID,
		&cardCardID,
		&cardTitle,
		&cardUserID,
		&cardParentID,
		&cardCreatedAt,
		&cardUpdatedAt,
	)
	if err != nil {
		return models.Entity{}, err
	}

	if cardID.Valid {
		entity.Card = &models.PartialCard{
			ID:        int(cardID.Int64),
			CardID:    cardCardID.String,
			Title:     cardTitle.String,
			UserID:    int(cardUserID.Int64),
			ParentID:  int(cardParentID.Int64),
			CreatedAt: cardCreatedAt.Time,
			UpdatedAt: cardUpdatedAt.Time,
			Tags:      []models.Tag{},
		}
	}

	return entity, nil
}

// MergeEntities merges two entities, combining their relationships
func MergeEntities(db *sql.DB, userID int, entity1ID int, entity2ID int) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify both entities exist and belong to the user
	var entity1, entity2 models.Entity
	err = tx.QueryRow(`
		SELECT id, user_id, name, description, type, card_pk
		FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity1ID, userID).Scan(
		&entity1.ID, &entity1.UserID, &entity1.Name,
		&entity1.Description, &entity1.Type, &entity1.CardPK)
	if err != nil {
		return fmt.Errorf("failed to find entity1: %w", err)
	}

	err = tx.QueryRow(`
		SELECT id, user_id, name, description, type, card_pk
		FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity2ID, userID).Scan(
		&entity2.ID, &entity2.UserID, &entity2.Name,
		&entity2.Description, &entity2.Type, &entity2.CardPK)
	if err != nil {
		return fmt.Errorf("failed to find entity2: %w", err)
	}

	// Move all card relationships from entity2 to entity1
	_, err = tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		SELECT user_id, $1, card_pk
		FROM entity_card_junction
		WHERE entity_id = $2
		ON CONFLICT (entity_id, card_pk) DO NOTHING`,
		entity1.ID, entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to merge card relationships: %w", err)
	}

	// Move all fact relationships from entity2 to entity1
	_, err = tx.Exec(`
		INSERT INTO entity_fact_junction (user_id, entity_id, fact_id)
		SELECT user_id, $1, fact_id
		FROM entity_fact_junction
		WHERE entity_id = $2
		ON CONFLICT (entity_id, fact_id) DO NOTHING`,
		entity1.ID, entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to merge fact relationships: %w", err)
	}

	// Delete entity2's relationships
	_, err = tx.Exec(`
		DELETE FROM entity_card_junction
		WHERE entity_id = $1`,
		entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2 card relationships: %w", err)
	}

	_, err = tx.Exec(`
		DELETE FROM entity_fact_junction
		WHERE entity_id = $1`,
		entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2 fact relationships: %w", err)
	}

	// Preserve card_pk from either entity (prefer entity1, fallback to entity2)
	cardPK := entity1.CardPK
	if cardPK == nil {
		cardPK = entity2.CardPK
	}

	// Update entity1 description to indicate merge
	newDescription := fmt.Sprintf("%s (merged with %s)", entity1.Description, entity2.Name)
	if len(newDescription) > 2000 {
		newDescription = entity1.Description
	}

	_, err = tx.Exec(`UPDATE entities SET description = $1, card_pk = $2 WHERE id = $3`,
		newDescription, cardPK, entity1.ID)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	// Delete entity2
	_, err = tx.Exec(`
		DELETE FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity2.ID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateEntity updates an entity in the database
func UpdateEntity(db *sql.DB, userID int, entityID int, params UpdateEntityParams) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify entity exists and belongs to user
	var exists bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM entities
			WHERE id = $1 AND user_id = $2
		)`,
		entityID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check entity existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("entity not found or does not belong to user")
	}

	// Check if name is unique for this user
	var nameExists bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM entities
			WHERE user_id = $1 AND name = $2 AND id != $3
		)`,
		userID, params.Name, entityID).Scan(&nameExists)
	if err != nil {
		return fmt.Errorf("failed to check name uniqueness: %w", err)
	}
	if nameExists {
		return fmt.Errorf("an entity with this name already exists")
	}

	// Update the entity
	_, err = tx.Exec(`
		UPDATE entities
		SET name = $1,
		    description = $2,
		    type = $3,
		    card_pk = $4,
		    updated_at = NOW()
		WHERE id = $5 AND user_id = $6`,
		params.Name, params.Description, params.Type, params.CardPK, entityID, userID)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteEntity deletes an entity and all its relationships
func DeleteEntity(db *sql.DB, userID int, entityID int) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify entity exists and belongs to the user
	var exists bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM entities
			WHERE id = $1 AND user_id = $2
		)`,
		entityID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check entity existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("entity not found or does not belong to user")
	}

	// Delete entity-card relationships first
	_, err = tx.Exec(`
		DELETE FROM entity_card_junction
		WHERE entity_id = $1 AND user_id = $2`,
		entityID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete entity relationships: %w", err)
	}

	// Delete the entity
	_, err = tx.Exec(`
		DELETE FROM entities
		WHERE id = $1 AND user_id = $2`,
		entityID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AddEntityToCard links an entity to a card
func AddEntityToCard(db *sql.DB, userID int, entityID int, cardPK int) error {
	// Verify entity exists and belongs to user
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM entities
			WHERE id = $1 AND user_id = $2
		)`,
		entityID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify entity: %w", err)
	}
	if !exists {
		return fmt.Errorf("entity not found or does not belong to user")
	}

	// Verify card exists and belongs to user
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM cards
			WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
		)`,
		cardPK, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify card: %w", err)
	}
	if !exists {
		return fmt.Errorf("card not found or access denied")
	}

	// Add the entity-card relationship
	_, err = db.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
		ON CONFLICT (entity_id, card_pk) DO NOTHING
	`, userID, entityID, cardPK)
	if err != nil {
		return fmt.Errorf("failed to add entity to card: %w", err)
	}

	return nil
}

// RemoveEntityFromCard removes the link between an entity and a card
func RemoveEntityFromCard(db *sql.DB, userID int, entityID int, cardPK int) error {
	// Delete the entity-card relationship
	_, err := db.Exec(`
		DELETE FROM entity_card_junction
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, entityID, cardPK, userID)
	if err != nil {
		return fmt.Errorf("failed to remove entity from card: %w", err)
	}

	return nil
}

// FindSimilarEntities finds entities similar to a given entity using Typesense
func FindSimilarEntities(db *sql.DB, typesenseClient *typesense.Client, userID int, entityID int, limit int) ([]map[string]interface{}, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// Get the entity to find similar entities for
	entity, err := GetEntityByID(db, userID, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		return nil, fmt.Errorf("TYPESENSE_COLLECTION environment variable not set")
	}

	// Build search query from entity name and description
	searchQuery := fmt.Sprintf("%s %s", entity.Name, entity.Description)

	// Filter for entities only, excluding the current entity
	filter := fmt.Sprintf("user_id:=%d && type:=entity && entity_pk:!=%d", userID, entityID)

	searchParams := &api.SearchCollectionParams{
		Q:        searchQuery,
		QueryBy:  "title,preview",
		FilterBy: &filter,
		PerPage:  &limit,
	}

	searchResult, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), searchParams)
	if err != nil {
		return nil, fmt.Errorf("typesense search error: %w", err)
	}

	var results []map[string]interface{}

	if searchResult.Hits != nil {
		for _, hit := range *searchResult.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				if entityPK, ok := doc["entity_pk"].(float64); ok {
					similarityScore := 0.0
					if hit.Highlight != nil {
						// Use a simple scoring mechanism - more highlights = higher similarity
						similarityScore = float64(len(*hit.Highlight))
					}

					result := map[string]interface{}{
						"id":          int(entityPK),
						"user_id":     userID,
						"name":        doc["title"].(string),
						"description": doc["preview"].(string),
						"type":        "entity",
						"created_at":  time.Unix(int64(doc["created_at"].(float64)), 0),
						"updated_at":  time.Unix(int64(doc["updated_at"].(float64)), 0),
						"score":       similarityScore,
					}

					// Handle linked card data if available
					if linkedCardPK, ok := doc["linked_card_pk"].(float64); ok && linkedCardPK > 0 {
						result["card_pk"] = int(linkedCardPK)
						if linkedCardID, ok := doc["linked_card_id"].(string); ok && linkedCardID != "" {
							result["linked_card"] = map[string]interface{}{
								"id":        int(linkedCardPK),
								"card_id":   linkedCardID,
								"title":     doc["linked_card_title"].(string),
								"user_id":   userID,
								"parent_id": int(doc["linked_card_parent_id"].(float64)),
							}
						}
					}

					results = append(results, result)
				}
			}
		}
	}

	return results, nil
}
