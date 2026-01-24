package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Valid field types for schema definitions
var validFieldTypes = map[string]bool{
	"text":          true,
	"number":        true,
	"date":          true,
	"boolean":       true,
	"select":        true,
	"multi-select":  true,
	"link_to_card":  true,
}

// validateFieldDefinition validates a single field definition
func validateFieldDefinition(field models.FieldDefinition) error {
	// Check field name
	if field.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	// Check field type
	if !validFieldTypes[field.Type] {
		return fmt.Errorf("invalid field type '%s': must be one of text, number, date, boolean, select, multi-select, link_to_card", field.Type)
	}

	// For select and multi-select types, options are required
	if (field.Type == "select" || field.Type == "multi-select") && len(field.Options) == 0 {
		return fmt.Errorf("field '%s' of type '%s' must have at least one option", field.Name, field.Type)
	}

	return nil
}

// validateSchemaFields validates that field names are unique and all fields are valid
func validateSchemaFields(fields []models.FieldDefinition) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields cannot be empty")
	}

	// Check for duplicate field names
	fieldNames := make(map[string]bool)
	for _, field := range fields {
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name '%s'", field.Name)
		}
		fieldNames[field.Name] = true

		// Validate individual field
		if err := validateFieldDefinition(field); err != nil {
			return err
		}
	}

	return nil
}

// validateCreateSchemaParams validates parameters for creating a schema
func validateCreateSchemaParams(params models.CreateSchemaDefinitionParams) error {
	// Validate name
	if params.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(params.Name) > 255 {
		return fmt.Errorf("name cannot exceed 255 characters")
	}

	// Validate fields
	if err := validateSchemaFields(params.Fields); err != nil {
		return err
	}

	return nil
}

// validateUpdateSchemaParams validates parameters for updating a schema
func validateUpdateSchemaParams(params models.UpdateSchemaDefinitionParams) error {
	// Validate name
	if params.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(params.Name) > 255 {
		return fmt.Errorf("name cannot exceed 255 characters")
	}

	// Validate fields
	if err := validateSchemaFields(params.Fields); err != nil {
		return err
	}

	return nil
}

// CreateSchemaRoute handles POST /api/schemas - Create a new schema
func (s *Handler) CreateSchemaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateSchemaDefinitionParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("Error decoding request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Set owner ID from context
	params.OwnerID = userID

	// Validate parameters
	if err := validateCreateSchemaParams(params); err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert fields to JSONB
	fieldsJSON, err := models.FieldsToJSONB(params.Fields)
	if err != nil {
		log.Printf("Error converting fields to JSONB: %v", err)
		http.Error(w, "Error processing fields", http.StatusInternalServerError)
		return
	}

	// Create schema
	query := `
		INSERT INTO schema_definitions (name, owner_id, fields, created_at, updated_at, is_deleted)
		VALUES ($1, $2, $3, NOW(), NOW(), FALSE)
		RETURNING id, name, owner_id, fields, created_at, updated_at, is_deleted
	`

	var schema models.SchemaDefinition
	var fieldsJSONB []byte

	err = s.DB.QueryRow(query, params.Name, params.OwnerID, fieldsJSON).Scan(
		&schema.ID,
		&schema.Name,
		&schema.OwnerID,
		&fieldsJSONB,
		&schema.CreatedAt,
		&schema.UpdatedAt,
		&schema.IsDeleted,
	)

	if err != nil {
		log.Printf("Error creating schema: %v", err)
		http.Error(w, "Error creating schema", http.StatusInternalServerError)
		return
	}

	// Unmarshal fields
	if len(fieldsJSONB) > 0 {
		if err := json.Unmarshal(fieldsJSONB, &schema.Fields); err != nil {
			log.Printf("Error unmarshaling fields JSON: %v", err)
			http.Error(w, "Error processing schema data", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schema)
}

// GetSchemasRoute handles GET /api/schemas - List all schemas for current user
func (s *Handler) GetSchemasRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	query := `
		SELECT id, name, owner_id, fields, created_at, updated_at, is_deleted
		FROM schema_definitions
		WHERE owner_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		log.Printf("Error querying schemas: %v", err)
		http.Error(w, "Error retrieving schemas", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schemas, err := models.ScanSchemaDefinitions(rows)
	if err != nil {
		log.Printf("Error scanning schemas: %v", err)
		http.Error(w, "Error processing schema data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schemas)
}

// GetSchemaRoute handles GET /api/schemas/{id} - Get a specific schema
func (s *Handler) GetSchemaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid schema ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, name, owner_id, fields, created_at, updated_at, is_deleted
		FROM schema_definitions
		WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE
	`

	schema, err := models.ScanSchemaDefinition(s.DB.QueryRow(query, id, userID))
	if err != nil {
		log.Printf("Error querying schema: %v", err)
		http.Error(w, "Error retrieving schema", http.StatusInternalServerError)
		return
	}

	if schema == nil {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// UpdateSchemaRoute handles PUT /api/schemas/{id} - Update a schema
func (s *Handler) UpdateSchemaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid schema ID", http.StatusBadRequest)
		return
	}

	var params models.UpdateSchemaDefinitionParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("Error decoding request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate parameters
	if err := validateUpdateSchemaParams(params); err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// First verify the schema exists and belongs to the user
	checkQuery := `
		SELECT id, is_deleted FROM schema_definitions
		WHERE id = $1 AND owner_id = $2
	`
	var existingID int
	var isDeleted bool
	err = s.DB.QueryRow(checkQuery, id, userID).Scan(&existingID, &isDeleted)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "Schema not found", http.StatusNotFound)
			return
		}
		log.Printf("Error checking schema ownership: %v", err)
		http.Error(w, "Error checking schema", http.StatusInternalServerError)
		return
	}

	if isDeleted {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	// Convert fields to JSONB
	fieldsJSON, err := models.FieldsToJSONB(params.Fields)
	if err != nil {
		log.Printf("Error converting fields to JSONB: %v", err)
		http.Error(w, "Error processing fields", http.StatusInternalServerError)
		return
	}

	// Update schema
	updateQuery := `
		UPDATE schema_definitions
		SET name = $1, fields = $2, updated_at = NOW()
		WHERE id = $3 AND owner_id = $4
		RETURNING id, name, owner_id, fields, created_at, updated_at, is_deleted
	`

	var schema models.SchemaDefinition
	var fieldsJSONB []byte

	err = s.DB.QueryRow(updateQuery, params.Name, fieldsJSON, id, userID).Scan(
		&schema.ID,
		&schema.Name,
		&schema.OwnerID,
		&fieldsJSONB,
		&schema.CreatedAt,
		&schema.UpdatedAt,
		&schema.IsDeleted,
	)

	if err != nil {
		log.Printf("Error updating schema: %v", err)
		http.Error(w, "Error updating schema", http.StatusInternalServerError)
		return
	}

	// Unmarshal fields
	if len(fieldsJSONB) > 0 {
		if err := json.Unmarshal(fieldsJSONB, &schema.Fields); err != nil {
			log.Printf("Error unmarshaling fields JSON: %v", err)
			http.Error(w, "Error processing schema data", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// DeleteSchemaRoute handles DELETE /api/schemas/{id} - Soft delete a schema
func (s *Handler) DeleteSchemaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid schema ID", http.StatusBadRequest)
		return
	}

	// Verify the schema exists and belongs to the user
	checkQuery := `
		SELECT id, is_deleted FROM schema_definitions
		WHERE id = $1 AND owner_id = $2
	`
	var existingID int
	var isDeleted bool
	err = s.DB.QueryRow(checkQuery, id, userID).Scan(&existingID, &isDeleted)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "Schema not found", http.StatusNotFound)
			return
		}
		log.Printf("Error checking schema ownership: %v", err)
		http.Error(w, "Error checking schema", http.StatusInternalServerError)
		return
	}

	if isDeleted {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	// Check how many cards are using this schema before deleting
	countQuery := `
		SELECT COUNT(*) FROM cards WHERE card_schema_id = $1 AND is_deleted = FALSE
	`
	var cardsAffected int
	err = s.DB.QueryRow(countQuery, id).Scan(&cardsAffected)
	if err != nil {
		log.Printf("Error counting cards using schema: %v", err)
		http.Error(w, "Error checking schema usage", http.StatusInternalServerError)
		return
	}

	// Soft delete by setting is_deleted = true
	deleteQuery := `
		UPDATE schema_definitions
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND owner_id = $2
	`

	result, err := s.DB.Exec(deleteQuery, id, userID)
	if err != nil {
		log.Printf("Error deleting schema: %v", err)
		http.Error(w, "Error deleting schema", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Error deleting schema", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	// Build response with warning if cards are affected
	response := map[string]interface{}{
		"deleted": true,
	}

	if cardsAffected > 0 {
		response["warning"] = fmt.Sprintf("Schema is being used by %d card(s). These cards will no longer display structured data.", cardsAffected)
		response["cards_affected"] = cardsAffected
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
