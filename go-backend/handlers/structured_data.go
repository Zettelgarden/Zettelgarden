package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// StructuredDataResponse represents the response for getting structured data
type StructuredDataResponse struct {
	SchemaID       int                `json:"schema_id,omitempty"`
	SchemaName     string             `json:"schema_name,omitempty"`
	SchemaSlug     string             `json:"schema_slug,omitempty"`
	StructuredData *json.RawMessage   `json:"structured_data,omitempty"`
}

// UpdateStructuredDataRequest is the request body for updating structured data
type UpdateStructuredDataRequest struct {
	SchemaID       *int             `json:"schema_id,omitempty"`
	StructuredData *json.RawMessage `json:"structured_data,omitempty"`
}

// PatchStructuredDataRequest is the request body for patching (merging) structured data
type PatchStructuredDataRequest struct {
	StructuredData *json.RawMessage `json:"structured_data,omitempty"`
}

// GetCardStructuredDataRoute returns the structured data for a card with schema information
func (s *Handler) GetCardStructuredDataRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Get the card
	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Build response
	response := StructuredDataResponse{
		StructuredData: card.StructuredData,
	}

	// If card has a schema, fetch schema details
	if card.SchemaID != nil {
		response.SchemaID = *card.SchemaID

		query := `SELECT name, slug FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		err := s.GetDB().QueryRow(query, *card.SchemaID, userID).Scan(&response.SchemaName, &response.SchemaSlug)
		if err != nil {
			log.Printf("Error fetching schema details: %v", err)
			// Continue without schema name - not critical
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateCardStructuredDataRoute replaces the structured data for a card
// This requires both schema_id and structured_data to be provided
func (s *Handler) UpdateCardStructuredDataRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify card exists
	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var params UpdateStructuredDataRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate: must provide schema_id if structured_data is provided
	if params.StructuredData != nil && params.SchemaID == nil {
		http.Error(w, "schema_id is required when providing structured_data", http.StatusBadRequest)
		return
	}

	// If no schema_id provided, keep existing
	schemaID := params.SchemaID
	if schemaID == nil {
		schemaID = card.SchemaID
	}

	// If structured_data is nil and no schema_id change, nothing to do
	if params.StructuredData == nil && params.SchemaID == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
		return
	}

	// Validate against schema if we have structured data
	structuredData := params.StructuredData
	if structuredData == nil && schemaID != nil && params.SchemaID != nil {
		// Schema changed but no data provided - keep existing data but validate against new schema
		structuredData = card.StructuredData
	}

	if schemaID != nil && structuredData != nil {
		// Fetch schema definition
		query := `SELECT id, name, slug, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
		schema, err := models.ScanSchemaDefinition(s.GetDB().QueryRow(query, *schemaID, userID))
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
		cleanedData, err := services.ValidateStructuredData(*structuredData, schema)
		if err != nil {
			log.Printf("Structured data validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid structured data: %v", err), http.StatusBadRequest)
			return
		}
		structuredData = &cleanedData

		// Validate link_to_card fields
		if err := services.ValidateLinkToCardFields(s.GetDB(), userID, *structuredData, schema); err != nil {
			log.Printf("Link to card validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid link_to_card reference: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Update only the structured data fields
	updatedCard, err := services.UpdateCardStructuredData(s.GetDB(), userID, id, schemaID, structuredData)
	if err != nil {
		log.Printf("Error updating structured data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCard)
}

// PatchCardStructuredDataRoute merges new data into existing structured data
// This allows partial updates without replacing the entire structured data object
func (s *Handler) PatchCardStructuredDataRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Get current card state
	card, err := s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Card must already have a schema to patch
	if card.SchemaID == nil {
		http.Error(w, "Card has no schema - use PUT to set structured data with a schema", http.StatusBadRequest)
		return
	}

	var params PatchStructuredDataRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if params.StructuredData == nil {
		http.Error(w, "structured_data is required", http.StatusBadRequest)
		return
	}

	// Fetch schema definition
	query := `SELECT id, name, slug, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
	schema, err := models.ScanSchemaDefinition(s.GetDB().QueryRow(query, *card.SchemaID, userID))
	if err != nil {
		log.Printf("Error fetching schema: %v", err)
		http.Error(w, "Failed to fetch schema definition", http.StatusInternalServerError)
		return
	}
	if schema == nil {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	// Merge existing data with new data
	mergedData, err := services.MergeStructuredData(s.GetDB(), userID, card.StructuredData, *params.StructuredData, schema)
	if err != nil {
		log.Printf("Error merging structured data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to merge structured data: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the merged result
	cleanedData, err := services.ValidateStructuredData(mergedData, schema)
	if err != nil {
		log.Printf("Structured data validation error: %v", err)
		http.Error(w, fmt.Sprintf("Invalid structured data: %v", err), http.StatusBadRequest)
		return
	}

	// Validate link_to_card fields
	if err := services.ValidateLinkToCardFields(s.GetDB(), userID, cleanedData, schema); err != nil {
		log.Printf("Link to card validation error: %v", err)
		http.Error(w, fmt.Sprintf("Invalid link_to_card reference: %v", err), http.StatusBadRequest)
		return
	}

	// Update with merged data
	updatedCard, err := services.UpdateCardStructuredData(s.GetDB(), userID, id, card.SchemaID, &cleanedData)
	if err != nil {
		log.Printf("Error updating structured data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCard)
}

// DeleteCardStructuredDataRoute clears the structured data and removes schema association
func (s *Handler) DeleteCardStructuredDataRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify card exists
	_, err = s.QueryFullCard(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Clear structured data
	updatedCard, err := services.UpdateCardStructuredData(s.GetDB(), userID, id, nil, nil)
	if err != nil {
		log.Printf("Error clearing structured data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCard)
}
