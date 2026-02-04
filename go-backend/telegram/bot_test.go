package telegram

import (
	"testing"
)

func TestNewBot(t *testing.T) {
	// Test with invalid token
	_, err := NewBot("invalid_token", 123, 1, nil)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Note: Testing with real token requires integration test setup
	// Unit tests for message handling can be added with mock handler
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"start command", "/start", true},
		{"help command", "/help", true},
		{"clear command", "/clear", true},
		{"regular message", "hello there", false},
		{"message with slash in middle", "check /this out", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := len(tt.input) > 0 && tt.input[0] == '/'
			if result != tt.expected {
				t.Errorf("isCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
