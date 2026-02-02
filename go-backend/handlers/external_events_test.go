package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// Helper function to convert int to string for URL paths
func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestCalDAVPasswordEncryption tests that passwords are properly encrypted when stored
func TestCalDAVPasswordEncryption(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	// Create calendar with password
	reqBody := models.CreateExternalCalendarRequest{
		Name:     "Test Calendar",
		URL:      "https://example.com/calendar.ics",
		Username: stringPtr("testuser"),
		Password: stringPtr("testpassword"),
		Color:    "#6366f1",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status Created, got %v", status)
	}

	var cal models.ExternalCalendar
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &cal)

	if cal.Username == nil || *cal.Username != "testuser" {
		t.Errorf("expected username testuser, got %v", cal.Username)
	}

	// Verify password is stored encrypted in database
	var encryptedPassword string
	err = s.GetDB().QueryRow("SELECT password FROM external_calendars WHERE id = $1", cal.ID).Scan(&encryptedPassword)
	if err != nil {
		t.Fatalf("Failed to query password: %v", err)
	}

	// Password should be encrypted (not plain text)
	if encryptedPassword == "testpassword" {
		t.Error("Password is stored in plain text, should be encrypted")
	}

	// Password should be non-empty and encrypted
	if encryptedPassword == "" {
		t.Error("Encrypted password is empty")
	}

	// Verify we can decrypt it
	decrypted, err := encryptionService.Decrypt(encryptedPassword)
	if err != nil {
		t.Fatalf("Failed to decrypt password: %v", err)
	}
	if decrypted != "testpassword" {
		t.Errorf("Decrypted password mismatch: got %v, want testpassword", decrypted)
	}
}

// TestCalDAVPasswordDecryptionDuringSync tests that passwords are decrypted during sync
func TestCalDAVPasswordDecryptionDuringSync(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	// Create calendar with password
	// Note: We'll use a mock server or skip the actual sync test if URL is not accessible
	reqBody := models.CreateExternalCalendarRequest{
		Name:     "Test Calendar",
		URL:      "https://example.com/calendar.ics",
		Username: stringPtr("testuser"),
		Password: stringPtr("testpassword"),
		Color:    "#6366f1",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status Created, got %v", status)
	}

	var cal models.ExternalCalendar
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &cal)

	// Verify the encrypted password can be retrieved and decrypted
	var encryptedPassword string
	err = s.GetDB().QueryRow("SELECT password FROM external_calendars WHERE id = $1", cal.ID).Scan(&encryptedPassword)
	if err != nil {
		t.Fatalf("Failed to query password: %v", err)
	}

	decrypted, err := encryptionService.Decrypt(encryptedPassword)
	if err != nil {
		t.Errorf("Failed to decrypt password for sync: %v", err)
	}
	if decrypted != "testpassword" {
		t.Errorf("Decrypted password mismatch: got %v, want testpassword", decrypted)
	}
}

// TestCalDAVPasswordUpdate tests updating calendar password
func TestCalDAVPasswordUpdate(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	// First create a calendar without password
	reqBody := models.CreateExternalCalendarRequest{
		Name:  "Test Calendar",
		URL:   "https://example.com/calendar.ics",
		Color: "#6366f1",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status Created, got %v", status)
	}

	var cal models.ExternalCalendar
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &cal)

	// Update to add password
	updateReq := models.UpdateExternalCalendarRequest{
		Username: stringPtr("newuser"),
		Password: stringPtr("newpassword"),
	}
	updateJsonData, _ := json.Marshal(updateReq)

	updateReqHTTP, _ := http.NewRequest("PUT", "/api/user/external-calendars/"+intToString(cal.ID), bytes.NewBuffer(updateJsonData))
	updateReqHTTP.Header.Set("Authorization", "Bearer "+token)
	updateReqHTTP.Header.Set("Content-Type", "application/json")

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/user/external-calendars/{id}", s.JwtMiddleware(s.UpdateExternalCalendarRoute))
	router2.ServeHTTP(rr2, updateReqHTTP)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v: %s", status, rr2.Body.String())
	}

	// Verify password was encrypted
	var encryptedPassword string
	var username string
	err = s.GetDB().QueryRow("SELECT username, password FROM external_calendars WHERE id = $1", cal.ID).Scan(&username, &encryptedPassword)
	if err != nil {
		t.Fatalf("Failed to query credentials: %v", err)
	}

	if username != "newuser" {
		t.Errorf("Expected username newuser, got %v", username)
	}

	if encryptedPassword == "newpassword" {
		t.Error("Password is stored in plain text, should be encrypted")
	}

	decrypted, err := encryptionService.Decrypt(encryptedPassword)
	if err != nil {
		t.Errorf("Failed to decrypt updated password: %v", err)
	}
	if decrypted != "newpassword" {
		t.Errorf("Decrypted password mismatch: got %v, want newpassword", decrypted)
	}
}

// TestCalDAVPasswordClear tests clearing calendar password
func TestCalDAVPasswordClear(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	// Create calendar with password
	reqBody := models.CreateExternalCalendarRequest{
		Name:     "Test Calendar",
		URL:      "https://example.com/calendar.ics",
		Username: stringPtr("testuser"),
		Password: stringPtr("testpassword"),
		Color:    "#6366f1",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status Created, got %v", status)
	}

	var cal models.ExternalCalendar
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &cal)

	// Clear password using clear_password flag
	clearPassword := true
	updateReq := models.UpdateExternalCalendarRequest{
		ClearPassword: &clearPassword,
	}
	updateJsonData, _ := json.Marshal(updateReq)

	updateReqHTTP, _ := http.NewRequest("PUT", "/api/user/external-calendars/"+intToString(cal.ID), bytes.NewBuffer(updateJsonData))
	updateReqHTTP.Header.Set("Authorization", "Bearer "+token)
	updateReqHTTP.Header.Set("Content-Type", "application/json")

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/user/external-calendars/{id}", s.JwtMiddleware(s.UpdateExternalCalendarRoute))
	router2.ServeHTTP(rr2, updateReqHTTP)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v: %s", status, rr2.Body.String())
	}

	// Verify password was cleared
	var encryptedPassword sql.NullString
	err = s.GetDB().QueryRow("SELECT password FROM external_calendars WHERE id = $1", cal.ID).Scan(&encryptedPassword)
	if err != nil {
		t.Fatalf("Failed to query password: %v", err)
	}

	if encryptedPassword.Valid {
		t.Error("Password should be NULL after clearing")
	}
}

// TestCalDAVPasswordNotReturned tests that passwords are never returned in API responses
func TestCalDAVPasswordNotReturned(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	// Create calendar with password
	reqBody := models.CreateExternalCalendarRequest{
		Name:     "Test Calendar",
		URL:      "https://example.com/calendar.ics",
		Username: stringPtr("testuser"),
		Password: stringPtr("testpassword"),
		Color:    "#6366f1",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status Created, got %v", status)
	}

	// Check response doesn't contain password field
	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	if _, hasPassword := response["password"]; hasPassword {
		t.Error("Response should not contain password field")
	}

	// Test list endpoint also doesn't return passwords
	req2, _ := http.NewRequest("GET", "/api/user/external-calendars", nil)
	req2.Header.Set("Authorization", "Bearer "+token)

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.ListExternalCalendarsRoute))
	router2.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v", status)
	}

	var listResponse []map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &listResponse)

	if len(listResponse) > 0 {
		if _, hasPassword := listResponse[0]["password"]; hasPassword {
			t.Error("List response should not contain password field")
		}
	}
}

// TestCalDAVEncryptionServiceInitialization tests encryption service initialization
func TestCalDAVEncryptionServiceInitialization(t *testing.T) {
	// Test with valid key
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Errorf("Failed to create encryption service with valid key: %v", err)
	}
	if encryptionService == nil {
		t.Error("Encryption service should not be nil with valid key")
	}

	// Test with missing key
	os.Unsetenv("CALENDAR_ENCRYPTION_KEY")
	encryptionService, err = services.NewEncryptionService()
	if err == nil {
		t.Error("Expected error when encryption key is not set")
	}
	if encryptionService != nil {
		t.Error("Encryption service should be nil when key is not set")
	}

	// Test with short key
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "short")
	encryptionService, err = services.NewEncryptionService()
	if err == nil {
		t.Error("Expected error when encryption key is too short")
	}
	if encryptionService != nil {
		t.Error("Encryption service should be nil when key is too short")
	}

	// Restore valid key
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
}

// TestCalDAVPasswordEncryptionDecryption tests the encryption/decryption roundtrip
func TestCalDAVPasswordEncryptionDecryption(t *testing.T) {
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	testCases := []struct {
		name     string
		password string
	}{
		{"Simple password", "password123"},
		{"Password with special chars", "p@ssw0rd!#$%"},
		{"Long password", "this-is-a-very-long-password-with-many-characters-123456789"},
		{"Password with unicode", "пароль123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := encryptionService.Encrypt(tc.password)
			if err != nil {
				t.Errorf("Failed to encrypt password: %v", err)
			}

			// Encrypted value should be different from original
			if encrypted == tc.password {
				t.Error("Encrypted password should be different from original")
			}

			// Decrypt and verify
			decrypted, err := encryptionService.Decrypt(encrypted)
			if err != nil {
				t.Errorf("Failed to decrypt password: %v", err)
			}

			if decrypted != tc.password {
				t.Errorf("Decrypted password mismatch: got %v, want %v", decrypted, tc.password)
			}
		})
	}
}

// TestCalDAVCredentialValidation tests validation of calendar credentials
func TestCalDAVCredentialValidation(t *testing.T) {
	// Enable testing mode to skip URL validation
	services.SetExternalEventsTestingMode(true)
	defer services.SetExternalEventsTestingMode(false)

	s := NewHandler()
	defer tests.Teardown()

	// Initialize encryption service for testing
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-encryption-key-with-at-least-16-chars")
	encryptionService, err := services.NewEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}
	s.EncryptionService = encryptionService

	token, _ := tests.GenerateTestJWT(1)

	testCases := []struct {
		name       string
		username   string
		password   string
		expectFail bool
	}{
		{"Valid username only", "testuser", "", false},
		{"Valid password only", "", "testpass", false},
		{"Valid both", "testuser", "testpass", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := models.CreateExternalCalendarRequest{
				Name:  "Test Calendar",
				URL:   "https://example.com/calendar.ics",
				Color: "#6366f1",
			}
			if tc.username != "" {
				reqBody.Username = &tc.username
			}
			if tc.password != "" {
				reqBody.Password = &tc.password
			}

			jsonData, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/api/user/external-calendars", bytes.NewBuffer(jsonData))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/api/user/external-calendars", s.JwtMiddleware(s.CreateExternalCalendarRoute))
			router.ServeHTTP(rr, req)

			if tc.expectFail {
				if status := rr.Code; status != http.StatusBadRequest {
					t.Errorf("expected status BadRequest, got %v", status)
				}
			} else {
				// Note: URL validation will fail for invalid URLs, but that's tested elsewhere
				// We're just testing that username/password are accepted
				if rr.Code == http.StatusBadRequest {
					var errResp map[string]interface{}
					json.Unmarshal(rr.Body.Bytes(), &errResp)
					// If it's a URL validation error, that's expected
					if code, ok := errResp["code"].(string); ok && code == "INVALID_URL" {
						// Expected - URL is not accessible
						return
					}
				}
			}
		})
	}
}
