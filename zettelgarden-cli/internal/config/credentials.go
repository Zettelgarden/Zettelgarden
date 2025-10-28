package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProfileCredential represents authentication credentials for a specific profile
type ProfileCredential struct {
	Endpoint    string    `json:"endpoint"`
	Token       string    `json:"token"`
	TokenExpiry time.Time `json:"token_expiry"`
}

// Credentials represents all stored authentication credentials
type Credentials struct {
	Profiles map[string]ProfileCredential `json:"profiles"`
}

// GetCredentialsPath returns the full path to the credentials file
func GetCredentialsPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// LoadCredentials loads credentials from file or returns empty credentials
func LoadCredentials() (*Credentials, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}

	// If credentials file doesn't exist, return empty credentials
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return &Credentials{
			Profiles: make(map[string]ProfileCredential),
		}, nil
	}

	// Check file permissions - warn if too permissive
	info, err := os.Stat(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat credentials file: %w", err)
	}

	// Check if file is readable by group or others (on Unix systems)
	if info.Mode().Perm()&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: credentials file has insecure permissions (%v). Run 'chmod 600 %s' to fix.\n",
			info.Mode().Perm(), credPath)
	}

	// Read credentials file
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Parse JSON
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Initialize profiles map if nil
	if creds.Profiles == nil {
		creds.Profiles = make(map[string]ProfileCredential)
	}

	return &creds, nil
}

// SaveCredentials saves credentials to file with secure permissions (0600)
func SaveCredentials(creds *Credentials) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	credPath := filepath.Join(configDir, "credentials.json")

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to file with restrictive permissions (0600 - owner read/write only)
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// GetToken retrieves the token for a specific profile
func GetToken(profileName string) (string, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return "", err
	}

	profileCred, exists := creds.Profiles[profileName]
	if !exists {
		return "", fmt.Errorf("no credentials found for profile '%s'", profileName)
	}

	// Check if token is expired
	if !profileCred.TokenExpiry.IsZero() && time.Now().After(profileCred.TokenExpiry) {
		return "", fmt.Errorf("token for profile '%s' has expired", profileName)
	}

	return profileCred.Token, nil
}

// SetToken stores a token for a specific profile
func SetToken(profileName, endpoint, token string, expiry time.Time) error {
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	// Ensure profiles map is initialized
	if creds.Profiles == nil {
		creds.Profiles = make(map[string]ProfileCredential)
	}

	// Store credential
	creds.Profiles[profileName] = ProfileCredential{
		Endpoint:    endpoint,
		Token:       token,
		TokenExpiry: expiry,
	}

	return SaveCredentials(creds)
}

// ClearToken removes the token for a specific profile
func ClearToken(profileName string) error {
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	// Delete profile credentials
	delete(creds.Profiles, profileName)

	return SaveCredentials(creds)
}

// IsTokenValid checks if a token exists and is not expired for a profile
func IsTokenValid(profileName string) bool {
	creds, err := LoadCredentials()
	if err != nil {
		return false
	}

	profileCred, exists := creds.Profiles[profileName]
	if !exists {
		return false
	}

	// Check if token is expired
	if !profileCred.TokenExpiry.IsZero() && time.Now().After(profileCred.TokenExpiry) {
		return false
	}

	return profileCred.Token != ""
}

// GetAllProfiles returns a list of all profile names with stored credentials
func GetAllProfiles() ([]string, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	profiles := make([]string, 0, len(creds.Profiles))
	for profileName := range creds.Profiles {
		profiles = append(profiles, profileName)
	}

	return profiles, nil
}
