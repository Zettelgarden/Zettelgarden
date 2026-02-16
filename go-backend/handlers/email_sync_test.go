package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestCreateEmailAccountRoute tests creating an email account
func TestCreateEmailAccountRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create request body
	emailAddr := "test@example.com"
	appPassword := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: emailAddr,
		AppPassword:  &appPassword,
	}
	body, _ := json.Marshal(params)

	// Create request
	req := httptest.NewRequest("POST", "/api/email/accounts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	h := s
	h.CreateEmailAccountRoute(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Parse response
	var account models.EmailAccount
	if err := json.NewDecoder(rr.Body).Decode(&account); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response
	if account.EmailAddress != "test@example.com" {
		t.Errorf("expected email address test@example.com, got %s", account.EmailAddress)
	}
	if account.UserID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, account.ID)
	}
	if account.IsActive != true {
		t.Errorf("expected account to be active")
	}
}

// TestListEmailAccountsRoute tests listing email accounts
func TestListEmailAccountsRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create an email account
	account := createTestEmailAccount(s, t, user.ID)

	// Create request
	req := httptest.NewRequest("GET", "/api/email/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	h := s
	h.ListEmailAccountsRoute(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var accounts []models.EmailAccount
	if err := json.NewDecoder(rr.Body).Decode(&accounts); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response
	if len(accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].ID != account.ID {
		t.Errorf("expected account ID %d, got %d", account.ID, accounts[0].ID)
	}
}

// TestGetEmailAccountRoute tests getting a specific email account
func TestGetEmailAccountRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create an email account
	account := createTestEmailAccount(s, t, user.ID)

	// Create request
	req := httptest.NewRequest("GET", "/api/email/accounts/"+intToString(account.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Use mux to set path variables
	r := mux.NewRouter()
	r.HandleFunc("/api/email/accounts/{id}", s.GetEmailAccountRoute)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	r.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var result models.EmailAccount
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response
	if result.ID != account.ID {
		t.Errorf("expected account ID %d, got %d", account.ID, result.ID)
	}
}

// TestDeleteEmailAccountRoute tests deleting an email account
func TestDeleteEmailAccountRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create an email account
	account := createTestEmailAccount(s, t, user.ID)

	// Create request
	req := httptest.NewRequest("DELETE", "/api/email/accounts/"+intToString(account.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Use mux to set path variables
	r := mux.NewRouter()
	r.HandleFunc("/api/email/accounts/{id}", s.DeleteEmailAccountRoute)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	r.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

// TestListEmailsRoute tests listing emails with filters
func TestListEmailsRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create request
	req := httptest.NewRequest("GET", "/api/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	h := s
	h.ListEmailsRoute(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var response struct {
		Emails []models.Email `json:"emails"`
		Total  int            `json:"total"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response (empty list is acceptable)
	if response.Total < 0 {
		t.Errorf("expected total >= 0, got %d", response.Total)
	}
}

// TestGetEmailRoute tests getting a specific email
func TestGetEmailRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create an email account and email
	account := createTestEmailAccount(s, t, user.ID)
	email := createTestEmail(s, t, user.ID, account.ID)

	// Create request
	req := httptest.NewRequest("GET", "/api/emails/"+intToString(email.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Use mux to set path variables
	r := mux.NewRouter()
	r.HandleFunc("/api/emails/{id}", s.GetEmailRoute)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	r.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var result models.Email
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response
	if result.ID != email.ID {
		t.Errorf("expected email ID %d, got %d", email.ID, result.ID)
	}
}

// TestGetEmailStatsRoute tests getting email statistics
func TestGetEmailStatsRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test user
	user := createTestUser(s, t)
	token, _ := tests.GenerateTestJWT(user.ID)

	// Create request
	req := httptest.NewRequest("GET", "/api/emails/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	h := s
	h.GetEmailStatsRoute(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var stats map[string]int
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response (empty stats is acceptable)
	if stats == nil {
		t.Errorf("expected stats map, got nil")
	}
}

// Helper functions

func createTestEmailAccount(s *Handler, t *testing.T, userID int) models.EmailAccount {
	emailAddr := "test@example.com"
	appPassword := "test-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: emailAddr,
		AppPassword:  &appPassword,
	}
	// Type assert to *sql.DB for EmailAccountService
	db := s.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)
	account, err := accountService.CreateEmailAccount(context.Background(), userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create test email account: %v", err)
	}
	return *account
}

func createTestEmail(s *Handler, t *testing.T, userID, accountID int) models.Email {
	subject := "Test Email"
	fromAddr := "sender@example.com"
	bodyText := "Test body"
	email := models.Email{
		UserID:         userID,
		EmailAccountID: &accountID,
		MessageID:      "test-message-id",
		Subject:        &subject,
		FromAddress:    &fromAddr,
		BodyText:       &bodyText,
		Status:         "unprocessed",
	}
	emailService := services.NewEmailService(s.GetDB())
	result, err := emailService.CreateEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("failed to create test email: %v", err)
	}
	return *result
}
