package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/tests"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

// Helper function to create a schema via API
func createSchema(t *testing.T, s *Handler, userID int, name string, fields []models.FieldDefinition) (*models.SchemaDefinition, *httptest.ResponseRecorder) {
	token, _ := tests.GenerateTestJWT(userID)

	params := models.CreateSchemaDefinitionParams{
		Name:   name,
		Fields: fields,
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	var schema models.SchemaDefinition
	if rr.Code == http.StatusCreated {
		tests.ParseJsonResponse(t, rr.Body.Bytes(), &schema)
		return &schema, rr
	}
	return nil, rr
}

// TestCreateSchemaRoute_Success tests successful schema creation with valid fields
func TestCreateSchemaRoute_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "status", Type: "select", Required: false, Options: []string{"todo", "in-progress", "done"}},
		{Name: "priority", Type: "number", Required: false},
	}

	schema, rr := createSchema(t, s, 1, "Task Schema", fields)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	if schema.Name != "Task Schema" {
		t.Errorf("Expected name 'Task Schema', got '%s'", schema.Name)
	}

	if schema.OwnerID != 1 {
		t.Errorf("Expected owner_id 1, got %d", schema.OwnerID)
	}

	if len(schema.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(schema.Fields))
	}
}

// TestCreateSchemaRoute_EmptyName tests error when name is empty
func TestCreateSchemaRoute_EmptyName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "name cannot be empty\n" {
		t.Errorf("Expected error message 'name cannot be empty', got '%s'", rr.Body.String())
	}
}

// TestCreateSchemaRoute_NameTooLong tests error when name exceeds 255 characters
func TestCreateSchemaRoute_NameTooLong(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
	}

	longName := string(make([]byte, 256))
	for i := range longName {
		longName = string(append([]byte(longName)[:i], 'a'))
	}

	params := models.CreateSchemaDefinitionParams{
		Name:   longName,
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "name cannot exceed 255 characters\n" {
		t.Errorf("Expected error message 'name cannot exceed 255 characters', got '%s'", rr.Body.String())
	}
}

// TestCreateSchemaRoute_EmptyFields tests error when fields array is empty
func TestCreateSchemaRoute_EmptyFields(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: []models.FieldDefinition{},
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "fields cannot be empty\n" {
		t.Errorf("Expected error message 'fields cannot be empty', got '%s'", rr.Body.String())
	}
}

// TestCreateSchemaRoute_DuplicateFieldNames tests error when field names are duplicated
func TestCreateSchemaRoute_DuplicateFieldNames(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "title", Type: "number", Required: false},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "duplicate field name 'title'\n" {
		t.Errorf("Expected error message 'duplicate field name 'title'', got '%s'", rr.Body.String())
	}
}

// TestCreateSchemaRoute_InvalidFieldType tests error when field type is invalid
func TestCreateSchemaRoute_InvalidFieldType(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "title", Type: "invalid_type", Required: true},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedMsg := "invalid field type 'invalid_type': must be one of text, number, date, boolean, select, multi-select, link_to_card\n"
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}

// TestCreateSchemaRoute_SelectWithoutOptions tests error when select type has no options
func TestCreateSchemaRoute_SelectWithoutOptions(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "status", Type: "select", Required: false},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "field 'status' of type 'select' must have at least one option\n" {
		t.Errorf("Expected error message 'field 'status' of type 'select' must have at least one option', got '%s'", rr.Body.String())
	}
}

// TestCreateSchemaRoute_AllValidFieldTypes tests successful creation with all valid field types
func TestCreateSchemaRoute_AllValidFieldTypes(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{
		{Name: "text_field", Type: "text", Required: true},
		{Name: "number_field", Type: "number", Required: false},
		{Name: "date_field", Type: "date", Required: false},
		{Name: "boolean_field", Type: "boolean", Required: false},
		{Name: "select_field", Type: "select", Required: false, Options: []string{"a", "b", "c"}},
		{Name: "multi_select_field", Type: "multi-select", Required: false, Options: []string{"x", "y", "z"}},
		{Name: "link_field", Type: "link_to_card", Required: false},
	}

	schema, rr := createSchema(t, s, 1, "Complete Schema", fields)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	if len(schema.Fields) != 7 {
		t.Errorf("Expected 7 fields, got %d", len(schema.Fields))
	}
}

// TestGetSchemasRoute_Success tests successful retrieval of user's schemas
func TestGetSchemasRoute_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schemas for user 1
	fields1 := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	createSchema(t, s, 1, "User1 Schema 1", fields1)
	createSchema(t, s, 1, "User1 Schema 2", fields1)

	// Create schema for user 2
	fields2 := []models.FieldDefinition{{Name: "description", Type: "text", Required: true}}
	createSchema(t, s, 2, "User2 Schema", fields2)

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("GET", "/api/schemas", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetSchemasRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var schemas []models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &schemas)

	if len(schemas) != 2 {
		t.Errorf("Expected 2 schemas for user 1, got %d", len(schemas))
	}

	// Verify all schemas belong to user 1
	for _, schema := range schemas {
		if schema.OwnerID != 1 {
			t.Errorf("Expected owner_id 1, got %d", schema.OwnerID)
		}
	}
}

// TestGetSchemasRoute_ExcludesDeleted tests that deleted schemas are not returned
func TestGetSchemasRoute_ExcludesDeleted(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	// Delete the schema
	token, _ := tests.GenerateTestJWT(1)
	deleteReq, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteReq.SetPathValue("id", strconv.Itoa(schema.ID))

	deleteRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(deleteRR, deleteReq)

	// Get schemas
	req, _ := http.NewRequest("GET", "/api/schemas", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetSchemasRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var schemas []models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &schemas)

	if len(schemas) != 0 {
		t.Errorf("Expected 0 schemas (deleted schema excluded), got %d", len(schemas))
	}
}

// TestGetSchemasRoute_EmptyArray tests that empty array is returned when no schemas exist
func TestGetSchemasRoute_EmptyArray(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("GET", "/api/schemas", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetSchemasRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var schemas []models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &schemas)

	if len(schemas) != 0 {
		t.Errorf("Expected 0 schemas, got %d", len(schemas))
	}
}

// TestGetSchemaRoute_Success tests successful retrieval of a specific schema
func TestGetSchemaRoute_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("GET", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.GetSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var retrievedSchema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &retrievedSchema)

	if retrievedSchema.ID != schema.ID {
		t.Errorf("Expected schema ID %d, got %d", schema.ID, retrievedSchema.ID)
	}

	if retrievedSchema.Name != "Test Schema" {
		t.Errorf("Expected name 'Test Schema', got '%s'", retrievedSchema.Name)
	}
}

// TestGetSchemaRoute_NotFound tests error when schema is not found
func TestGetSchemaRoute_NotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("GET", "/api/schemas/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "99999")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.GetSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected error message 'Schema not found', got '%s'", rr.Body.String())
	}
}

// TestGetSchemaRoute_DeletedSchema tests error when trying to get a deleted schema
func TestGetSchemaRoute_DeletedSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	// Delete the schema
	token, _ := tests.GenerateTestJWT(1)
	deleteReq, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteReq.SetPathValue("id", strconv.Itoa(schema.ID))

	deleteRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(deleteRR, deleteReq)

	// Try to get the deleted schema
	req, _ := http.NewRequest("GET", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.GetSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

// TestGetSchemaRoute_OtherUserSchema tests error when trying to get another user's schema
func TestGetSchemaRoute_OtherUserSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "User1 Schema", fields)

	token, _ := tests.GenerateTestJWT(2)

	req, _ := http.NewRequest("GET", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.GetSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

// TestUpdateSchemaRoute_UpdateName tests successful update of schema name
func TestUpdateSchemaRoute_UpdateName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Original Name", fields)

	token, _ := tests.GenerateTestJWT(1)

	updateFields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Updated Name",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var updatedSchema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &updatedSchema)

	if updatedSchema.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updatedSchema.Name)
	}
}

// TestUpdateSchemaRoute_UpdateFields tests successful update of schema fields
func TestUpdateSchemaRoute_UpdateFields(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	updateFields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "status", Type: "select", Required: false, Options: []string{"todo", "done"}},
	}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var updatedSchema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &updatedSchema)

	if len(updatedSchema.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(updatedSchema.Fields))
	}
}

// TestUpdateSchemaRoute_SchemaNotFound tests error when schema is not found
func TestUpdateSchemaRoute_SchemaNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Updated Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/99999", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "99999")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected error message 'Schema not found', got '%s'", rr.Body.String())
	}
}

// TestUpdateSchemaRoute_OtherUserSchema tests error when trying to update another user's schema
func TestUpdateSchemaRoute_OtherUserSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "User1 Schema", fields)

	token, _ := tests.GenerateTestJWT(2)

	updateFields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Hacked Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

// TestUpdateSchemaRoute_InvalidFieldType tests error when updating with invalid field type
func TestUpdateSchemaRoute_InvalidFieldType(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	updateFields := []models.FieldDefinition{{Name: "bad_field", Type: "invalid_type", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestDeleteSchemaRoute_UnusedSchema tests successful deletion of unused schema (no warning)
func TestDeleteSchemaRoute_UnusedSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Unused Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if response["deleted"] != true {
		t.Errorf("Expected deleted=true, got %v", response["deleted"])
	}

	if _, exists := response["warning"]; exists {
		t.Errorf("Expected no warning for unused schema, got %v", response["warning"])
	}

	// Verify schema is soft deleted
	var isDeleted bool
	err := s.Server.Tx.QueryRow("SELECT is_deleted FROM schema_definitions WHERE id = $1", schema.ID).Scan(&isDeleted)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	if !isDeleted {
		t.Errorf("Expected schema to be soft deleted (is_deleted=true), got false")
	}
}

// TestDeleteSchemaRoute_WithCards tests successful deletion of schema with cards (with warning)
func TestDeleteSchemaRoute_WithCards(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema
	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Used Schema", fields)

	// Create a card that uses the schema
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (card_id, user_id, title, body, link, created_at, updated_at, parent_id, card_schema_id)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6, $7)
		RETURNING id
	`, "test-card", 1, "Test Card", "Test Body", "test-link", 1, schema.ID).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}
	log.Printf("card added %v", cardID)

	// Clean up the test card - always run even if test fails
	t.Cleanup(func() {
		s.Server.Tx.Exec("DELETE FROM cards WHERE id = $1", cardID)
	})

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if response["deleted"] != true {
		t.Errorf("Expected deleted=true, got %v", response["deleted"])
	}

	if _, exists := response["warning"]; !exists {
		t.Errorf("Expected warning for schema with cards, got none")
	}

	cardsAffected, ok := response["cards_affected"].(float64)
	if !ok {
		t.Errorf("Expected cards_affected to be a number, got %v", response["cards_affected"])
	} else if cardsAffected != 1 {
		t.Errorf("Expected cards_affected=1, got %v", cardsAffected)
	}
}

// TestDeleteSchemaRoute_SchemaNotFound tests error when schema is not found
func TestDeleteSchemaRoute_SchemaNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("DELETE", "/api/schemas/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "99999")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected error message 'Schema not found', got '%s'", rr.Body.String())
	}
}

// TestDeleteSchemaRoute_OtherUserSchema tests error when trying to delete another user's schema
func TestDeleteSchemaRoute_OtherUserSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "User1 Schema", fields)

	token, _ := tests.GenerateTestJWT(2)

	req, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	// Verify schema still exists
	var isDeleted bool
	err := s.Server.Tx.QueryRow("SELECT is_deleted FROM schema_definitions WHERE id = $1", schema.ID).Scan(&isDeleted)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	if isDeleted {
		t.Errorf("Expected schema to not be deleted, but it was")
	}
}

// TestCreateSchemaRoute_MultiSelectWithoutOptions tests error when multi-select has no options
func TestCreateSchemaRoute_MultiSelectWithoutOptions(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "tags", Type: "multi-select", Required: false},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "field 'tags' of type 'multi-select' must have at least one option\n" {
		t.Errorf("Expected error message 'field 'tags' of type 'multi-select' must have at least one option', got '%s'", rr.Body.String())
	}
}

// TestUpdateSchemaRoute_EmptyName tests error when updating with empty name
func TestUpdateSchemaRoute_EmptyName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	updateFields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestCreateSchemaRoute_EmptyFieldName tests error when field name is empty
func TestCreateSchemaRoute_EmptyFieldName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{
		{Name: "", Type: "text", Required: true},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "field name cannot be empty\n" {
		t.Errorf("Expected error message 'field name cannot be empty', got '%s'", rr.Body.String())
	}
}

// TestGetSchemaRoute_InvalidID tests error when schema ID is invalid
// With slug support, non-numeric strings are now treated as slugs, so we get 404 instead of 400
func TestGetSchemaRoute_InvalidID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("GET", "/api/schemas/nonexistent-slug", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "nonexistent-slug")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.GetSchemaRoute))
	router.ServeHTTP(rr, req)

	// Non-numeric strings are treated as slugs, so a non-existent slug returns 404
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected error message 'Schema not found', got '%s'", rr.Body.String())
	}
}

// TestDeleteSchemaRoute_InvalidID tests error when schema ID is invalid
func TestDeleteSchemaRoute_InvalidID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("DELETE", "/api/schemas/invalid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "invalid")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "Invalid schema ID\n" {
		t.Errorf("Expected error message 'Invalid schema ID', got '%s'", rr.Body.String())
	}
}

// TestUpdateSchemaRoute_InvalidID tests error when schema ID is invalid
func TestUpdateSchemaRoute_InvalidID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Updated Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/invalid", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "invalid")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	if rr.Body.String() != "Invalid schema ID\n" {
		t.Errorf("Expected error message 'Invalid schema ID', got '%s'", rr.Body.String())
	}
}

// TestGetSchemasRoute_MultipleUsers tests that each user only sees their own schemas
func TestGetSchemasRoute_MultipleUsers(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// Create multiple schemas for user 1
	createSchema(t, s, 1, "User1 Schema A", fields)
	createSchema(t, s, 1, "User1 Schema B", fields)

	// Create multiple schemas for user 2
	createSchema(t, s, 2, "User2 Schema A", fields)
	createSchema(t, s, 2, "User2 Schema B", fields)
	createSchema(t, s, 2, "User2 Schema C", fields)

	// User 1 should only see their 2 schemas
	token1, _ := tests.GenerateTestJWT(1)
	req1, _ := http.NewRequest("GET", "/api/schemas", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)

	rr1 := httptest.NewRecorder()
	handler1 := http.HandlerFunc(s.JwtMiddleware(s.GetSchemasRoute))
	handler1.ServeHTTP(rr1, req1)

	var schemas1 []models.SchemaDefinition
	tests.ParseJsonResponse(t, rr1.Body.Bytes(), &schemas1)

	if len(schemas1) != 2 {
		t.Errorf("Expected 2 schemas for user 1, got %d", len(schemas1))
	}

	for _, schema := range schemas1 {
		if schema.OwnerID != 1 {
			t.Errorf("User 1 should only see their own schemas, got owner_id %d", schema.OwnerID)
		}
	}

	// User 2 should only see their 3 schemas
	token2, _ := tests.GenerateTestJWT(2)
	req2, _ := http.NewRequest("GET", "/api/schemas", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)

	rr2 := httptest.NewRecorder()
	handler2 := http.HandlerFunc(s.JwtMiddleware(s.GetSchemasRoute))
	handler2.ServeHTTP(rr2, req2)

	var schemas2 []models.SchemaDefinition
	tests.ParseJsonResponse(t, rr2.Body.Bytes(), &schemas2)

	if len(schemas2) != 3 {
		t.Errorf("Expected 3 schemas for user 2, got %d", len(schemas2))
	}

	for _, schema := range schemas2 {
		if schema.OwnerID != 2 {
			t.Errorf("User 2 should only see their own schemas, got owner_id %d", schema.OwnerID)
		}
	}
}

// TestDeleteSchemaRoute_MultipleCardsWithSchema tests warning with multiple affected cards
func TestDeleteSchemaRoute_MultipleCardsWithSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema
	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Shared Schema", fields)

	// Create multiple cards that use the schema
	for i := 1; i <= 3; i++ {
		_, err := s.Server.Tx.Exec(`
			INSERT INTO cards (card_id, user_id, title, body, link, created_at, updated_at, parent_id, card_schema_id)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6, $7)
		`, "test-card-"+strconv.Itoa(i), 1, "Test Card "+strconv.Itoa(i), "Test Body", "test-link", 1, schema.ID)
		if err != nil {
			t.Fatalf("Failed to create test card: %v", err)
		}
	}

	token, _ := tests.GenerateTestJWT(1)

	req, _ := http.NewRequest("DELETE", "/api/schemas/"+strconv.Itoa(schema.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.DeleteSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	cardsAffected, ok := response["cards_affected"].(float64)
	if !ok {
		t.Errorf("Expected cards_affected to be a number, got %v", response["cards_affected"])
	} else if cardsAffected != 3 {
		t.Errorf("Expected cards_affected=3, got %v", cardsAffected)
	}

	// Verify warning message contains the count
	warning, ok := response["warning"].(string)
	if !ok {
		t.Errorf("Expected warning to be a string, got %v", response["warning"])
	}
	log.Printf("Warning message: %s", warning)
	if warning == "" {
		t.Errorf("Expected warning message for schema with cards, got empty string")
	}
}

// TestCreateSchemaRoute_DuplicateName tests error when creating a schema with a duplicate name
func TestCreateSchemaRoute_DuplicateName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// Create first schema
	schema1, rr1 := createSchema(t, s, 1, "Duplicate Test Schema", fields)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("Failed to create first schema: %d - %s", rr1.Code, rr1.Body.String())
	}
	if schema1 == nil {
		t.Fatalf("First schema creation returned nil")
	}

	// Try to create second schema with same name
	token, _ := tests.GenerateTestJWT(1)
	params := models.CreateSchemaDefinitionParams{
		Name:   "Duplicate Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected status 409 (Conflict), got %d. Body: %s", rr.Code, rr.Body.String())
	}

	expectedMsg := "A schema with this name already exists\n"
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}

// TestCreateSchemaRoute_DuplicateNameDifferentUsers tests that duplicate names are allowed for different users
func TestCreateSchemaRoute_DuplicateNameDifferentUsers(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// User 1 creates a schema
	_, rr1 := createSchema(t, s, 1, "Shared Schema Name", fields)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("Failed to create schema for user 1: %d - %s", rr1.Code, rr1.Body.String())
	}

	// User 2 should be able to create a schema with the same name
	schema2, rr2 := createSchema(t, s, 2, "Shared Schema Name", fields)
	if rr2.Code != http.StatusCreated {
		t.Errorf("User 2 should be able to create schema with same name, got status %d. Body: %s", rr2.Code, rr2.Body.String())
	}

	if schema2 == nil {
		t.Fatal("Second schema creation returned nil")
	}

	if schema2.Name != "Shared Schema Name" {
		t.Errorf("Expected name 'Shared Schema Name', got '%s'", schema2.Name)
	}

	if schema2.OwnerID != 2 {
		t.Errorf("Expected owner_id 2, got %d", schema2.OwnerID)
	}
}

// TestCreateSchemaRoute_DuplicateNameCaseInsensitive tests that duplicate names are case-insensitive
func TestCreateSchemaRoute_DuplicateNameCaseInsensitive(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// Create first schema with lowercase name
	_, rr1 := createSchema(t, s, 1, "test schema", fields)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("Failed to create first schema: %d - %s", rr1.Code, rr1.Body.String())
	}

	// Try to create second schema with uppercase version of same name
	token, _ := tests.GenerateTestJWT(1)
	params := models.CreateSchemaDefinitionParams{
		Name:   "TEST SCHEMA",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected status 409 (Conflict) for case-insensitive duplicate, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// TestCreateSchemaRoute_DuplicateNameWithWhitespace tests that duplicate names ignore surrounding whitespace
func TestCreateSchemaRoute_DuplicateNameWithWhitespace(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// Create first schema
	_, rr1 := createSchema(t, s, 1, "My Schema", fields)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("Failed to create first schema: %d - %s", rr1.Code, rr1.Body.String())
	}

	// Try to create second schema with same name but extra whitespace
	token, _ := tests.GenerateTestJWT(1)
	params := models.CreateSchemaDefinitionParams{
		Name:   "  My Schema  ",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected status 409 (Conflict) for name with whitespace, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// TestUpdateSchemaRoute_DuplicateName tests error when updating a schema to a name that already exists
func TestUpdateSchemaRoute_DuplicateName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// Create two schemas with different names
	_, _ = createSchema(t, s, 1, "Existing Schema", fields)
	schema2, _ := createSchema(t, s, 1, "Target Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Try to update schema2 to have the same name as schema1
	updateFields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Existing Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema2.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema2.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected status 409 (Conflict), got %d. Body: %s", rr.Code, rr.Body.String())
	}

	expectedMsg := "A schema with this name already exists\n"
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}

// TestUpdateSchemaRoute_DuplicateNameDifferentUsers tests that updating to a duplicate name is allowed for different users
func TestUpdateSchemaRoute_DuplicateNameDifferentUsers(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}

	// User 1 creates a schema
	_, _ = createSchema(t, s, 1, "Shared Schema Name", fields)

	// User 2 creates a schema
	schema2, _ := createSchema(t, s, 2, "Different Name", fields)

	// User 2 should be able to update their schema to the same name as user 1's schema
	token, _ := tests.GenerateTestJWT(2)
	updateFields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Shared Schema Name",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema2.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema2.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("User 2 should be able to update to name used by user 1, got status %d. Body: %s", rr.Code, rr.Body.String())
	}

	var updatedSchema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &updatedSchema)

	if updatedSchema.Name != "Shared Schema Name" {
		t.Errorf("Expected name 'Shared Schema Name', got '%s'", updatedSchema.Name)
	}
}

// TestUpdateSchemaRoute_SameName tests that updating a schema to its own name succeeds
func TestUpdateSchemaRoute_SameName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Update schema with same name but different fields
	updateFields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "status", Type: "select", Required: false, Options: []string{"active", "inactive"}},
	}
	params := models.UpdateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 when updating schema with same name, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var updatedSchema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &updatedSchema)

	if len(updatedSchema.Fields) != 2 {
		t.Errorf("Expected 2 fields after update, got %d", len(updatedSchema.Fields))
	}
}

// TestCreateSchemaRoute_ExceedsMaxFields tests error when schema exceeds maximum fields
func TestCreateSchemaRoute_ExceedsMaxFields(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create fields array exceeding MaxFieldsPerSchema (50)
	fields := make([]models.FieldDefinition, MaxFieldsPerSchema+1)
	for i := 0; i < len(fields); i++ {
		fields[i] = models.FieldDefinition{
			Name:     fmt.Sprintf("field_%d", i),
			Type:     "text",
			Required: false,
		}
	}

	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedMsg := fmt.Sprintf("schema exceeds maximum of %d fields (got %d)\n", MaxFieldsPerSchema, MaxFieldsPerSchema+1)
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}

// TestCreateSchemaRoute_ExceedsMaxOptions tests error when select field exceeds maximum options
func TestCreateSchemaRoute_ExceedsMaxOptions(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create options array exceeding MaxOptionsPerField (100)
	options := make([]string, MaxOptionsPerField+1)
	for i := 0; i < len(options); i++ {
		options[i] = fmt.Sprintf("option_%d", i)
	}

	fields := []models.FieldDefinition{
		{Name: "status", Type: "select", Required: false, Options: options},
	}
	params := models.CreateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: fields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedMsg := fmt.Sprintf("field 'status' exceeds maximum of %d options (got %d)\n", MaxOptionsPerField, MaxOptionsPerField+1)
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}

// TestUpdateSchemaRoute_ExceedsMaxFields tests error when update exceeds maximum fields
func TestUpdateSchemaRoute_ExceedsMaxFields(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{{Name: "title", Type: "text", Required: true}}
	schema, _ := createSchema(t, s, 1, "Test Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Create fields array exceeding MaxFieldsPerSchema (50)
	updateFields := make([]models.FieldDefinition, MaxFieldsPerSchema+1)
	for i := 0; i < len(updateFields); i++ {
		updateFields[i] = models.FieldDefinition{
			Name:     fmt.Sprintf("field_%d", i),
			Type:     "text",
			Required: false,
		}
	}

	params := models.UpdateSchemaDefinitionParams{
		Name:   "Test Schema",
		Fields: updateFields,
	}
	jsonData, _ := json.Marshal(params)

	req, _ := http.NewRequest("PUT", "/api/schemas/"+strconv.Itoa(schema.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(schema.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/schemas/{id}", s.JwtMiddleware(s.UpdateSchemaRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedMsg := fmt.Sprintf("schema exceeds maximum of %d fields (got %d)\n", MaxFieldsPerSchema, MaxFieldsPerSchema+1)
	if rr.Body.String() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, rr.Body.String())
	}
}
