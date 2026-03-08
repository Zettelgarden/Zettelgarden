package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// createTestSchemaForStructuredData is a helper to create a schema for testing
func createTestSchemaForStructuredData(s *Handler, t *testing.T, userID int, name string, fields []models.FieldDefinition) int {
	params := models.CreateSchemaDefinitionParams{
		Name:    name,
		OwnerID: userID,
		Fields:  fields,
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal schema params: %v", err)
	}

	token, _ := tests.GenerateTestJWT(userID)
	req, err := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("Failed to create test schema: %v - %v", status, rr.Body.String())
	}

	var schema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &schema)
	return schema.ID
}

// TestGetCardStructuredDataRoute tests retrieving structured data for a card
func TestGetCardStructuredDataRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema
	schemaFields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "count", Type: "number", Required: false},
	}
	schemaID := createTestSchemaForStructuredData(s, t, 1, "Test Schema", schemaFields)

	// Create a card with schema
	structuredDataJSON := `{"title":"Test Title","count":42}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-1",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Test getting structured data
	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("GET", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.GetCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response StructuredDataResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.SchemaID != schemaID {
		t.Errorf("Expected schema_id %d, got %d", schemaID, response.SchemaID)
	}
	if response.SchemaName != "Test Schema" {
		t.Errorf("Expected schema_name 'Test Schema', got '%s'", response.SchemaName)
	}
	if response.StructuredData == nil {
		t.Error("Expected structured_data to be set")
	}
}

// TestGetCardStructuredDataRoute_NoSchema tests retrieving structured data for a card without schema
func TestGetCardStructuredDataRoute_NoSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a card without schema (user 1, card ID 1 already exists from test data)
	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("GET", "/api/cards/1/structured-data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.GetCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response StructuredDataResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.SchemaID != 0 {
		t.Errorf("Expected no schema_id, got %d", response.SchemaID)
	}
	if response.StructuredData != nil {
		t.Error("Expected structured_data to be nil for card without schema")
	}
}

// TestUpdateCardStructuredDataRoute tests replacing structured data
func TestUpdateCardStructuredDataRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema
	schemaFields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "count", Type: "number", Required: false},
	}
	schemaID := createTestSchemaForStructuredData(s, t, 1, "Test Schema Update", schemaFields)

	// Create a card with initial structured data
	initialDataJSON := `{"title":"Initial Title","count":10}`
	var initialData json.RawMessage
	_ = json.Unmarshal([]byte(initialDataJSON), &initialData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-update",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schemaID,
		StructuredData: &initialData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Update structured data
	newDataJSON := `{"title":"Updated Title","count":99}`
	var newData json.RawMessage
	_ = json.Unmarshal([]byte(newDataJSON), &newData)

	updateReq := UpdateStructuredDataRequest{
		SchemaID:       &schemaID,
		StructuredData: &newData,
	}
	reqBody, _ := json.Marshal(updateReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.UpdateCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var updatedCard models.Card
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedCard); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify the structured data was updated
	if updatedCard.StructuredData == nil {
		t.Fatal("Expected structured_data to be set")
	}

	var resultData map[string]interface{}
	_ = json.Unmarshal(*updatedCard.StructuredData, &resultData)

	if resultData["title"] != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%v'", resultData["title"])
	}
	if resultData["count"] != float64(99) {
		t.Errorf("Expected count 99, got %v", resultData["count"])
	}
}

// TestUpdateCardStructuredDataRoute_RequiresSchemaID tests that schema_id is required when providing data
func TestUpdateCardStructuredDataRoute_RequiresSchemaID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to update structured data without schema_id (card ID 1 exists from test data)
	newDataJSON := `{"title":"Test"}`
	var newData json.RawMessage
	_ = json.Unmarshal([]byte(newDataJSON), &newData)

	updateReq := UpdateStructuredDataRequest{
		StructuredData: &newData,
		// SchemaID is nil
	}
	reqBody, _ := json.Marshal(updateReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PUT", "/api/cards/1/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.UpdateCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestPatchCardStructuredDataRoute tests merging structured data
func TestPatchCardStructuredDataRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema with multiple fields
	schemaFields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: false},
		{Name: "field2", Type: "number", Required: false},
		{Name: "field3", Type: "text", Required: false},
	}
	schemaID := createTestSchemaForStructuredData(s, t, 1, "Test Schema Patch", schemaFields)

	// Create a card with initial structured data
	initialDataJSON := `{"field1":"value1","field2":10,"field3":"unchanged"}`
	var initialData json.RawMessage
	_ = json.Unmarshal([]byte(initialDataJSON), &initialData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-patch",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schemaID,
		StructuredData: &initialData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Patch only field2
	patchDataJSON := `{"field2":99}`
	var patchData json.RawMessage
	_ = json.Unmarshal([]byte(patchDataJSON), &patchData)

	patchReq := PatchStructuredDataRequest{
		StructuredData: &patchData,
	}
	reqBody, _ := json.Marshal(patchReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PATCH", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.PatchCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var updatedCard models.Card
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedCard); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify the patch merged correctly
	var resultData map[string]interface{}
	_ = json.Unmarshal(*updatedCard.StructuredData, &resultData)

	// field1 should be unchanged
	if resultData["field1"] != "value1" {
		t.Errorf("Expected field1 'value1', got '%v'", resultData["field1"])
	}
	// field2 should be updated
	if resultData["field2"] != float64(99) {
		t.Errorf("Expected field2 99, got %v", resultData["field2"])
	}
	// field3 should be unchanged
	if resultData["field3"] != "unchanged" {
		t.Errorf("Expected field3 'unchanged', got '%v'", resultData["field3"])
	}
}

// TestDeleteCardStructuredDataRoute tests clearing structured data
func TestDeleteCardStructuredDataRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema
	schemaFields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: false},
	}
	schemaID := createTestSchemaForStructuredData(s, t, 1, "Test Schema Delete", schemaFields)

	// Create a card with schema
	structuredDataJSON := `{"title":"Test"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-delete",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Delete structured data
	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("DELETE", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.DeleteCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var updatedCard models.Card
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedCard); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify schema_id and structured_data are cleared
	if updatedCard.SchemaID != nil {
		t.Errorf("Expected schema_id to be nil, got %d", *updatedCard.SchemaID)
	}
	if updatedCard.StructuredData != nil {
		t.Error("Expected structured_data to be nil")
	}
}

// TestPatchCardStructuredDataRoute_NoExistingSchema tests that patch fails without existing schema
func TestPatchCardStructuredDataRoute_NoExistingSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Card ID 1 exists from test data without a schema
	// Try to patch structured data
	patchDataJSON := `{"field1":"test"}`
	var patchData json.RawMessage
	_ = json.Unmarshal([]byte(patchDataJSON), &patchData)

	patchReq := PatchStructuredDataRequest{
		StructuredData: &patchData,
	}
	reqBody, _ := json.Marshal(patchReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PATCH", "/api/cards/1/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.PatchCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestUpdateCardStructuredDataRoute_SchemaChangeWithoutData tests changing schema without providing new data
func TestUpdateCardStructuredDataRoute_SchemaChangeWithoutData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create first schema with field "title"
	schema1Fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
	}
	schema1ID := createTestSchemaForStructuredData(s, t, 1, "Schema One", schema1Fields)

	// Create second schema with different field "name"
	schema2Fields := []models.FieldDefinition{
		{Name: "name", Type: "text", Required: true},
	}
	schema2ID := createTestSchemaForStructuredData(s, t, 1, "Schema Two", schema2Fields)

	// Create a card with schema1 and data that validates against schema1
	initialDataJSON := `{"title":"Test Title"}`
	var initialData json.RawMessage
	_ = json.Unmarshal([]byte(initialDataJSON), &initialData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-schema-change",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schema1ID,
		StructuredData: &initialData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Try to change to schema2 without providing new data
	// The existing data {"title":"Test Title"} doesn't have "name" field which is required in schema2
	updateReq := UpdateStructuredDataRequest{
		SchemaID: &schema2ID,
		// StructuredData is nil - keep existing
	}
	reqBody, _ := json.Marshal(updateReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.UpdateCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	// Should fail because existing data doesn't validate against new schema
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 (data doesn't validate against new schema), got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestPatchCardStructuredDataRoute_InvalidLinkToCard tests that patch validates link_to_card references
func TestPatchCardStructuredDataRoute_InvalidLinkToCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a schema with link_to_card field
	schemaFields := []models.FieldDefinition{
		{Name: "related_card", Type: "link_to_card", Required: false},
	}
	schemaID := createTestSchemaForStructuredData(s, t, 1, "Test Schema Link", schemaFields)

	// Create a card with schema
	initialDataJSON := `{}`
	var initialData json.RawMessage
	_ = json.Unmarshal([]byte(initialDataJSON), &initialData)

	card, err := services.CreateCard(s.GetDB(), 1, models.EditCardParams{
		CardID:         "test-invalid-link",
		Title:          "Card with Schema",
		Body:           "Body content",
		SchemaID:       &schemaID,
		StructuredData: &initialData,
	})
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Patch with invalid link_to_card reference (non-existent card ID)
	patchDataJSON := `{"related_card":99999}`
	var patchData json.RawMessage
	_ = json.Unmarshal([]byte(patchDataJSON), &patchData)

	patchReq := PatchStructuredDataRequest{
		StructuredData: &patchData,
	}
	reqBody, _ := json.Marshal(patchReq)

	token, _ := tests.GenerateTestJWT(1)
	req := httptest.NewRequest("PATCH", "/api/cards/"+strconv.Itoa(card.ID)+"/structured-data", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(card.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/structured-data", s.JwtMiddleware(s.PatchCardStructuredDataRoute))
	router.ServeHTTP(rr, req)

	// Should fail because link_to_card references non-existent card
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 (invalid link_to_card reference), got %d: %s", rr.Code, rr.Body.String())
	}
}
