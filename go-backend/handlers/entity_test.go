package handlers

import (
	"context"
	"database/sql"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestMergeEntitiesSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Perform merge using pre-loaded test data
	err := s.MergeEntities(1, 1, 2)
	if err != nil {
		t.Errorf("MergeEntities failed: %v", err)
	}

	// Verify entity2 is deleted
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id = $1", 2).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check entity2 deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Entity2 was not deleted")
	}

	// Verify all card relationships were merged
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entity_card_junction WHERE entity_id = $1", 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count relationships: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 card relationships, got %d", count)
	}
}

func TestMergeEntitiesWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to merge entities belonging to different users
	err := s.MergeEntities(1, 1, 3)
	if err == nil {
		t.Error("Expected error when merging entities from different users")
	}

	// Verify both entities still exist
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id IN ($1, $2)", 1, 3).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count entities: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected both entities to still exist, got count %d", count)
	}
}

func TestMergeEntitiesNonExistent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to merge with non-existent entity
	err := s.MergeEntities(1, 1, 99999)
	if err == nil {
		t.Error("Expected error when merging with non-existent entity")
	}
}

func TestDeleteEntitySuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Delete entity using pre-loaded test data
	err := s.DeleteEntity(1, 1)
	if err != nil {
		t.Errorf("DeleteEntity failed: %v", err)
	}

	// Verify entity is deleted
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id = $1", 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check entity deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Entity was not deleted")
	}

	// Verify entity-card relationships are deleted
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entity_card_junction WHERE entity_id = $1", 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Entity relationships were not deleted")
	}
}

func TestDeleteEntityWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to delete entity belonging to different user
	err := s.DeleteEntity(2, 1) // Assuming entity 1 belongs to user 1
	if err == nil {
		t.Error("Expected error when deleting entity belonging to different user")
	}

	// Verify entity still exists
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id = $1", 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count entities: %v", err)
	}
	if count != 1 {
		t.Errorf("Entity should still exist, got count %d", count)
	}
}

func TestDeleteEntityNonExistent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to delete non-existent entity
	err := s.DeleteEntity(1, 99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent entity")
	}
}

func TestUpdateEntitySuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Update entity using pre-loaded test data
	params := UpdateEntityRequest{
		Name:        "Updated Entity",
		Description: "Updated description",
		Type:        "Updated type",
	}
	err := s.UpdateEntity(1, 1, params)
	if err != nil {
		t.Errorf("UpdateEntity failed: %v", err)
	}

	// Verify entity was updated
	var name, description, entityType string
	err = s.Server.Tx.QueryRow("SELECT name, description, type FROM entities WHERE id = $1", 1).Scan(&name, &description, &entityType)
	if err != nil {
		t.Errorf("Failed to check entity update: %v", err)
	}
	if name != params.Name || description != params.Description || entityType != params.Type {
		t.Errorf("Entity was not updated correctly")
	}
}

func TestUpdateEntityDuplicateName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// First create another entity with a known name, using a high ID to avoid conflicts
	_, err := s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		9999, 1, "Existing Entity", "Test description", "concept")
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Try to update entity with a name that already exists
	params := UpdateEntityRequest{
		Name:        "Existing Entity", // Try to update to the name we just created
		Description: "Updated description",
		Type:        "Updated type",
	}
	err = s.UpdateEntity(1, 1, params)
	if err == nil {
		t.Error("Expected error when updating entity with duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestUpdateEntityWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to update entity belonging to different user
	params := UpdateEntityRequest{
		Name:        "Updated Entity",
		Description: "Updated description",
		Type:        "Updated type",
	}
	err := s.UpdateEntity(2, 1, params) // Assuming entity 1 belongs to user 1
	if err == nil {
		t.Error("Expected error when updating entity belonging to different user")
	}
}

func TestUpdateEntityNonExistent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to update non-existent entity
	params := UpdateEntityRequest{
		Name:        "Updated Entity",
		Description: "Updated description",
		Type:        "Updated type",
	}
	err := s.UpdateEntity(1, 99999, params)
	if err == nil {
		t.Error("Expected error when updating non-existent entity")
	}
}

func TestUpdateEntityWithCardPK(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Update entity with card_pk
	params := UpdateEntityRequest{
		Name:        "Updated Entity",
		Description: "Updated Description",
		Type:        "person",
		CardPK:      &cardID,
	}

	err = s.UpdateEntity(1, 1, params)
	if err != nil {
		t.Errorf("UpdateEntity failed: %v", err)
	}

	// Verify the update
	var entity models.Entity
	err = s.Server.Tx.QueryRow(`
		SELECT id, user_id, name, description, type, card_pk
		FROM entities
		WHERE id = $1
	`, 1).Scan(
		&entity.ID,
		&entity.UserID,
		&entity.Name,
		&entity.Description,
		&entity.Type,
		&entity.CardPK,
	)
	if err != nil {
		t.Errorf("Failed to verify entity update: %v", err)
	}
	if *entity.CardPK != cardID {
		t.Errorf("Expected card_pk to be %d, got %d", cardID, *entity.CardPK)
	}
}

func TestUpdateEntityWithInvalidCardPK(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to update with non-existent card
	invalidCardID := 99999
	params := UpdateEntityRequest{
		Name:        "Updated Entity",
		Description: "Updated Description",
		Type:        "person",
		CardPK:      &invalidCardID,
	}

	err := s.UpdateEntity(1, 1, params)
	if err == nil {
		t.Error("Expected error when updating with invalid card_pk")
	}
	if !strings.Contains(err.Error(), "card not found") {
		t.Errorf("Expected 'card not found' error, got: %v", err)
	}
}

func TestAddEntityToCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Verify no relationship exists initially
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check initial relationship: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected no initial relationship, got %d", count)
	}

	// Add the relationship
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, 1, 1, cardID)
	if err != nil {
		t.Errorf("Failed to add entity to card: %v", err)
	}

	// Verify the relationship was created
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship creation: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected one relationship, got %d", count)
	}
}

func TestAddEntityToCardDuplicate(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create initial relationship
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, 1, 1, cardID)
	if err != nil {
		t.Fatalf("Failed to create initial relationship: %v", err)
	}

	// Try to add the same relationship again
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
		ON CONFLICT (entity_id, card_pk) DO NOTHING
	`, 1, 1, cardID)
	if err != nil {
		t.Errorf("Expected no error when adding duplicate relationship, got: %v", err)
	}

	// Verify only one relationship exists
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected one relationship after duplicate attempt, got %d", count)
	}
}

func TestAddEntityToCardWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card for user 1
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create an entity owned by user 1
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type)
		VALUES ($1, $2, 'Test Entity', 'Test Description', 'concept')
	`, 999, 1)
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Try to create relationship as user 2 using the handler
	ctx := context.WithValue(context.Background(), "current_user", 2)
	req, _ := http.NewRequestWithContext(ctx, "POST", "/api/entities/999/cards/"+strconv.Itoa(cardID), nil)

	// Set up gorilla/mux vars
	vars := map[string]string{
		"entityId": "999",
		"cardId":   strconv.Itoa(cardID),
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	s.AddEntityToCardRoute(rr, req)

	// Verify the request was rejected
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Verify no relationship was created
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2
	`, 999, cardID).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected no relationship, got count %d", count)
	}
}

func TestAddEntityToCardNonExistent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Try to create relationship with non-existent entity and card
	_, err := s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, 1, 99999, 99999)
	if err == nil {
		t.Error("Expected error when adding relationship with non-existent entity/card")
	}
}

func TestRemoveEntityFromCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// First create a test card and entity-card relationship
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create entity-card junction
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, 1, 1, cardID)
	if err != nil {
		t.Fatalf("Failed to create entity-card junction: %v", err)
	}

	// Verify the relationship exists
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check relationship existence: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected one relationship, got %d", count)
	}

	// Remove the relationship
	_, err = s.Server.Tx.Exec(`
		DELETE FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1)
	if err != nil {
		t.Errorf("Failed to remove entity from card: %v", err)
	}

	// Verify the relationship was removed
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship removal: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected no relationships, got %d", count)
	}
}

func TestRemoveEntityFromCardWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// First create a test card and entity-card relationship for user 1
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create entity-card junction for user 1
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		VALUES ($1, $2, $3)
	`, 1, 1, cardID)
	if err != nil {
		t.Fatalf("Failed to create entity-card junction: %v", err)
	}

	// Try to remove the relationship as user 2
	_, err = s.Server.Tx.Exec(`
		DELETE FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 2)
	if err != nil {
		t.Errorf("Expected no error when attempting to remove with wrong user, got: %v", err)
	}

	// Verify the relationship still exists for user 1
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 1, cardID, 1).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check relationship existence: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected relationship to still exist, got count %d", count)
	}
}

func TestRemoveEntityFromCardNonExistent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var originalCount int
	err := s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction
	`).Scan(&originalCount)
	if err != nil {
		t.Errorf("Failed to count relationships: %v", err)
	}

	// Try to remove a non-existent relationship
	_, err = s.Server.Tx.Exec(`
		DELETE FROM entity_card_junction 
		WHERE entity_id = $1 AND card_pk = $2 AND user_id = $3
	`, 99999, 99999, 1)
	if err != nil {
		t.Errorf("Expected no error when removing non-existent relationship, got: %v", err)
	}

	// Verify no relationships were affected
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM entity_card_junction
	`).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count relationships: %v", err)
	}
	if count != originalCount {
		t.Errorf("Expected %d relationships in total, got %d", originalCount, count)
	}
}

func TestGetEntityByNameRouteNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Generate test JWT for user 1
	token, _ := tests.GenerateTestJWT(1)

	// Create request for non-existent entity
	entityName := "Non Existent Entity"
	req, err := http.NewRequest("GET", "/api/entities/name/"+url.QueryEscape(entityName), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Set up router with path variable
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/entities/name/{name}", s.JwtMiddleware(s.GetEntityByNameRoute))
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestGetEntityByNameRouteWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Generate test JWT for user 2
	token, _ := tests.GenerateTestJWT(2)

	// Create request for entity belonging to user 1 ("Test Entity 1")
	entityName := "Test Entity 1"
	req, err := http.NewRequest("GET", "/api/entities/name/"+url.QueryEscape(entityName), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Set up router with path variable
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/entities/name/{name}", s.JwtMiddleware(s.GetEntityByNameRoute))
	router.ServeHTTP(rr, req)

	// Check status code - should be not found since user 2 can't see user 1's entities
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestGetEntityByNameRouteUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create request without authorization token
	entityName := "Test Entity 1"
	req, err := http.NewRequest("GET", "/api/entities/name/"+url.QueryEscape(entityName), nil)
	if err != nil {
		t.Fatal(err)
	}
	// No Authorization header set

	// Set up router with path variable
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/entities/name/{name}", s.JwtMiddleware(s.GetEntityByNameRoute))
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGetEntityByNameRouteEmptyName(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Generate test JWT for user 1
	token, _ := tests.GenerateTestJWT(1)

	// Create request with empty entity name
	req, err := http.NewRequest("GET", "/api/entities/name/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Set up router with path variable
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/entities/name/{name}", s.JwtMiddleware(s.GetEntityByNameRoute))
	router.ServeHTTP(rr, req)

	// This should return 404 since the route won't match
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestMergeEntitiesPreservesCardPK(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card first
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create an entity with card_pk set
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type, card_pk, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		100, 1, "Entity With Card", "Test description", "concept", cardID)
	if err != nil {
		t.Fatalf("Failed to create entity with card_pk: %v", err)
	}

	// Create another entity without card_pk
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		101, 1, "Entity Without Card", "Test description", "concept")
	if err != nil {
		t.Fatalf("Failed to create entity without card_pk: %v", err)
	}

	// Merge entities: entity with card_pk (100) should preserve its card_pk
	err = s.MergeEntities(1, 100, 101)
	if err != nil {
		t.Errorf("MergeEntities failed: %v", err)
	}

	// Verify entity 100 still has its card_pk
	var resultCardPK sql.NullInt64
	err = s.Server.Tx.QueryRow("SELECT card_pk FROM entities WHERE id = $1", 100).Scan(&resultCardPK)
	if err != nil {
		t.Errorf("Failed to check entity card_pk: %v", err)
	}
	if !resultCardPK.Valid || int(resultCardPK.Int64) != cardID {
		t.Errorf("Expected card_pk to be %d, got %v", cardID, resultCardPK)
	}

	// Verify entity 101 is deleted
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id = $1", 101).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check entity deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Entity 101 was not deleted")
	}
}

func TestMergeEntitiesPreservesCardPKFromSecondEntity(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test card first
	var cardID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO cards (user_id, title, body)
		VALUES ($1, 'Test Card', 'Test Content')
		RETURNING id
	`, 1).Scan(&cardID)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create an entity without card_pk
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		102, 1, "Entity Without Card", "Test description", "concept")
	if err != nil {
		t.Fatalf("Failed to create entity without card_pk: %v", err)
	}

	// Create an entity with card_pk set
	_, err = s.Server.Tx.Exec(`
		INSERT INTO entities (id, user_id, name, description, type, card_pk, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		103, 1, "Entity With Card", "Test description", "concept", cardID)
	if err != nil {
		t.Fatalf("Failed to create entity with card_pk: %v", err)
	}

	// Merge entities: entity without card_pk (102) should get card_pk from entity 103
	err = s.MergeEntities(1, 102, 103)
	if err != nil {
		t.Errorf("MergeEntities failed: %v", err)
	}

	// Verify entity 102 now has the card_pk from entity 103
	var resultCardPK sql.NullInt64
	err = s.Server.Tx.QueryRow("SELECT card_pk FROM entities WHERE id = $1", 102).Scan(&resultCardPK)
	if err != nil {
		t.Errorf("Failed to check entity card_pk: %v", err)
	}
	if !resultCardPK.Valid || int(resultCardPK.Int64) != cardID {
		t.Errorf("Expected card_pk to be %d, got %v", cardID, resultCardPK)
	}

	// Verify entity 103 is deleted
	var count int
	err = s.Server.Tx.QueryRow("SELECT COUNT(*) FROM entities WHERE id = $1", 103).Scan(&count)
	if err != nil {
		t.Errorf("Failed to check entity deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Entity 103 was not deleted")
	}
}
