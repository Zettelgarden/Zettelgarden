// +build ignore

// This file contains example test patterns that you can copy when writing new tests.
//
// It is excluded from compilation with the "ignore" build tag so it serves as
// documentation only. Copy the examples to appropriate locations:
//
//   - Handler tests → handlers/<feature>_test.go
//   - Service tests → services/<feature>_test.go
//
// The examples below show real patterns - just copy them to your test file
// and adjust imports/paths as needed.

package testsexamples

import (
	"bytes"
	"go-backend/models"
	"go-backend/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// ===========================================================================
// HANDLER TEST EXAMPLES
// Place these in handlers/<feature>_test.go
// ===========================================================================

// ExampleHandler_GetCard demonstrates a basic GET endpoint test
// COPY THIS to handlers/cards_test.go or similar
func ExampleHandler_GetCard(t *testing.T) {
	// Setup: Import handlers package and create handler
	//   "go-backend/handlers"
	//
	//   h := handlers.NewHandler()
	//   defer tests.Teardown()
	//
	// For this example, imagine we have a handler set up:
	//   h := &handlers.Handler{}  // In real code: handlers.NewHandler()

	// Arrange: Create test data using services
	//   userID := 1
	//   card, _ := services.CreateCard(h.GetDB(), userID, models.EditCardParams{
	//       Title:  "Example Card",
	//       Body:   "Example content",
	//       CardID: "example-get",
	//   })

	// Generate auth token
	//   token, _ := tests.GenerateTestJWT(userID)

	// Act: Create and execute HTTP request
	//   req, _ := http.NewRequest("GET", "/api/cards/"+card.CardID, nil)
	//   req.Header.Set("Authorization", "Bearer "+token)
	//
	//   rr := httptest.NewRecorder()
	//   router := mux.NewRouter()
	//   router.HandleFunc("/api/cards/{card_id}", h.JwtMiddleware(h.GetCardByCardIDRoute))
	//   router.ServeHTTP(rr, req)

	// Assert: Check response
	//   if status := rr.Code; status != http.StatusOK {
	//       t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	//   }
	//
	//   var response models.Card
	//   tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	//
	//   if response.Title != "Example Card" {
	//       t.Errorf("Expected title 'Example Card', got '%s'", response.Title)
	//   }

	_ = models.Card{}   // Remove when copying
	_ = services.CreateCard // Remove when copying
	_ = mux.NewRouter() // Remove when copying
	_ = http.StatusOK   // Remove when copying
}

// ExampleHandler_CreateCard demonstrates a POST endpoint test
// COPY THIS to handlers/cards_test.go or similar
func ExampleHandler_CreateCard(t *testing.T) {
	// Setup
	//   h := handlers.NewHandler()
	//   defer tests.Teardown()
	//
	//   userID := 1
	//   token, _ := tests.GenerateTestJWT(userID)

	// Create request body
	//   body := tests.CreateJsonBody(t, map[string]string{
	//       "title":  "New Card",
	//       "body":   "New content",
	//       "card_id": "new123",
	//   })
	//
	//   req, _ := http.NewRequest("POST", "/api/cards", body)
	//   req.Header.Set("Authorization", "Bearer "+token)
	//   req.Header.Set("Content-Type", "application/json")
	//
	//   rr := httptest.NewRecorder()
	//   router := mux.NewRouter()
	//   router.HandleFunc("/api/cards", h.JwtMiddleware(h.CreateCardRoute))
	//   router.ServeHTTP(rr, req)

	// Assert
	//   if status := rr.Code; status != http.StatusCreated {
	//       t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
	//   }

	_ = bytes.NewReader([]byte{}) // Remove when copying
	_ = http.StatusCreated         // Remove when copying
}

// ExampleHandler_Unauthorized demonstrates testing unauthorized access
// COPY THIS to any handler test file
func ExampleHandler_Unauthorized(t *testing.T) {
	// Setup
	//   h := handlers.NewHandler()
	//   defer tests.Teardown()

	// Request without auth token
	//   req, _ := http.NewRequest("GET", "/api/cards/1", nil)
	//
	//   rr := httptest.NewRecorder()
	//   router := mux.NewRouter()
	//   router.HandleFunc("/api/cards/{card_id}", h.JwtMiddleware(h.GetCardByCardIDRoute))
	//   router.ServeHTTP(rr, req)

	// Should return 401 Unauthorized
	//   if status := rr.Code; status != http.StatusUnauthorized {
	//       t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, status)
	//   }

	_ = http.StatusUnauthorized // Remove when copying
}

// ExampleHandler_UpdateCard demonstrates a PUT endpoint test
// COPY THIS to handlers/cards_test.go
func ExampleHandler_UpdateCard(t *testing.T) {
	// Setup
	//   h := handlers.NewHandler()
	//   defer tests.Teardown()
	//
	//   userID := 1
	//
	//   // Create existing card
	//   card, _ := services.CreateCard(h.GetDB(), userID, models.EditCardParams{
	//       Title:  "Original Title",
	//       Body:   "Original content",
	//       CardID: "update-example",
	//   })
	//
	//   token, _ := tests.GenerateTestJWT(userID)

	// Update request
	//   updateBody := tests.CreateJsonBody(t, map[string]string{
	//       "title": "Updated Title",
	//   })
	//
	//   req, _ := http.NewRequest("PUT", "/api/cards/"+card.CardID, updateBody)
	//   req.Header.Set("Authorization", "Bearer "+token)
	//   req.Header.Set("Content-Type", "application/json")
	//
	//   rr := httptest.NewRecorder()
	//   router := mux.NewRouter()
	//   router.HandleFunc("/api/cards/{card_id}", h.JwtMiddleware(h.UpdateCardRoute))
	//   router.ServeHTTP(rr, req)

	// Assert
	//   if status := rr.Code; status != http.StatusOK {
	//       t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	//   }

	_ = http.StatusOK // Remove when copying
}

// ===========================================================================
// SERVICE TEST EXAMPLES
// Place these in services/<feature>_test.go
// ===========================================================================

// ExampleService_CreateCard demonstrates a basic service function test
// COPY THIS to services/cards_test.go or similar
func ExampleService_CreateCard(t *testing.T) {
	// Setup: Import tests package and get server
	//   "go-backend/tests"
	//
	//   s := tests.Setup()
	//   defer tests.Teardown()
	//
	//   userID := 1
	//   params := models.EditCardParams{
	//       Title:  "Service Test Card",
	//       Body:   "Service test content",
	//       CardID: "service-test-123",
	//   }

	// Execute: Call service function
	//   card, err := services.CreateCard(s.DB, userID, params)
	//   if err != nil {
	//       t.Fatalf("CreateCard failed: %v", err)
	//   }

	// Assert: Verify results
	//   if card.Title != params.Title {
	//       t.Errorf("Expected title '%s', got '%s'", params.Title, card.Title)
	//   }
	//
	//   if card.UserID != userID {
	//       t.Errorf("Expected userID %d, got %d", userID, card.UserID)
	//   }

	_ = t    // Remove when copying
	_ = s    // Remove when copying
}

// ExampleService_CreateCard_InvalidInput demonstrates error handling
// COPY THIS to services/cards_test.go
func ExampleService_CreateCard_InvalidInput(t *testing.T) {
	//   s := tests.Setup()
	//   defer tests.Teardown()
	//
	//   userID := 1
	//   params := models.EditCardParams{
	//       Title: "", // Invalid: empty title
	//       Body:  "Some content",
	//   }
	//
	//   _, err := services.CreateCard(s.DB, userID, params)

	// Should return error for invalid input
	//   if err == nil {
	//       t.Error("Expected error for empty title, got nil")
	//   }

	_ = t // Remove when copying
}

// ExampleService_GetCard_NotFound demonstrates testing not-found error
// COPY THIS to services/cards_test.go
func ExampleService_GetCard_NotFound(t *testing.T) {
	//   s := tests.Setup()
	//   defer tests.Teardown()

	// Try to get a card that doesn't exist
	//   card, err := services.GetCardByCardID(s.DB, 1, "nonexistent-card")

	// Should return error
	//   if err == nil {
	//       t.Error("Expected error for non-existent card, got nil")
	//   }
	//
	//   if card != nil {
	//       t.Errorf("Expected nil card, got %+v", card)
	//   }

	_ = t // Remove when copying
}

// ExampleService_WithMultipleOperations demonstrates testing multi-step workflows
// COPY THIS to services/cards_test.go
func ExampleService_WithMultipleOperations(t *testing.T) {
	//   s := tests.Setup()
	//   defer tests.Teardown()
	//
	//   userID := 1

	// Step 1: Create a card
	//   card, _ := services.CreateCard(s.DB, userID, models.EditCardParams{
	//       Title:  "Workflow Card",
	//       Body:   "Initial content",
	//       CardID: "workflow-test",
	//   })

	// Step 2: Update the card
	//   updated, _ := services.UpdateCard(s.DB, userID, card.ID, models.EditCardParams{
	//       Title:  "Updated Workflow Card",
	//       Body:   "Updated content",
	//       CardID: "workflow-test",
	//   })

	// Step 3: Verify the update
	//   if updated.Title != "Updated Workflow Card" {
	//       t.Errorf("Expected updated title, got '%s'", updated.Title)
	//   }

	// Step 4: Delete the card
	//   services.DeleteCard(s.DB, userID, card.ID)

	// Step 5: Verify deletion
	//   _, err := services.GetCardByCardID(s.DB, userID, "workflow-test")
	//   if err == nil {
	//       t.Error("Expected error after deletion, got nil")
	//   }

	_ = t // Remove when copying
}

// ===========================================================================
// TABLE-DRIVEN TEST EXAMPLE
// Can be used in either handler or service tests
// ===========================================================================

// Example_TableDrivenTests demonstrates testing multiple cases
// COPY THIS pattern to any test file
func Example_TableDrivenTests(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name    string
		cardID  string
		wantErr bool
	}{
		{"valid card ID", "valid123", false},
		{"empty card ID", "", true},
		{"card ID with spaces", "invalid 123", true},
		{"card ID with special chars", "invalid@123", true},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//   s := tests.Setup()
			//   defer tests.Teardown()
			//
			//   userID := 1
			//   params := models.EditCardParams{
			//       Title:  "Test Card",
			//       Body:   "Test content",
			//       CardID: tc.cardID,
			//   }
			//
			//   _, err := services.CreateCard(s.DB, userID, params)
			//
			//   if (err != nil) != tc.wantErr {
			//       t.Errorf("CreateCard() error = %v, wantErr %v", err, tc.wantErr)
			//   }
		})
	}
	_ = t // Remove when copying
}

// ===========================================================================
// AUTHORIZATION TESTING EXAMPLE
// ===========================================================================

// ExampleAuthorization_UserIsolation demonstrates testing user isolation
// COPY THIS to handlers/cards_test.go
func ExampleAuthorization_UserIsolation(t *testing.T) {
	//   h := handlers.NewHandler()
	//   defer tests.Teardown()
	//
	//   router := mux.NewRouter()
	//   router.HandleFunc("/api/cards", h.JwtMiddleware(h.CreateCardRoute))
	//   router.HandleFunc("/api/cards/{card_id}", h.JwtMiddleware(h.UpdateCardRoute))

	// User 1 creates a card
	//   user1Token, _ := tests.GenerateTestJWT(1)
	//   createBody := tests.CreateJsonBody(t, map[string]string{
	//       "title":  "User 1 Card",
	//       "body":   "User 1 content",
	//       "card_id": "user1-private",
	//   })
	//
	//   req, _ := http.NewRequest("POST", "/api/cards", createBody)
	//   req.Header.Set("Authorization", "Bearer "+user1Token)
	//   req.Header.Set("Content-Type", "application/json")
	//
	//   rr := httptest.NewRecorder()
	//   router.ServeHTTP(rr, req)
	//
	//   var createdCard models.Card
	//   tests.ParseJsonResponse(t, rr.Body.Bytes(), &createdCard)

	// User 2 tries to update User 1's card (should fail)
	//   user2Token, _ := tests.GenerateTestJWT(2)
	//   updateBody := tests.CreateJsonBody(t, map[string]string{
	//       "title": "Hacked Title",
	//   })
	//
	//   updateReq, _ := http.NewRequest("PUT", "/api/cards/"+createdCard.CardID, updateBody)
	//   updateReq.Header.Set("Authorization", "Bearer "+user2Token)
	//   updateReq.Header.Set("Content-Type", "application/json")
	//
	//   updateRr := httptest.NewRecorder()
	//   router.ServeHTTP(updateRr, updateReq)

	// Should return 403 Forbidden or 404 Not Found
	//   status := updateRr.Code
	//   if status != http.StatusForbidden && status != http.StatusNotFound {
	//       t.Errorf("Expected status 403 or 404 for unauthorized update, got %d", status)
	//   }

	_ = t    // Remove when copying
	_ = http.StatusForbidden
	_ = http.StatusNotFound
}

// ===========================================================================
// INTEGRATION TEST EXAMPLE
// ===========================================================================

// ExampleIntegration_Backlinks demonstrates an integration test
// COPY THIS to services/references_test.go
func ExampleIntegration_Backlinks(t *testing.T) {
	//   s := tests.Setup()
	//   defer tests.Teardown()
	//
	//   userID := 1

	// Create target card
	//   target, _ := services.CreateCard(s.DB, userID, models.EditCardParams{
	//       Title:  "Target Card",
	//       Body:   "Target content",
	//       CardID: "backlink-target",
	//   })

	// Create source card that references target
	//   source, _ := services.CreateCard(s.DB, userID, models.EditCardParams{
	//       Title:  "Source Card",
	//       Body:   "This references [backlink-target]",
	//       CardID: "backlink-source",
	//   })

	// Get direct links from source
	//   directLinks, _ := services.GetDirectLinks(s.DB, userID, source)

	// Verify the backlink
	//   if len(directLinks) != 1 {
	//       t.Fatalf("Expected 1 direct link, got %d", len(directLinks))
	//   }
	//
	//   if directLinks[0].ID != target.ID {
	//       t.Errorf("Expected target ID %d, got %d", target.ID, directLinks[0].ID)
	//   }

	_ = t // Remove when copying
}
