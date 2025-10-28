package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCredentialsEmpty(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Load credentials (should return empty since file doesn't exist)
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if creds.Profiles == nil {
		t.Error("Expected profiles map to be initialized")
	}

	if len(creds.Profiles) != 0 {
		t.Errorf("Expected empty profiles, got %d profiles", len(creds.Profiles))
	}
}

func TestSaveAndLoadCredentials(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create credentials
	expiry := time.Now().Add(24 * time.Hour)
	creds := &Credentials{
		Profiles: map[string]ProfileCredential{
			"default": {
				Endpoint:    "https://zettelgarden.com",
				Token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
				TokenExpiry: expiry,
			},
			"staging": {
				Endpoint:    "https://staging.zettelgarden.com",
				Token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.staging",
				TokenExpiry: expiry,
			},
		},
	}

	// Save credentials
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Verify credentials file was created with correct permissions
	credPath, _ := GetCredentialsPath()
	info, err := os.Stat(credPath)
	if err != nil {
		t.Fatalf("Credentials file not created: %v", err)
	}

	// Check permissions are 0600 (owner read/write only)
	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Expected permissions %v, got %v", expectedPerm, info.Mode().Perm())
	}

	// Load credentials back
	loadedCreds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	// Verify loaded credentials match saved credentials
	if len(loadedCreds.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(loadedCreds.Profiles))
	}

	defaultCred := loadedCreds.Profiles["default"]
	if defaultCred.Token != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test" {
		t.Errorf("Expected default token to match, got '%s'", defaultCred.Token)
	}

	if defaultCred.Endpoint != "https://zettelgarden.com" {
		t.Errorf("Expected default endpoint to match, got '%s'", defaultCred.Endpoint)
	}

	// Verify token expiry is preserved (within 1 second tolerance)
	if defaultCred.TokenExpiry.Sub(expiry).Abs() > time.Second {
		t.Errorf("Expected token expiry to match, got %v, expected %v", defaultCred.TokenExpiry, expiry)
	}
}

func TestGetToken(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set up credentials
	expiry := time.Now().Add(24 * time.Hour)
	creds := &Credentials{
		Profiles: map[string]ProfileCredential{
			"default": {
				Endpoint:    "https://zettelgarden.com",
				Token:       "valid-token",
				TokenExpiry: expiry,
			},
			"expired": {
				Endpoint:    "https://zettelgarden.com",
				Token:       "expired-token",
				TokenExpiry: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
			},
		},
	}
	SaveCredentials(creds)

	// Test getting valid token
	token, err := GetToken("default")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "valid-token" {
		t.Errorf("Expected 'valid-token', got '%s'", token)
	}

	// Test getting expired token
	_, err = GetToken("expired")
	if err == nil {
		t.Error("Expected error for expired token")
	}

	// Test getting non-existent profile
	_, err = GetToken("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent profile")
	}
}

func TestSetToken(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set a token
	expiry := time.Now().Add(24 * time.Hour)
	err := SetToken("default", "https://zettelgarden.com", "test-token", expiry)
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	// Retrieve and verify the token
	token, err := GetToken("default")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "test-token" {
		t.Errorf("Expected 'test-token', got '%s'", token)
	}

	// Update the token
	newExpiry := time.Now().Add(48 * time.Hour)
	err = SetToken("default", "https://zettelgarden.com", "updated-token", newExpiry)
	if err != nil {
		t.Fatalf("SetToken (update) failed: %v", err)
	}

	// Retrieve and verify the updated token
	token, err = GetToken("default")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "updated-token" {
		t.Errorf("Expected 'updated-token', got '%s'", token)
	}
}

func TestClearToken(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set up credentials
	expiry := time.Now().Add(24 * time.Hour)
	SetToken("default", "https://zettelgarden.com", "test-token", expiry)

	// Verify token exists
	token, err := GetToken("default")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "test-token" {
		t.Errorf("Expected token to exist")
	}

	// Clear the token
	err = ClearToken("default")
	if err != nil {
		t.Fatalf("ClearToken failed: %v", err)
	}

	// Verify token no longer exists
	_, err = GetToken("default")
	if err == nil {
		t.Error("Expected error after clearing token")
	}

	// Clearing non-existent token should not error
	err = ClearToken("nonexistent")
	if err != nil {
		t.Errorf("ClearToken should not error for non-existent profile: %v", err)
	}
}

func TestIsTokenValid(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set up credentials
	validExpiry := time.Now().Add(24 * time.Hour)
	expiredExpiry := time.Now().Add(-1 * time.Hour)

	SetToken("valid", "https://zettelgarden.com", "valid-token", validExpiry)
	SetToken("expired", "https://zettelgarden.com", "expired-token", expiredExpiry)

	// Test valid token
	if !IsTokenValid("valid") {
		t.Error("Expected token to be valid")
	}

	// Test expired token
	if IsTokenValid("expired") {
		t.Error("Expected token to be invalid (expired)")
	}

	// Test non-existent token
	if IsTokenValid("nonexistent") {
		t.Error("Expected token to be invalid (non-existent)")
	}
}

func TestGetAllProfiles(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Initially no profiles
	profiles, err := GetAllProfiles()
	if err != nil {
		t.Fatalf("GetAllProfiles failed: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(profiles))
	}

	// Add some profiles
	expiry := time.Now().Add(24 * time.Hour)
	SetToken("default", "https://zettelgarden.com", "token1", expiry)
	SetToken("staging", "https://staging.zettelgarden.com", "token2", expiry)
	SetToken("local", "http://localhost:8080", "token3", expiry)

	// Get all profiles
	profiles, err = GetAllProfiles()
	if err != nil {
		t.Fatalf("GetAllProfiles failed: %v", err)
	}
	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}

	// Verify all profiles are present (order doesn't matter)
	profileMap := make(map[string]bool)
	for _, p := range profiles {
		profileMap[p] = true
	}

	expectedProfiles := []string{"default", "staging", "local"}
	for _, expected := range expectedProfiles {
		if !profileMap[expected] {
			t.Errorf("Expected profile '%s' to be in list", expected)
		}
	}
}

func TestFilePermissionWarning(t *testing.T) {
	// Create a temporary directory for test credentials
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create credentials file with insecure permissions
	configDir := filepath.Join(tmpDir, ".config", "zettelgarden")
	os.MkdirAll(configDir, 0700)
	credPath := filepath.Join(configDir, "credentials.json")
	os.WriteFile(credPath, []byte(`{"profiles":{}}`), 0644) // Too permissive

	// Note: This test captures stderr output is complex in Go
	// The function will print a warning to stderr, which is the desired behavior
	// We're just testing that it doesn't error out
	_, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials should not fail due to permissions warning: %v", err)
	}
}
