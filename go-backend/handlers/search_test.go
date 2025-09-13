package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"
	"go-backend/models"
	"go-backend/server"

	"github.com/stretchr/testify/assert"
)

func TestSearchRoute_EntitySearch(t *testing.T) {
	// Initialize a new handler with a mock server
	// The DB connection is nil, so ClassicCardSearch will fail,
	// and we expect an internal server error. This tests the routing logic.
	s := &Handler{
		Server: &server.Server{},
	}

	// Create a request body for the test
	searchParams := SearchRequestParams{
		SearchTerm: "@[MyEntity]",
	}
	body, _ := json.Marshal(searchParams)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	// Add a user to the request context
	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Define a mock handler function to use in our test
	mockHandler := http.HandlerFunc(s.SearchRoute)

	// Serve the HTTP request to our mock handler
	mockHandler.ServeHTTP(rr, req)

	// Check the status code - we expect an error because the DB is not available
	assert.Equal(t, http.StatusInternalServerError, rr.Code, "Expected status Internal Server Error")
}

func TestSearchRoute_TagSearch(t *testing.T) {
	// Initialize a new handler with a mock server
	// The DB connection is nil, so ClassicCardSearch will fail,
	// and we expect an internal server error. This tests the routing logic.
	s := &Handler{
		Server: &server.Server{},
	}

	// Create a request body for the test
	searchParams := SearchRequestParams{
		SearchTerm: "#MyTag",
	}
	body, _ := json.Marshal(searchParams)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	// Add a user to the request context
	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Define a mock handler function to use in our test
	mockHandler := http.HandlerFunc(s.SearchRoute)

	// Serve the HTTP request to our mock handler
	mockHandler.ServeHTTP(rr, req)

	// Check the status code - we expect an error because the DB is not available
	assert.Equal(t, http.StatusInternalServerError, rr.Code, "Expected status Internal Server Error")
}

func TestConvertCardsToSearchResults(t *testing.T) {
	now := time.Now()
	cards := []models.Card{
		{
			ID:        1,
			CardID:    "card-1",
			Title:     "Test Card 1",
			Body:      "This is a test card.",
			ParentID:  0,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	searchResults := convertCardsToSearchResults(cards)

	assert.Equal(t, 1, len(searchResults))
	assert.Equal(t, "1", searchResults[0].ID)
	assert.Equal(t, "Test Card 1", searchResults[0].Title)
	assert.Equal(t, "card", searchResults[0].Type)
	assert.Equal(t, "This is a test card.", searchResults[0].Preview)

	metadata, ok := searchResults[0].Metadata.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "card-1", metadata["card_id"])
}
