package services

import (
	"os"
	"testing"
)

func TestNewEncryptionService(t *testing.T) {
	// Test missing env var
	os.Unsetenv("CALENDAR_ENCRYPTION_KEY")
	_, err := NewEncryptionService()
	if err == nil {
		t.Error("Expected error when CALENDAR_ENCRYPTION_KEY not set")
	}

	// Test with env var set
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-key-for-encryption-purposes-32-chars")
	_, err = NewEncryptionService()
	if err != nil {
		t.Errorf("Expected no error with env var set: %v", err)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-key-for-encryption-purposes-32-chars")
	svc, err := NewEncryptionService()
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name  string
		input string
	}{
		{"simple password", "password123"},
		{"with special chars", "p@$$w0rd!#$%"},
		{"empty string", ""},
		{"long password", "this-is-a-very-long-calendar-password-that-might-be-used-by-some-users"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := svc.Encrypt(tc.input)
			if err != nil {
				t.Errorf("Encrypt failed: %v", err)
				return
			}

			// Encrypted value should differ from input
			if encrypted == tc.input && tc.input != "" {
				t.Error("Encrypted value should differ from plaintext")
			}

			// Decrypt and verify
			decrypted, err := svc.Decrypt(encrypted)
			if err != nil {
				t.Errorf("Decrypt failed: %v", err)
				return
			}

			if decrypted != tc.input {
				t.Errorf("Decrypted value mismatch: got %q, want %q", decrypted, tc.input)
			}
		})
	}
}

func TestDecryptInvalid(t *testing.T) {
	os.Setenv("CALENDAR_ENCRYPTION_KEY", "test-key-for-encryption-purposes-32-chars")
	svc, err := NewEncryptionService()
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name string
		input string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YWJj"},
		{"corrupted data", "YWJjZGVmZ2hpamtsbW5vcHFy"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Decrypt(tc.input)
			if err == nil {
				t.Error("Expected error for invalid input")
			}
		})
	}
}
