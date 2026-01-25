package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	readability "github.com/go-shiori/go-readability"
	"github.com/gorilla/mux"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html"
)

// validateStructuredData validates structured_data against a schema definition and returns cleaned data
func validateStructuredData(structuredData json.RawMessage, schema *models.SchemaDefinition) (json.RawMessage, error) {
	// Parse the structured data into a map
	var data map[string]interface{}
	if len(structuredData) > 0 {
		if err := json.Unmarshal(structuredData, &data); err != nil {
			return nil, fmt.Errorf("invalid structured_data JSON: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	// Build a map of field definitions for quick lookup
	fieldMap := make(map[string]models.FieldDefinition)
	for _, field := range schema.Fields {
		fieldMap[field.Name] = field
	}

	// Check all required fields are present
	for _, field := range schema.Fields {
		if field.Required {
			if _, exists := data[field.Name]; !exists {
				return nil, fmt.Errorf("required field '%s' is missing", field.Name)
			}
		}
	}

	// Validate each field and clean data (remove fields not in schema)
	cleanedData := make(map[string]interface{})
	for fieldName, value := range data {
		fieldDef, exists := fieldMap[fieldName]
		if !exists {
			// Skip fields not defined in schema (remove old/renamed fields)
			continue
		}

		if err := validateFieldValue(fieldName, value, fieldDef); err != nil {
			return nil, err
		}
		cleanedData[fieldName] = value
	}

	// Marshal cleaned data back to JSON
	cleanedJSON, err := json.Marshal(cleanedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cleaned data: %w", err)
	}

	return cleanedJSON, nil
}

// validateFieldValue validates a single field value against its definition
func validateFieldValue(fieldName string, value interface{}, fieldDef models.FieldDefinition) error {
	if value == nil {
		if fieldDef.Required {
			return fmt.Errorf("required field '%s' cannot be null", fieldName)
		}
		return nil
	}

	switch fieldDef.Type {
	case "text":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}

	case "number":
		// Accept both float64 and int from JSON
		switch v := value.(type) {
		case float64, int, int64, float32:
			// Valid number types
		case string:
			// Try to parse string as number
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				return fmt.Errorf("field '%s' must be a number", fieldName)
			}
		default:
			return fmt.Errorf("field '%s' must be a number", fieldName)
		}

	case "date":
		// Accept ISO 8601 date string
		dateStr, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a date string", fieldName)
		}
		// Validate ISO 8601 format
		_, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// Also try date-only format
			_, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid ISO 8601 date", fieldName)
			}
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", fieldName)
		}

	case "select":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string for select type", fieldName)
		}
		// Check if value is in options
		validOption := false
		for _, opt := range fieldDef.Options {
			if opt == strVal {
				validOption = true
				break
			}
		}
		if !validOption {
			return fmt.Errorf("field '%s' value '%s' is not in valid options", fieldName, strVal)
		}

	case "multi-select":
		// Expect array of strings
		arrayVal, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("field '%s' must be an array for multi-select type", fieldName)
		}
		for _, item := range arrayVal {
			strItem, ok := item.(string)
			if !ok {
				return fmt.Errorf("field '%s' array items must be strings", fieldName)
			}
			// Check if value is in options
			validOption := false
			for _, opt := range fieldDef.Options {
				if opt == strItem {
					validOption = true
					break
				}
			}
			if !validOption {
				return fmt.Errorf("field '%s' value '%s' is not in valid options", fieldName, strItem)
			}
		}

	case "link_to_card":
		// Expect integer card ID
		// Note: We don't validate the card exists here to avoid circular dependencies
		// This will be validated at the handler level where we have DB access
		switch value.(type) {
		case float64, int, int64, float32:
			// Valid number types
		case string:
			// Try to parse string as int
			_, err := strconv.Atoi(value.(string))
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid card ID (integer)", fieldName)
			}
		default:
			return fmt.Errorf("field '%s' must be a valid card ID (integer)", fieldName)
		}

	default:
		return fmt.Errorf("unknown field type '%s' for field '%s'", fieldDef.Type, fieldName)
	}

	return nil
}

// validateLinkToCardFields validates that link_to_card fields reference valid cards
func validateLinkToCardFields(db *sql.DB, userID int, structuredData json.RawMessage, schema *models.SchemaDefinition) error {
	// Parse the structured data
	var data map[string]interface{}
	if err := json.Unmarshal(structuredData, &data); err != nil {
		return err
	}

	// Check each field
	for _, field := range schema.Fields {
		if field.Type == "link_to_card" {
			value, hasValue := data[field.Name]
			if !hasValue || value == nil {
				continue
			}

			// Extract card ID
			var cardID int
			switch v := value.(type) {
			case float64:
				cardID = int(v)
			case int:
				cardID = v
			case string:
				parsedID, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("field '%s': invalid card ID format", field.Name)
				}
				cardID = parsedID
			default:
				return fmt.Errorf("field '%s': invalid card ID type", field.Name)
			}

			// Validate card exists and belongs to user
			var cardExists bool
			err := db.QueryRow(`
				SELECT EXISTS(SELECT 1 FROM cards WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE)
			`, cardID, userID).Scan(&cardExists)
			if err != nil {
				return fmt.Errorf("field '%s': failed to validate card reference", field.Name)
			}
			if !cardExists {
				return fmt.Errorf("field '%s': referenced card (ID %d) does not exist", field.Name, cardID)
			}
		}
	}

	return nil
}

func (s *Handler) getDirectlinks(userID int, card models.Card) []models.PartialCard {
	backlinks := services.ExtractBacklinks(card.Body)
	var directLinks []models.PartialCard

	for _, value := range backlinks {
		card, err := s.QueryPartialCard(userID, value)
		if err == nil {
			directLinks = append(directLinks, card)
		}

	}

	return directLinks
}

func getUniqueCards(input []models.PartialCard) []models.PartialCard {
	u := make([]models.PartialCard, 0, len(input))
	m := make(map[string]bool)

	for _, card := range input {
		if _, ok := m[card.CardID]; !ok {
			m[card.CardID] = true
			u = append(u, card)
		}
	}
	return u
}

func (s *Handler) getReferences(userID int, card models.Card) ([]models.PartialCard, error) {
	directLinks := s.getDirectlinks(userID, card)
	backlinks, _ := services.GetBacklinks(s.DB, userID, card.CardID)
	links := append(directLinks, backlinks...)
	if len(links) == 0 {
		return []models.PartialCard{}, nil
	}
	sort.Slice(links, func(x, y int) bool {
		return links[x].CardID > links[y].CardID
	})
	links = getUniqueCards(links)

	// Fetch tags for each card
	for i := range links {
		tags, err := services.QueryTagsForCard(s.DB, userID, links[i].ID)
		if err != nil {
			log.Printf("Failed to fetch tags for card ID %d: %v", links[i].ID, err)
			// Continue without tags rather than failing entirely
			links[i].Tags = []models.Tag{}
		} else {
			links[i].Tags = tags
		}
	}

	return links, nil
}

func getCardById(cards []models.Card, id int) (models.Card, error) {
	for _, card := range cards {
		if card.ID == id {
			return card, nil
		}
	}
	return models.Card{}, fmt.Errorf("unable to find card")

}

func (s *Handler) checkChunkLinkedOrRelated(
	userID int,
	mainCard models.Card,
	relatedCard models.CardChunk,
) bool {
	if relatedCard.ParentID == mainCard.ID {
		return true
	}
	references, err := s.getReferences(userID, mainCard)
	if err != nil {
		return true
	}
	for _, ref := range references {
		if ref.ID == relatedCard.ID {
			return true
		}
	}
	return false
}

// GetCardFilesRoute returns the files for a given card
func (s *Handler) GetCardFilesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	files, err := s.getFilesFromCardPK(userID, card.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// GetCardTagsRoute returns the tags for a given card
func (s *Handler) GetCardTagsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tags, err := services.QueryTagsForCard(s.DB, userID, card.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// GetCardTasksRoute returns the tasks for a given card
func (s *Handler) GetCardTasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tasks, err := s.QueryTasksByCard(userID, card.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetCardEntitiesRoute returns the entities for a given card
func (s *Handler) GetCardEntitiesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	entities, err := s.QueryEntitiesForCard(userID, card.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// GetCardChildrenRoute returns the children for a given card
func (s *Handler) GetCardChildrenRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	children, err := services.GetChildCards(s.DB, userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(children)
}

// GetCardWithDescendantsRoute returns a card with all its descendants recursively, including depth information
func (s *Handler) GetCardWithDescendantsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Get card with descendants with depth information
	result, err := services.GetCardWithDescendants(s.DB, userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetCardWithDescendantsPaginatedRoute returns a card with descendants limited to a specific depth (for performance)
func (s *Handler) GetCardWithDescendantsPaginatedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	maxDepthStr := mux.Vars(r)["depth"]
	maxDepth, err := strconv.Atoi(maxDepthStr)
	if err != nil {
		http.Error(w, "Invalid depth", http.StatusBadRequest)
		return
	}

	// Get card with descendants limited by depth
	result, err := services.GetCardWithDescendantsLimited(s.DB, userID, id, maxDepth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CategorizedReferences represents references categorized by their relationship type
type CategorizedReferences struct {
	Bidirectional []models.PartialCard `json:"bidirectional"` // Two-way links (mutual references)
	Outgoing      []models.PartialCard `json:"outgoing"`      // One-way links (this card references them)
	Incoming      []models.PartialCard `json:"incoming"`      // One-way links (they reference this card)
}

// GetCardReferencesRoute returns the references (directlinks + backlinks) for a given card, categorized by relationship type
func (s *Handler) GetCardReferencesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	directLinks := s.getDirectlinks(userID, card)
	backlinks, _ := services.GetBacklinks(s.DB, userID, card.CardID)

	// Fetch tags for all cards first
	allCards := append(directLinks, backlinks...)
	for i := range allCards {
		tags, err := services.QueryTagsForCard(s.DB, userID, allCards[i].ID)
		if err != nil {
			log.Printf("Failed to fetch tags for card ID %d: %v", allCards[i].ID, err)
			allCards[i].Tags = []models.Tag{}
		} else {
			allCards[i].Tags = tags
		}
	}

	// Create maps for quick lookup
	directMap := make(map[int]models.PartialCard)
	backMap := make(map[int]models.PartialCard)

	for _, card := range directLinks {
		directMap[card.ID] = card
	}
	for _, card := range backlinks {
		backMap[card.ID] = card
	}

	// Categorize references
	categorized := CategorizedReferences{
		Bidirectional: []models.PartialCard{},
		Outgoing:      []models.PartialCard{},
		Incoming:      []models.PartialCard{},
	}

	// Find bidirectional links
	for id, card := range directMap {
		if _, exists := backMap[id]; exists {
			categorized.Bidirectional = append(categorized.Bidirectional, card)
		} else {
			categorized.Outgoing = append(categorized.Outgoing, card)
		}
	}

	// Find incoming-only links
	for id, card := range backMap {
		if _, exists := directMap[id]; !exists {
			categorized.Incoming = append(categorized.Incoming, card)
		}
	}

	// Sort each category by card_id
	sort.Slice(categorized.Bidirectional, func(i, j int) bool {
		return categorized.Bidirectional[i].CardID > categorized.Bidirectional[j].CardID
	})
	sort.Slice(categorized.Outgoing, func(i, j int) bool {
		return categorized.Outgoing[i].CardID > categorized.Outgoing[j].CardID
	})
	sort.Slice(categorized.Incoming, func(i, j int) bool {
		return categorized.Incoming[i].CardID > categorized.Incoming[j].CardID
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categorized)
}

// GetCardRoute returns a specific card by ID with related details
func (s *Handler) GetCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if the card is starred by the current user
	isStarred, err := s.IsCardStarred(userID, id)
	if err != nil {
		log.Printf("Error checking if card is starred: %v", err)
		// Continue even if we can't determine star status
	} else {
		card.IsStarred = isStarred
	}
	parent, err := s.QueryPartialCardByID(userID, card.ParentID)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	card.Parent = parent

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (s *Handler) UpdateCardRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("asdsa id %v %v", id, err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	_, err = s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var params models.EditCardParams

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("err? %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate schema data if provided
	if params.SchemaID != nil {
		// Fetch the schema definition
		query := `SELECT id, name, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		schema, err := models.ScanSchemaDefinition(s.DB.QueryRow(query, *params.SchemaID, userID))
		if err != nil {
			log.Printf("Error fetching schema: %v", err)
			http.Error(w, "Failed to fetch schema definition", http.StatusInternalServerError)
			return
		}
		if schema == nil {
			http.Error(w, "Schema not found", http.StatusNotFound)
			return
		}

		// Validate and clean structured_data against schema
		if params.StructuredData != nil && len(*params.StructuredData) > 0 {
			cleanedData, err := validateStructuredData(*params.StructuredData, schema)
			if err != nil {
				log.Printf("Structured data validation error: %v", err)
				http.Error(w, fmt.Sprintf("Invalid structured data: %v", err), http.StatusBadRequest)
				return
			}
			params.StructuredData = &cleanedData
		} else {
			// If schema_id is provided but no structured_data, check if all fields are optional
			for _, field := range schema.Fields {
				if field.Required {
					http.Error(w, fmt.Sprintf("Schema requires field '%s' but no structured_data provided", field.Name), http.StatusBadRequest)
					return
				}
			}
		}

		// Additional validation for link_to_card fields
		if params.StructuredData != nil && len(*params.StructuredData) > 0 {
			if err := validateLinkToCardFields(s.DB, userID, *params.StructuredData, schema); err != nil {
				log.Printf("Link to card validation error: %v", err)
				http.Error(w, fmt.Sprintf("Invalid link_to_card reference: %v", err), http.StatusBadRequest)
				return
			}
		}
	} else if params.StructuredData != nil && len(*params.StructuredData) > 0 {
		// structured_data provided without schema_id
		http.Error(w, "structured_data requires schema_id to be specified", http.StatusBadRequest)
		return
	}

	card, err := services.UpdateCard(s.DB, userID, id, params)
	if err != nil {
		log.Printf("error updating card: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.UserHasSubscription(userID) {
		s.GenerateMemory(uint(userID), card.Body)
		if params.ProcessEntitiesAndFacts != nil && *params.ProcessEntitiesAndFacts {
			s.ProcessEntitiesAndFacts(userID, card)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (s *Handler) CreateCardRoute(w http.ResponseWriter, r *http.Request) {
	var params models.EditCardParams
	var err error
	userID := r.Context().Value("current_user").(int)

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("err? %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate schema data if provided
	if params.SchemaID != nil {
		// Fetch the schema definition
		query := `SELECT id, name, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		schema, err := models.ScanSchemaDefinition(s.DB.QueryRow(query, *params.SchemaID, userID))
		if err != nil {
			log.Printf("Error fetching schema: %v", err)
			http.Error(w, "Failed to fetch schema definition", http.StatusInternalServerError)
			return
		}
		if schema == nil {
			http.Error(w, "Schema not found", http.StatusNotFound)
			return
		}

		// Validate and clean structured_data against schema
		if params.StructuredData != nil && len(*params.StructuredData) > 0 {
			cleanedData, err := validateStructuredData(*params.StructuredData, schema)
			if err != nil {
				log.Printf("Structured data validation error: %v", err)
				http.Error(w, fmt.Sprintf("Invalid structured data: %v", err), http.StatusBadRequest)
				return
			}
			params.StructuredData = &cleanedData
		} else {
			// If schema_id is provided but no structured_data, check if all fields are optional
			for _, field := range schema.Fields {
				if field.Required {
					http.Error(w, fmt.Sprintf("Schema requires field '%s' but no structured_data provided", field.Name), http.StatusBadRequest)
					return
				}
			}
		}

		// Additional validation for link_to_card fields
		if params.StructuredData != nil && len(*params.StructuredData) > 0 {
			if err := validateLinkToCardFields(s.DB, userID, *params.StructuredData, schema); err != nil {
				log.Printf("Link to card validation error: %v", err)
				http.Error(w, fmt.Sprintf("Invalid link_to_card reference: %v", err), http.StatusBadRequest)
				return
			}
		}
	} else if params.StructuredData != nil && len(*params.StructuredData) > 0 {
		// structured_data provided without schema_id
		http.Error(w, "structured_data requires schema_id to be specified", http.StatusBadRequest)
		return
	}

	card, err := services.CreateCard(s.DB, userID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.UserHasSubscription(userID) {
		s.GenerateMemory(uint(userID), card.Body)
		if params.ProcessEntitiesAndFacts == nil || *params.ProcessEntitiesAndFacts {
			s.ProcessEntitiesAndFacts(userID, card)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (s *Handler) DeleteCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	err = services.DeleteCard(s.DB, userID, id)
	if err != nil {
		if err.Error() == "card not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err.Error() == "card has backlinks, cannot be deleted" || err.Error() == "card has children, cannot be deleted" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Handler) GetNextRootCardIDRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	nextID := s.getNextRootCardID(userID)

	response := models.NextIDResponse{
		NextID: nextID,
		Error:  false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) getNextRootCardID(userID int) string {
	var result string

	// Query to get the highest numeric card_id
	query := `
        SELECT card_id 
        FROM cards 
        WHERE user_id = $1 
        AND is_deleted = FALSE 
        AND card_id ~ '^[0-9]+$'  -- Only match pure numeric card_ids
        ORDER BY CAST(card_id AS INTEGER) DESC
        LIMIT 1
    `

	err := s.DB.QueryRow(query, userID).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error finding next root card ID: %v", err)
		return "1" // Default to 1 if there's an error
	}

	if result == "" {
		return "1" // If no cards exist, start with 1
	}

	// Convert the highest card_id to int and increment
	highestNumber, err := strconv.Atoi(result)
	if err != nil {
		log.Printf("Error converting card_id to number: %v", err)
		return "1"
	}

	nextNumber := highestNumber + 1
	return strconv.Itoa(nextNumber)
}

func (s *Handler) GetNextChildCardIDRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	nextID := s.getNextChildCardID(userID, id)

	response := models.NextIDResponse{
		NextID: nextID,
		Error:  false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) getNextChildCardID(userID int, parentID int) string {
	// 1. Get parent card's card_id (human readable ID)
	var parentCardID string
	err := s.DB.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
	if err != nil {
		log.Printf("Error finding parent card ID for parentID %d: %v", parentID, err)
		return "" // Return empty on error
	}

	// 2. Get all existing children using service
	children, err := services.GetChildCards(s.DB, userID, parentID)
	if err != nil {
		log.Printf("Error getting child cards for parentID %d: %v", parentID, err)
		return parentCardID + ".1" // Default to .1 if there's an error
	}

	// 3. Extract numeric suffixes from children's card_ids
	childNumbers := make([]int, 0)
	parentIDLength := len(parentCardID)

	for _, child := range children {
		childID := child.CardID

		// Verify this is actually a direct child by checking it starts with parent ID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}

		// Get the part after the parent ID
		suffix := childID[parentIDLength:]

		// Extract the first number after any separator using regex
		re := regexp.MustCompile(`^[.\\/-]+(\d+)`)
		match := re.FindStringSubmatch(suffix)
		if len(match) == 2 {
			num, err := strconv.Atoi(match[1])
			if err == nil {
				childNumbers = append(childNumbers, num)
			}
		}
	}

	// 4. Find the highest number and increment
	if len(childNumbers) == 0 {
		return parentCardID + ".1" // No existing children, start with 1
	}

	maxNumber := 0
	for _, num := range childNumbers {
		if num > maxNumber {
			maxNumber = num
		}
	}

	nextNumber := maxNumber + 1
	return fmt.Sprintf("%s.%d", parentCardID, nextNumber)
}

func (s *Handler) QueryPartialCardByID(userID, id int) (models.PartialCard, error) {
	return services.GetPartialCard(s.DB, userID, id)
}

func (s *Handler) QueryPartialCard(userID int, cardID string) (models.PartialCard, error) {
	return services.GetPartialCardByCardID(s.DB, userID, cardID)

}

func (s *Handler) QueryFullCard(userID int, id int) (models.Card, error) {
	s.logCardView(id, userID)
	return services.GetFullCard(s.DB, userID, id)
}

func (s *Handler) GetCardAuditEventsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify the user owns this card
	_, err = s.QueryFullCard(userID, cardID)
	if err != nil {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	events, err := services.GetAuditEvents(s.DB, "card", cardID)
	if err != nil {
		log.Printf("Error getting audit events: %v", err)
		http.Error(w, "Error retrieving audit events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// GetTemplatesRoute returns all templates for the current user
func (s *Handler) GetTemplatesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	templates, err := s.QueryTemplates(userID)
	if err != nil {
		log.Printf("Error querying templates: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplateRoute returns a specific template by ID
func (s *Handler) GetTemplateRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	template, err := s.QueryTemplate(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// CreateTemplateRoute creates a new template
func (s *Handler) CreateTemplateRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateTemplateParams
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	template, err := s.CreateTemplate(userID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// UpdateTemplateRoute updates an existing template
func (s *Handler) UpdateTemplateRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var params models.UpdateTemplateParams
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	template, err := s.UpdateTemplate(userID, id, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplateRoute deletes a template
func (s *Handler) DeleteTemplateRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	err = s.DeleteTemplate(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// QueryTemplates returns all templates for a user
func (s *Handler) QueryTemplates(userID int) ([]models.CardTemplate, error) {
	query := `
	SELECT id, user_id, name, title, body, created_at, updated_at
	FROM card_templates
	WHERE user_id = $1
	ORDER BY updated_at DESC
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.CardTemplate
	for rows.Next() {
		var template models.CardTemplate
		if err := rows.Scan(
			&template.ID,
			&template.UserID,
			&template.Name,
			&template.Title,
			&template.Body,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// QueryTemplate returns a specific template by ID
func (s *Handler) QueryTemplate(userID, id int) (models.CardTemplate, error) {
	var template models.CardTemplate

	query := `
	SELECT id, user_id, name, title, body, created_at, updated_at
	FROM card_templates
	WHERE id = $1 AND user_id = $2
	`

	err := s.DB.QueryRow(query, id, userID).Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return models.CardTemplate{}, fmt.Errorf("template not found")
	}

	return template, nil
}

// CreateTemplate creates a new template
func (s *Handler) CreateTemplate(userID int, params models.CreateTemplateParams) (models.CardTemplate, error) {
	var template models.CardTemplate

	query := `
	INSERT INTO card_templates (user_id, name, title, body, created_at, updated_at)
	VALUES ($1, $2, $3, $4, NOW(), NOW())
	RETURNING id, user_id, name, title, body, created_at, updated_at
	`

	err := s.DB.QueryRow(query, userID, params.Name, params.Title, params.Body).Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return models.CardTemplate{}, err
	}

	return template, nil
}

// UpdateTemplate updates an existing template
func (s *Handler) UpdateTemplate(userID, id int, params models.UpdateTemplateParams) (models.CardTemplate, error) {
	var template models.CardTemplate

	query := `
	UPDATE card_templates
	SET name = $1, title = $2, body = $3, updated_at = NOW()
	WHERE id = $4 AND user_id = $5
	RETURNING id, user_id, name, title, body, created_at, updated_at
	`

	err := s.DB.QueryRow(query, params.Name, params.Title, params.Body, id, userID).Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return models.CardTemplate{}, fmt.Errorf("failed to update template: %v", err)
	}

	return template, nil
}

// DeleteTemplate deletes a template
func (s *Handler) DeleteTemplate(userID, id int) error {
	query := `
	DELETE FROM card_templates
	WHERE id = $1 AND user_id = $2
	`

	result, err := s.DB.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

type Parser struct {
	// Add any dependencies here if needed
}

type ParseResult struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	URL      string `json:"url,omitempty"`
	Author   string `json:"author,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
	SiteName string `json:"site_name,omitempty"`
	// Add any other fields you want to return
}

func (p *Parser) ParseHTML(htmlContent string, urlStr string) (ParseResult, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return ParseResult{}, errors.New("empty HTML provided")
	}

	// Parse the HTML string into html.Node
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ParseResult{}, err
	}

	// Parse the URL
	pageURL, err := url.Parse(urlStr)
	if err != nil {
		return ParseResult{}, err
	}

	// Create parser and parse the document
	parser := readability.NewParser()
	article, err := parser.ParseDocument(doc, pageURL)
	if err != nil {
		return ParseResult{}, err
	}
	markdown, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		return ParseResult{}, err
	}

	result := ParseResult{
		Title:    article.Title,
		Content:  markdown,
		URL:      urlStr,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		SiteName: article.SiteName,
	}

	return result, nil
}

type SuggestTitleRequest struct {
	Body string `json:"body"`
}

type SuggestTitleResponse struct {
	SuggestedTitle string `json:"suggested_title"`
}

func (s *Handler) SuggestCardTitleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req SuggestTitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	// Get user memory for context
	userMemory, err := GetUserMemory(s.DB, int(userID))
	if err != nil {
		log.Printf("Error getting user memory: %v", err)
		// Continue without memory if there's an error
		userMemory = ""
	}

	suggestedTitle, err := s.suggestCardTitle(userID, req.Body, userMemory)
	if err != nil {
		log.Printf("Error suggesting card title: %v", err)
		http.Error(w, "Error generating title suggestion", http.StatusInternalServerError)
		return
	}

	response := SuggestTitleResponse{
		SuggestedTitle: suggestedTitle,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) suggestCardTitle(userID int, body string, userMemory string) (string, error) {
	client := services.NewDefaultClient(s.DB, userID)
	client.RequestType = "title_suggestion"

	memoryContext := ""
	if userMemory != "" {
		memoryContext = fmt.Sprintf("\n\nUser Context (from their knowledge base):\n%s", userMemory)
	}

	prompt := fmt.Sprintf(`You are an expert at creating concise, meaningful titles for knowledge management notes. Your task is to suggest a title for a note based on its content.

Guidelines:
- Create a title that captures the main concept or key insight
- Keep it concise (ideally 2-8 words)
- Make it descriptive enough to be searchable and memorable
- Consider the user's interests and knowledge domain when relevant
- Avoid generic titles like "Notes" or "Thoughts"
- Use title case

Note Content:
%s%s

Respond with ONLY the suggested title, no explanation or additional text.`, body, memoryContext)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	response, err := services.ExecuteLLMRequest(context.Background(), client, messages)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	title := strings.TrimSpace(response.Choices[0].Message.Content)
	// Remove any quotes that might be around the title
	title = strings.Trim(title, "\"'")

	return title, nil
}

// GetUnsortedCardsRoute returns paginated unsorted cards (cards with empty card_id)
func (s *Handler) GetUnsortedCardsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse pagination parameters
	page := 1
	perPage := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	offset := (page - 1) * perPage

	// Query for unsorted cards (card_id = '')
	query := `
	SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
	FROM cards
	WHERE user_id = $1 AND is_deleted = FALSE AND card_id = ''
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := s.DB.Query(query, userID, perPage, offset)
	if err != nil {
		log.Printf("Error querying unsorted cards: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var unsortedCards []models.Card
	for rows.Next() {
		var card models.Card
		err := rows.Scan(
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
			log.Printf("Error scanning unsorted card: %v", err)
			continue
		}
		unsortedCards = append(unsortedCards, card)
	}

	// Get total count for pagination
	var total int
	countQuery := `SELECT COUNT(*) FROM cards WHERE user_id = $1 AND is_deleted = FALSE AND card_id = ''`
	err = s.DB.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		log.Printf("Error counting unsorted cards: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"cards":       unsortedCards,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": (total + perPage - 1) / perPage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ParseURLRequest struct {
	URL string `json:"url"`
}

func (h *Handler) ParseURLRoute(w http.ResponseWriter, r *http.Request) {
	var req ParseURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	// Parse the URL using readability
	article, err := readability.FromURL(req.URL, 30*time.Second) // adjust timeout as needed
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	markdown, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to your response format if needed
	result := ParseResult{
		Title:    article.Title,
		Content:  markdown,
		URL:      req.URL,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		SiteName: article.SiteName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// RestoreCardToAuditEventRoute restores a card to the state it was in at the time of the audit event
func (s *Handler) RestoreCardToAuditEventRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	auditEventID, err := strconv.Atoi(mux.Vars(r)["auditEventId"])
	if err != nil {
		http.Error(w, "Invalid audit event ID", http.StatusBadRequest)
		return
	}

	// Verify the user owns this card
	_, err = s.QueryFullCard(userID, cardID)
	if err != nil {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	// Restore the card
	restoredCard, err := services.RestoreCardToAuditEvent(s.DB, userID, cardID, auditEventID)
	if err != nil {
		log.Printf("Error restoring card: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(restoredCard)
}

// CreateArticleRequest is the request payload for creating an article from a URL
type CreateArticleRequest struct {
	URL    string `json:"url"`
	CardID string `json:"card_id,omitempty"`
	Tags   string `json:"tags,omitempty"`
}

// CreateArticleRoute handles creating an article card from a URL in a single atomic operation
func (s *Handler) CreateArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	// Step 1: Parse the URL to extract article content
	article, err := readability.FromURL(req.URL, 30*time.Second)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", req.URL, err)
		http.Error(w, fmt.Sprintf("Failed to parse URL: %v", err), http.StatusInternalServerError)
		return
	}

	markdown, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		log.Printf("Error converting to markdown: %v", err)
		http.Error(w, fmt.Sprintf("Failed to convert content: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 2: Get next root card ID if not provided
	cardID := req.CardID
	if cardID == "" {
		cardID = s.getNextRootCardID(userID)
	}

	// Step 3: Build body with tags
	tags := req.Tags
	if tags == "" {
		tags = "#to-read #reference"
	}
	body := markdown + "\n\n" + tags

	// Step 4: Create the card
	params := models.EditCardParams{
		CardID:                  cardID,
		Title:                   article.Title,
		Body:                    body,
		Link:                    req.URL,
		ProcessEntitiesAndFacts: boolPtr(true),
	}

	card, err := services.CreateCard(s.DB, userID, params)
	if err != nil {
		log.Printf("Error creating card: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create card: %v", err), http.StatusBadRequest)
		return
	}

	// Step 5: Process entities and facts if user has subscription
	if s.UserHasSubscription(userID) {
		s.GenerateMemory(uint(userID), card.Body)
		s.ProcessEntitiesAndFacts(userID, card)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
