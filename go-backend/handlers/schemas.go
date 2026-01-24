package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
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

// getUserID safely extracts the user ID from the request context
// Returns an error if the user is not authenticated or the context value is invalid
func getUserID(r *http.Request) (int, error) {
	userID, ok := r.Context().Value("current_user").(int)
	if !ok {
		return 0, fmt.Errorf("user not authenticated or invalid context")
	}
	return userID, nil
}

// verifySchemaOwnership checks if a schema exists, belongs to the user, and is not deleted
// Returns httpError, shouldContinue - if httpError is true, an error response was written
func verifySchemaOwnership(db *sql.DB, schemaID, userID int, w http.ResponseWriter) bool {
	checkQuery := `
		SELECT id, is_deleted FROM schema_definitions
		WHERE id = $1 AND owner_id = $2
	`
	var existingID int
	var isDeleted bool
	err := db.QueryRow(checkQuery, schemaID, userID).Scan(&existingID, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Schema not found", http.StatusNotFound)
		} else {
			log.Printf("Error checking schema ownership: %v", err)
			http.Error(w, "Error checking schema", http.StatusInternalServerError)
		}
		return false
	}

	if isDeleted {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return false
	}

	return true
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

// validateSchemaParams validates common schema parameters (name and fields)
func validateSchemaParams(name string, fields []models.FieldDefinition) error {
	// Validate name
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("name cannot exceed 255 characters")
	}

	// Validate fields
	if err := validateSchemaFields(fields); err != nil {
		return err
	}

	return nil
}

// validateCreateSchemaParams validates parameters for creating a schema
func validateCreateSchemaParams(params models.CreateSchemaDefinitionParams) error {
	return validateSchemaParams(params.Name, params.Fields)
}

// validateUpdateSchemaParams validates parameters for updating a schema
func validateUpdateSchemaParams(params models.UpdateSchemaDefinitionParams) error {
	return validateSchemaParams(params.Name, params.Fields)
}

// CreateSchemaRoute handles POST /api/schemas - Create a new schema
func (s *Handler) CreateSchemaRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
	userID, err := getUserID(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
	userID, err := getUserID(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
	userID, err := getUserID(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	// Validate that the ID in the request body matches the URL path parameter
	if params.ID != 0 && params.ID != id {
		log.Printf("ID mismatch: URL parameter %d vs request body %d", id, params.ID)
		http.Error(w, "Schema ID in request body does not match URL parameter", http.StatusBadRequest)
		return
	}

	// Validate parameters
	if err := validateUpdateSchemaParams(params); err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify the schema exists, belongs to the user, and is not deleted
	if !verifySchemaOwnership(s.DB, id, userID, w) {
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
	userID, err := getUserID(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid schema ID", http.StatusBadRequest)
		return
	}

	// Verify the schema exists, belongs to the user, and is not deleted
	if !verifySchemaOwnership(s.DB, id, userID, w) {
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
