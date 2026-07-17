package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/models"
	"go-backend/server"
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
	if relatedCard.ParentID != nil && *relatedCard.ParentID == mainCard.ID {
		return true
	}
	references, err := services.GetReferences(s.DB, userID, mainCard)
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

	categorized, err := services.GetCategorizedReferences(s.DB, userID, card)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
	var parent models.PartialCard
	if card.ParentID != nil {
		parent, err = s.QueryPartialCardByID(userID, *card.ParentID)
		if err != nil {
			log.Printf("err %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
		query := `SELECT id, name, slug, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		schema, err := models.ScanSchemaDefinition(s.GetDB().QueryRow(query, *params.SchemaID, userID))
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
			cleanedData, err := services.ValidateStructuredData(*params.StructuredData, schema)
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
			if err := services.ValidateLinkToCardFields(s.GetDB(), userID, *params.StructuredData, schema); err != nil {
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

	card, err := services.UpdateCard(s.GetDB(), userID, id, params)
	if err != nil {
		log.Printf("error updating card: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shouldProcess := params.ProcessEntitiesAndFacts != nil && *params.ProcessEntitiesAndFacts
	s.ProcessCardAfterCreation(userID, card, shouldProcess)
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
		query := `SELECT id, name, slug, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		schema, err := models.ScanSchemaDefinition(s.GetDB().QueryRow(query, *params.SchemaID, userID))
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
			cleanedData, err := services.ValidateStructuredData(*params.StructuredData, schema)
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
			if err := services.ValidateLinkToCardFields(s.GetDB(), userID, *params.StructuredData, schema); err != nil {
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

	// Use transaction during testing, regular DB otherwise
	card, err := services.CreateCard(s.GetDB(), userID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shouldProcess := params.ProcessEntitiesAndFacts == nil || *params.ProcessEntitiesAndFacts
	s.ProcessCardAfterCreation(userID, card, shouldProcess)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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

	err := s.GetDB().QueryRow(query, userID).Scan(&result)
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
	err := s.GetDB().QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
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

	if len(children) == 0 {
		return parentCardID + ".1" // No existing children, start with .1
	}

	// 3. Classify children by their suffix type relative to the parent's card_id.
	parentIDLength := len(parentCardID)

	type suffixEntry struct {
		sep     string
		suffix  string
		isAlpha bool
	}

	var matchedEntries []suffixEntry

	numRe := regexp.MustCompile(`^[.\\/-]+(\d+)$`)
	alphaRe := regexp.MustCompile(`^[.\\/-]+([A-Z])$`)

	for _, child := range children {
		childID := child.CardID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}
		suffix := childID[parentIDLength:]

		if match := numRe.FindStringSubmatch(suffix); len(match) == 2 {
			matchedEntries = append(matchedEntries, suffixEntry{sep: string(suffix[0]), suffix: match[1], isAlpha: false})
		} else if match := alphaRe.FindStringSubmatch(suffix); len(match) == 2 {
			matchedEntries = append(matchedEntries, suffixEntry{sep: string(suffix[0]), suffix: match[1], isAlpha: true})
		}
	}

	// 4. If children matched the parent's card_id prefix, determine next suffix
	if len(matchedEntries) > 0 {
		type schemeKey struct {
			sep     string
			isAlpha bool
		}
		schemeCounts := make(map[schemeKey]int)
		for _, e := range matchedEntries {
			schemeCounts[schemeKey{e.sep, e.isAlpha}]++
		}
		var bestScheme schemeKey
		bestCount := 0
		for s, c := range schemeCounts {
			if c > bestCount {
				bestCount = c
				bestScheme = s
			}
		}

		if bestScheme.isAlpha {
			maxLetter := byte('A')
			for _, e := range matchedEntries {
				if e.isAlpha && e.sep == bestScheme.sep && e.suffix[0] > maxLetter {
					maxLetter = e.suffix[0]
				}
			}
			nextLetter := string(rune(maxLetter) + 1)
			return parentCardID + bestScheme.sep + nextLetter
		}

		maxNumber := 0
		for _, e := range matchedEntries {
			if !e.isAlpha && e.sep == bestScheme.sep {
				num, _ := strconv.Atoi(e.suffix)
				if num > maxNumber {
					maxNumber = num
				}
			}
		}
		return fmt.Sprintf("%s%s%d", parentCardID, bestScheme.sep, maxNumber+1)
	}

	// 5. Children don't use the parent's card_id as prefix.
	//    Detect the common prefix scheme among existing children and increment.
	type prefixEntry struct {
		prefix string
		sep    string
		num    int
	}

	re := regexp.MustCompile(`^(.+?)([./\-])(\d+)$`)
	var entries []prefixEntry

	for _, child := range children {
		match := re.FindStringSubmatch(child.CardID)
		if len(match) == 4 {
			num, err := strconv.Atoi(match[3])
			if err == nil {
				entries = append(entries, prefixEntry{
					prefix: match[1],
					sep:    match[2],
					num:    num,
				})
			}
		}
	}

	if len(entries) == 0 {
		return parentCardID + ".1"
	}

	prefixCounts := make(map[string]int)
	for _, e := range entries {
		prefixCounts[e.prefix]++
	}
	var bestPrefix string
	bestCount := 0
	for p, c := range prefixCounts {
		if c > bestCount {
			bestCount = c
			bestPrefix = p
		}
	}

	var bestSep string
	for _, e := range entries {
		if e.prefix == bestPrefix {
			bestSep = e.sep
			break
		}
	}

	maxNumber := 0
	for _, e := range entries {
		if e.prefix == bestPrefix && e.num > maxNumber {
			maxNumber = e.num
		}
	}

	return fmt.Sprintf("%s%s%d", bestPrefix, bestSep, maxNumber+1)
}

func (s *Handler) QueryPartialCardByID(userID, id int) (models.PartialCard, error) {
	return services.GetPartialCard(s.GetDB(), userID, id)
}

func (s *Handler) QueryPartialCard(userID int, cardID string) (models.PartialCard, error) {
	return services.GetPartialCardByCardID(s.GetDB(), userID, cardID)

}

func (s *Handler) QueryFullCard(userID int, id int) (models.Card, error) {
	s.logCardView(id, userID)
	return services.GetFullCard(s.GetDB(), userID, id)
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

	rows, err := s.GetDB().Query(query, userID)
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

	err := s.GetDB().QueryRow(query, id, userID).Scan(
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

	err := s.GetDB().QueryRow(query, userID, params.Name, params.Title, params.Body).Scan(
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

	err := s.GetDB().QueryRow(query, params.Name, params.Title, params.Body, id, userID).Scan(
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

	result, err := s.GetDB().Exec(query, id, userID)
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
	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
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

	rows, err := s.GetDB().Query(query, userID, perPage, offset)
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
	err = s.GetDB().QueryRow(countQuery, userID).Scan(&total)
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

	s.ProcessCardAfterCreation(userID, card, true)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(card)
}

// ProcessCardAfterCreation handles post-creation processing for a card
// This includes entity/fact extraction for PRO users
// Call this after any card creation to ensure consistent processing
func (s *Handler) ProcessCardAfterCreation(userID int, card models.Card, shouldProcess bool) {
	if !shouldProcess {
		return
	}
	if s.UserHasSubscription(userID) {
		s.ProcessEntitiesAndFacts(userID, card)
	}
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// GetRelatedCardsRoute returns cards related to the source card based on
// shared entities, shared tags, and semantic similarity
func (s *Handler) GetRelatedCardsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}
	if cardID <= 0 {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Get source card
	card, err := s.QueryFullCard(userID, cardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get cards by shared entities
	entityScores, err := services.GetCardsBySharedEntities(s.GetDB(), userID, cardID)
	if err != nil {
		log.Printf("Failed to get cards by shared entities: %v", err)
		http.Error(w, "Failed to find related cards", http.StatusInternalServerError)
		return
	}

	// Get cards by shared tags
	tagScores, err := services.GetCardsBySharedTags(s.GetDB(), userID, cardID)
	if err != nil {
		log.Printf("Failed to get cards by shared tags: %v", err)
		http.Error(w, "Failed to find related cards", http.StatusInternalServerError)
		return
	}

	// Get semantically similar cards (up to 20 for more candidate variety)
	var semanticScores []server.SimilarCard
	if s.Server != nil && s.Server.TypesenseClient != nil {
		ctx := r.Context()
		semanticScores, err = s.Server.FindSimilarCards(ctx, card, 20)
		if err != nil {
			log.Printf("Failed to find similar cards: %v", err)
			// Continue without semantic results rather than failing completely
			semanticScores = []server.SimilarCard{}
		}
	}

	// Merge scores from all sources
	combinedScores := make(map[int]float64)
	cardReasons := make(map[int][]string)

	// Add entity scores (each shared entity = 3 points)
	for cardID, score := range entityScores {
		combinedScores[cardID] += float64(score)
		cardReasons[cardID] = append(cardReasons[cardID], "entities")
	}

	// Add tag scores (each shared tag = 1 point)
	for cardID, score := range tagScores {
		combinedScores[cardID] += float64(score)
		cardReasons[cardID] = append(cardReasons[cardID], "tags")
	}

	// Add semantic scores (already 0-1 range)
	for _, sc := range semanticScores {
		// Scale semantic score (0-1) to 0-10 range for better balance with entity/tag scores
		scaledScore := sc.Score * 10.0
		combinedScores[sc.ID] += scaledScore
		cardReasons[sc.ID] = append(cardReasons[sc.ID], "similarity")
	}

	// Return early if no candidates
	if len(combinedScores) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]models.RelatedCard{})
		return
	}

	// Collect IDs to exclude
	excludeIDs := make(map[int]bool)
	excludeIDs[cardID] = true // Exclude current card
	if card.ParentID != nil {
		if *card.ParentID != cardID {
			excludeIDs[*card.ParentID] = true // Exclude parent
		}
	}

	// Get siblings (cards with same parent)
	if card.ParentID != nil {
		siblings, err := services.GetChildCards(s.GetDB(), userID, *card.ParentID)
		if err == nil {
			for _, sibling := range siblings {
				excludeIDs[sibling.ID] = true // Exclude siblings
			}
		}
	}

	// Get children (already includes cardID != childID check)
	children, err := services.GetChildCards(s.GetDB(), userID, cardID)
	if err == nil {
		for _, child := range children {
			excludeIDs[child.ID] = true // Exclude children
		}
	}

	// Get references and exclude them
	references, err := services.GetReferences(s.GetDB(), userID, card)
	if err == nil {
		for _, ref := range references {
			excludeIDs[ref.ID] = true // Exclude references
		}
	}

	// Build results, excluding specified cards
	var relatedCards []models.RelatedCard
	for relatedCardID, score := range combinedScores {
		if excludeIDs[relatedCardID] {
			continue // Skip excluded cards
		}

		// Get partial card with tags
		partialCard, err := services.GetPartialCard(s.GetDB(), userID, relatedCardID)
		if err != nil {
			log.Printf("Failed to get partial card %d: %v", relatedCardID, err)
			continue
		}

		// Fetch tags for this card
		tags, err := services.QueryTagsForCard(s.GetDB(), userID, relatedCardID)
		if err != nil {
			log.Printf("Failed to fetch tags for card %d: %v", relatedCardID, err)
			// Continue with empty tags
			partialCard.Tags = []models.Tag{}
		} else {
			partialCard.Tags = tags
		}

		// Deduplicate reasons
		uniqueReasons := make(map[string]bool)
		var reasons []string
		for _, reason := range cardReasons[relatedCardID] {
			if !uniqueReasons[reason] {
				uniqueReasons[reason] = true
				reasons = append(reasons, reason)
			}
		}

		relatedCards = append(relatedCards, models.RelatedCard{
			Card:    partialCard,
			Score:   score,
			Reasons: reasons,
		})
	}

	// Sort by score descending
	sort.Slice(relatedCards, func(i, j int) bool {
		return relatedCards[i].Score > relatedCards[j].Score
	})

	// Limit to top 10
	if len(relatedCards) > 10 {
		relatedCards = relatedCards[:10]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(relatedCards); err != nil {
		log.Printf("Failed to encode related cards: %v", err)
		return
	}
}
