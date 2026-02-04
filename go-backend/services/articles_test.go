package services

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	// Test with a real URL that should be accessible
	result, err := ParseURL("https://example.com")
	if err != nil {
		t.Fatalf("ParseURL failed: %v", err)
	}

	if result.Title == "" {
		t.Error("Expected title to be set")
	}

	if result.Content == "" {
		t.Error("Expected content to be set")
	}

	if result.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got '%s'", result.URL)
	}
}

func TestParseURLInvalid(t *testing.T) {
	_, err := ParseURL("")
	if err == nil {
		t.Error("Expected error for empty URL")
	}

	_, err = ParseURL("not-a-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}
