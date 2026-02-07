package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

const SIMILARITY_THRESHOLD = 0.15

// EntityWithScore extends Entity with similarity score
type EntityWithScore struct {
	models.Entity
	Score float64 `json:"score"`
}

func validateEntityName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("entity name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("entity name cannot exceed 255 characters")
	}

	// Check for characters that could break search syntax
	invalidChars := []string{"\n", "\r", "\t"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("entity name cannot contain newlines or tab characters")
		}
	}

	// Warn about potentially problematic characters but don't forbid them
	// (since they can be escaped on the frontend)
	return nil
}

func validateEntityDescription(description string) error {
	if len(description) > 2000 {
		return fmt.Errorf("entity description cannot exceed 2000 characters")
	}
	return nil
}

type UpdateEntityRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	CardPK      *int   `json:"card_pk"`
}

func (s *Handler) GetEntitiesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	searchTerm := r.URL.Query().Get("search")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	sortBy := r.URL.Query().Get("sort_by")
	sortDirection := r.URL.Query().Get("sort_direction")

	// Use the service layer for entity retrieval
	params := services.EntityQueryParams{
		SearchTerm:    searchTerm,
		Page:          page,
		PerPage:       perPage,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	}

	response, err := services.GetEntities(s.DB, s.Server.TypesenseClient, userID, params)
	if err != nil {
		log.Printf("Error getting entities: %v", err)
		http.Error(w, "Failed to retrieve entities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) QueryEntitiesForCard(userID int, cardPK int) ([]models.Entity, error) {
	query := `
	SELECT DISTINCT
		e.id, e.user_id, e.name, e.description, e.type, e.created_at, e.updated_at, e.card_pk
	FROM 
		entities e
	LEFT JOIN 
		entity_card_junction ecj ON e.id = ecj.entity_id
	WHERE 
		e.user_id = $2 
		AND (ecj.card_pk = $1 OR e.card_pk = $1)`

	rows, err := s.DB.Query(query, cardPK, userID)
	if err != nil {
		log.Printf("err %v", err)
		return []models.Entity{}, err
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var entity models.Entity
		if err := rows.Scan(
			&entity.ID,
			&entity.UserID,
			&entity.Name,
			&entity.Description,
			&entity.Type,
			&entity.CreatedAt,
			&entity.UpdatedAt,
			&entity.CardPK,
		); err != nil {
			log.Printf("err %v", err)
			return entities, err
		}
		entities = append(entities, entity)
	}
	return entities, nil
}

func (s *Handler) MergeEntities(userID int, entity1ID int, entity2ID int) error {
	// Start transaction
	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Only rollback if we're not in testing mode (test framework handles cleanup)
	if s.ShouldCommitTx() {
		defer tx.Rollback()
	}

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

	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
	client.RequestType = "other"

	newDescription, err := services.GenerateNewEntityDescription(client, entity1, entity2, entity1.Name)
	if err != nil {
		newDescription = entity1.Description
	}

	// Preserve card_pk from either entity (prefer entity1, fallback to entity2)
	cardPK := entity1.CardPK
	if cardPK == nil {
		cardPK = entity2.CardPK
	}

	_, err = tx.Exec(`UPDATE entities SET description = $1, card_pk = $2 WHERE id = $3`, newDescription, cardPK, entity1.ID)
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
	if s.ShouldCommitTx() {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	// Update Typesense index: upsert surviving, delete removed
	// Skip during testing since the transaction isn't committed
	if s.ShouldCommitTx() {
		go func() {
		// Fetch the updated entity1 data after merge
		var updatedEntity models.Entity
		err := s.DB.QueryRow(`
			SELECT id, user_id, name, description, type, created_at, updated_at, card_pk
			FROM entities
			WHERE id = $1 AND user_id = $2
		`, entity1.ID, userID).Scan(
			&updatedEntity.ID,
			&updatedEntity.UserID,
			&updatedEntity.Name,
			&updatedEntity.Description,
			&updatedEntity.Type,
			&updatedEntity.CreatedAt,
			&updatedEntity.UpdatedAt,
			&updatedEntity.CardPK,
		)
		if err != nil {
			log.Printf("Error fetching updated entity after merge: %v", err)
			return
		}

		var partialCard *models.PartialCard
		if updatedEntity.CardPK != nil {
			card, err := s.QueryPartialCardByID(userID, *updatedEntity.CardPK)
			if err == nil {
				partialCard = &card
			}
		}
		s.upsertEntityToTypesense(updatedEntity, partialCard)
		s.deleteEntityTypesense(entity2.ID)
		}()
	}

	return nil
}

type MergeEntitiesRequest struct {
	Entity1ID int `json:"entity1_id"`
	Entity2ID int `json:"entity2_id"`
}

func (s *Handler) MergeEntitiesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req MergeEntitiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Entity1ID == 0 || req.Entity2ID == 0 {
		http.Error(w, "Both entity IDs are required", http.StatusBadRequest)
		return
	}

	if req.Entity1ID == req.Entity2ID {
		http.Error(w, "Cannot merge an entity with itself", http.StatusBadRequest)
		return
	}

	err := s.MergeEntities(userID, req.Entity1ID, req.Entity2ID)
	if err != nil {
		log.Printf("Error merging entities: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Entities merged successfully",
	})
}

func (s *Handler) DeleteEntity(userID int, entityID int) error {
	// Start transaction
	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Only rollback if we're not in testing mode (test framework handles cleanup)
	if s.ShouldCommitTx() {
		defer tx.Rollback()
	}

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
	if s.ShouldCommitTx() {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	// Delete from Typesense after successful commit
	if s.ShouldCommitTx() {
		go s.deleteEntityTypesense(entityID)
	}

	return nil
}

func (s *Handler) DeleteEntityRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract entityID from URL parameters using mux instead of chi
	entityID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid entity ID", http.StatusBadRequest)
		return
	}

	err = s.DeleteEntity(userID, entityID)
	if err != nil {
		log.Printf("Error deleting entity: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Entity deleted successfully",
	})
}

func (s *Handler) validateCardAccess(userID int, cardPK int) error {
	var exists bool
	err := s.GetDB().QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM cards
			WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
		)
	`, cardPK, userID).Scan(&exists)

	if err != nil {
		return fmt.Errorf("error checking card access: %w", err)
	}

	if !exists {
		return fmt.Errorf("card not found or access denied")
	}

	return nil
}

func (s *Handler) UpdateEntity(userID int, entityID int, params UpdateEntityRequest) error {
	// Start transaction
	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Only rollback if we're not in testing mode (test framework handles cleanup)
	if s.ShouldCommitTx() {
		defer tx.Rollback()
	}

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

	// Validate card access if CardPK is provided
	if params.CardPK != nil {
		if err := s.validateCardAccess(userID, *params.CardPK); err != nil {
			return fmt.Errorf("invalid card reference: %w", err)
		}
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

	// First update the entity without the embedding
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

	// Commit transaction for the basic update
	if s.ShouldCommitTx() {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	// Only attempt to update embedding if not in test mode
	if !s.Server.Testing {
		// Launch goroutine to handle embedding update
		go func() {
			var entity models.Entity
			err := s.DB.QueryRow(`
				SELECT id, user_id, name, description, type, created_at, updated_at, card_pk
				FROM entities 
				WHERE id = $1 AND user_id = $2
			`, entityID, userID).Scan(
				&entity.ID,
				&entity.UserID,
				&entity.Name,
				&entity.Description,
				&entity.Type,
				&entity.CreatedAt,
				&entity.UpdatedAt,
				&entity.CardPK,
			)
			if err != nil {
				log.Printf("Error fetching updated entity %d: %v", entityID, err)
				return
			}

			// Also update Typesense with card if available
			var partialCard *models.PartialCard
			if entity.CardPK != nil {
				card, err := s.QueryPartialCardByID(userID, *entity.CardPK)
				if err == nil {
					partialCard = &card
				}
			}
			s.upsertEntityToTypesense(entity, partialCard)
		}()
	}

	return nil
}

func (s *Handler) UpdateEntityRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract entityID from URL parameters
	entityID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid entity ID", http.StatusBadRequest)
		return
	}

	var params UpdateEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate entity name and description
	if err := validateEntityName(params.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEntityDescription(params.Description); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.UpdateEntity(userID, entityID, params)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("Error updating entity: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Entity updated successfully",
	})
}

func (s *Handler) AddEntityToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract entityID and cardPK from URL parameters
	vars := mux.Vars(r)
	entityID, err := strconv.Atoi(vars["entityId"])
	if err != nil {
		http.Error(w, "Invalid entity ID", http.StatusBadRequest)
		return
	}

	cardPK, err := strconv.Atoi(vars["cardId"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify entity exists and belongs to user
	var exists bool
	err = s.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 
			FROM entities 
			WHERE id = $1 AND user_id = $2
		)`,
		entityID, userID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking entity existence: %v", err)
		http.Error(w, "Failed to verify entity", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Entity not found or does not belong to user", http.StatusNotFound)
		return
	}

	// Verify card exists and belongs to user
	if err := s.validateCardAccess(userID, cardPK); err != nil {
		http.Error(w, "Card not found or access denied", http.StatusNotFound)
		return
	}

	// Add the entity-card relationship
	_, err = s.DB.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
		ON CONFLICT (entity_id, card_pk) DO NOTHING
	`, userID, entityID, cardPK)
	if err != nil {
		log.Printf("Error adding entity to card: %v", err)
		http.Error(w, "Failed to add entity to card", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Entity added to card successfully",
	})
}

func (s *Handler) RemoveEntityFromCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract entityID and cardPK from URL parameters
	vars := mux.Vars(r)
	entityID, err := strconv.Atoi(vars["entityId"])
	if err != nil {
		http.Error(w, "Invalid entity ID", http.StatusBadRequest)
		return
	}

	cardPK, err := strconv.Atoi(vars["cardId"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Delete the entity-card relationship
	_, err = s.DB.Exec(`
		DELETE FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, entityID, cardPK, userID)
	if err != nil {
		log.Printf("Error removing entity from card: %v", err)
		http.Error(w, "Failed to remove entity from card", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Entity removed from card successfully",
	})
}

func (s *Handler) GetEntityByID(userID int, entityID int) (models.Entity, error) {
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

	err := s.DB.QueryRow(query, userID, entityID).Scan(
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
		var parentIDPtr *int
		if cardParentID.Valid {
			parentID := int(cardParentID.Int64)
			parentIDPtr = &parentID
		}
		entity.Card = &models.PartialCard{
			ID:        int(cardID.Int64),
			CardID:    cardCardID.String,
			Title:     cardTitle.String,
			UserID:    int(cardUserID.Int64),
			ParentID:  parentIDPtr,
			CreatedAt: cardCreatedAt.Time,
			UpdatedAt: cardUpdatedAt.Time,
		}
	}

	return entity, nil
}

func (s *Handler) GetEntityByIDRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	entityID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid entity ID", http.StatusBadRequest)
		return
	}

	entity, err := s.GetEntityByID(userID, entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Entity not found", http.StatusNotFound)
			return
		}
		log.Printf("error querying entity by id: %v", err)
		http.Error(w, "Failed to query entity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entity)
}

func (s *Handler) GetEntityByNameRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract entity name from URL parameters
	vars := mux.Vars(r)
	entityName := vars["name"]
	if entityName == "" {
		http.Error(w, "Entity name is required", http.StatusBadRequest)
		return
	}

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

	err := s.DB.QueryRow(query, userID, entityName).Scan(
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
		if err == sql.ErrNoRows {
			http.Error(w, "Entity not found", http.StatusNotFound)
			return
		}
		log.Printf("error querying entity by name: %v", err)
		http.Error(w, "Failed to query entity", http.StatusInternalServerError)
		return
	}

	// Set the linked card if it exists
	if cardID.Valid {
		var parentIDPtr *int
		if cardParentID.Valid {
			parentID := int(cardParentID.Int64)
			parentIDPtr = &parentID
		}
		entity.Card = &models.PartialCard{
			ID:        int(cardID.Int64),
			CardID:    cardCardID.String,
			Title:     cardTitle.String,
			UserID:    int(cardUserID.Int64),
			ParentID:  parentIDPtr,
			CreatedAt: cardCreatedAt.Time,
			UpdatedAt: cardUpdatedAt.Time,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entity)
}

func (s *Handler) LinkCardToEntityIfPossible(userID int, card models.Card) error {
	// Skip during testing to avoid external service calls
	if s.Server.Testing {
		return nil
	}

	var entityID int
	var cardPK sql.NullInt64

	log.Printf("dwe be linking boys")

	err := s.DB.QueryRow(`
        SELECT id, card_pk FROM entities
        WHERE user_id = $1 AND name = $2
        LIMIT 1
    `, userID, card.Title).Scan(&entityID, &cardPK)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No matching entity
		}
		return fmt.Errorf("error querying entity for card linking: %w", err)
	}

	if cardPK.Valid {
		return nil // Already linked
	}

	_, err = s.DB.Exec(`
        UPDATE entities
        SET card_pk = $1, updated_at = NOW()
        WHERE id = $2
    `, card.ID, entityID)
	if err != nil {
		return fmt.Errorf("error updating entity with card link: %w", err)
	}

	// Update Typesense index after successful link
	go func() {
		var ent models.Entity
		err := s.DB.QueryRow(`SELECT id, user_id, name, description, type, created_at, updated_at, card_pk
			FROM entities WHERE id = $1`, entityID).
			Scan(&ent.ID, &ent.UserID, &ent.Name, &ent.Description, &ent.Type, &ent.CreatedAt, &ent.UpdatedAt, &ent.CardPK)
		if err != nil {
			log.Printf("failed to fetch entity for typesense after link: %v", err)
			return
		}

		partialCard, err := s.QueryPartialCardByID(ent.UserID, card.ID)
		if err == nil {
			s.upsertEntityToTypesense(ent, &partialCard)
		} else {
			s.upsertEntityToTypesense(ent, nil)
		}
	}()

	return nil
}

func (s *Handler) GetEntityByLinkedCardPKRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardPK, err := strconv.Atoi(mux.Vars(r)["card_pk"])
	if err != nil {
		http.Error(w, "Invalid card_pk", http.StatusBadRequest)
		return
	}

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
            e.user_id = $1 AND e.card_pk = $2
        GROUP BY 
            e.id, e.user_id, e.name, e.description, e.type, e.created_at, e.updated_at, e.card_pk,
            c.id, c.card_id, c.title, c.user_id, c.parent_id, c.created_at, c.updated_at
    `

	var entity models.Entity
	var cardID sql.NullInt64
	var cardCardID, cardTitle sql.NullString
	var cardUserID, cardParentID sql.NullInt64
	var cardCreatedAt, cardUpdatedAt sql.NullTime

	err = s.DB.QueryRow(query, userID, cardPK).Scan(
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
		if err == sql.ErrNoRows {

			results := []models.Entity{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
			return
		}
		log.Printf("error querying entity by id: %v", err)
		http.Error(w, "Failed to query entity", http.StatusInternalServerError)
		return
	}

	if cardID.Valid {
		var parentIDPtr *int
		if cardParentID.Valid {
			parentID := int(cardParentID.Int64)
			parentIDPtr = &parentID
		}
		entity.Card = &models.PartialCard{
			ID:        int(cardID.Int64),
			CardID:    cardCardID.String,
			Title:     cardTitle.String,
			UserID:    int(cardUserID.Int64),
			ParentID:  parentIDPtr,
			CreatedAt: cardCreatedAt.Time,
			UpdatedAt: cardUpdatedAt.Time,
		}
	}

	results := []models.Entity{}
	results = append(results, entity)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Handler) upsertEntityToTypesense(entity models.Entity, card *models.PartialCard) {
	if s.Server.Testing {
		return
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	doc := map[string]interface{}{
		"id":         "entity-" + strconv.Itoa(entity.ID),
		"fact_pk":    -1,
		"card_id":    "",
		"card_pk":    -1,
		"entity_pk":  entity.ID,
		"user_id":    entity.UserID,
		"type":       "entity",
		"title":      entity.Name,
		"preview":    entity.Description,
		"parent_id":  -1,
		"created_at": entity.CreatedAt.Unix(),
		"updated_at": entity.UpdatedAt.Unix(),
	}

	doc["linked_card_id"] = ""
	doc["linked_card_pk"] = -1
	doc["linked_card_title"] = ""
	doc["linked_card_parent_id"] = -1
	doc["tags"] = []string{}

	if card != nil {
		doc["linked_card_id"] = card.CardID
		doc["linked_card_pk"] = card.ID
		doc["linked_card_title"] = card.Title
		doc["linked_card_parent_id"] = card.ParentID
	}
	log.Printf("upserting %v", doc)

	_, err := s.Server.TypesenseClient.Collection(collectionName).
		Documents().Upsert(context.Background(), doc)
	if err != nil {
		log.Printf("failed to upsert entity ID %d: %v", entity.ID, err)
	}
}

func floatsToString(floats []float32) string {
	if floats == nil {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[")
	for i, f := range floats {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteString("]")
	return b.String()
}

func (s *Handler) GetSimilarEntitiesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	entityID, err := strconv.Atoi(vars["id"])
	// if err != nil {
	// 	http.Error(w, "Invalid entity ID", http.StatusBadRequest)
	// 	return
	// }
	entity, err := s.GetEntityByID(userID, entityID)
	if err != nil {
		log.Printf("error getting entity for similar entity: %v", err)

		http.Error(w, "Entity not found", http.StatusNotFound)
		return
	}

	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Use server similarity function
	entityObjs, err := s.Server.FindSimilarEntities(r.Context(), entity, limit)
	if err != nil {
		log.Printf("error finding similar entities: %v", err)
		http.Error(w, "Failed to search for similar entities", http.StatusInternalServerError)
		return
	}

	entityIDs := make([]int, len(entityObjs))
	// Build a map from entity ID to similarity score
	scoreMap := make(map[int]float64, len(entityObjs))
	for i := range len(entityObjs) {
		entityIDs[i] = entityObjs[i].ID
		scoreMap[entityObjs[i].ID] = entityObjs[i].Score
	}

	if len(entityIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EntityWithScore{})
		return
	}

	// 3. Fetch full entity data from DB
	query := `
        SELECT
            e.id, e.user_id, e.name, e.description, e.type, e.created_at, e.updated_at, e.card_pk,
            (SELECT COUNT(DISTINCT ecj.card_pk) FROM entity_card_junction ecj WHERE ecj.entity_id = e.id) as card_count,
            c.id as linked_card_id, c.card_id as linked_card_card_id, c.title as linked_card_title,
            c.user_id as linked_card_user_id, c.parent_id as linked_card_parent_id,
            c.created_at as linked_card_created_at, c.updated_at as linked_card_updated_at
        FROM
            entities e
            LEFT JOIN cards c ON e.card_pk = c.id AND c.is_deleted = FALSE
        WHERE
            e.id = ANY($1)
        ORDER BY
            array_position($1, e.id)
    `

	rows, err := s.DB.Query(query, pq.Array(entityIDs))
	if err != nil {
		log.Printf("error querying similar entities from db: %v", err)
		http.Error(w, "Failed to query similar entities", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entities []EntityWithScore
	for rows.Next() {
		var entity EntityWithScore
		var cardID sql.NullInt64
		var cardCardID, cardTitle sql.NullString
		var cardUserID, cardParentID sql.NullInt64
		var cardCreatedAt, cardUpdatedAt sql.NullTime

		err := rows.Scan(
			&entity.ID, &entity.UserID, &entity.Name, &entity.Description, &entity.Type,
			&entity.CreatedAt, &entity.UpdatedAt, &entity.CardPK, &entity.CardCount,
			&cardID, &cardCardID, &cardTitle, &cardUserID, &cardParentID,
			&cardCreatedAt, &cardUpdatedAt,
		)
		if err != nil {
			log.Printf("error scanning similar entity: %v", err)
			http.Error(w, "Failed to scan similar entities", http.StatusInternalServerError)
			return
		}

		// Attach the similarity score from the map
		entity.Score = scoreMap[entity.ID]

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

		entities = append(entities, entity)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

func (s *Handler) deleteEntityTypesense(entityPK int) {
	if s.Server.Testing {
		return
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	_, err := s.Server.TypesenseClient.Collection(collectionName).
		Document("entity-" + strconv.Itoa(entityPK)).Delete(context.Background())
	if err != nil {
		log.Printf("failed to delete entity ID %d: %v", entityPK, err)
	}
}
