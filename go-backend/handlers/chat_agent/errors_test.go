package chat_agent

import (
	"errors"
	"testing"
)

func TestGetUserFacingErrorMessage(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		customMessage  string
		expectedSubstr string
	}{
		{
			name:           "nil error returns default",
			err:            nil,
			customMessage:  "",
			expectedSubstr: "Something went wrong",
		},
		{
			name:           "custom message overrides default",
			err:            nil,
			customMessage:  "Custom error message",
			expectedSubstr: "Custom error message",
		},
		{
			name:           "generic error uses classifier suggestion",
			err:            errors.New("some error"),
			customMessage:  "",
			expectedSubstr: "unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUserFacingErrorMessage(tt.err, tt.customMessage)
			if !contains(result, tt.expectedSubstr) {
				t.Errorf("getUserFacingErrorMessage() = %v, expected to contain %v", result, tt.expectedSubstr)
			}
		})
	}
}

func TestIsToolResultEmpty(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected bool
	}{
		{
			name:     "nil result is empty",
			result:   nil,
			expected: true,
		},
		{
			name:     "empty map is empty",
			result:   map[string]interface{}{},
			expected: true,
		},
		{
			name: "result with only error is empty",
			result: map[string]interface{}{
				"error": "some error",
			},
			expected: true,
		},
		{
			name: "result with string content is not empty",
			result: map[string]interface{}{
				"content": "some text",
			},
			expected: false,
		},
		{
			name: "result with array is not empty",
			result: map[string]interface{}{
				"items": []interface{}{1, 2, 3},
			},
			expected: false,
		},
		{
			name: "result with nested map is not empty",
			result: map[string]interface{}{
				"data": map[string]interface{}{"key": "value"},
			},
			expected: false,
		},
		{
			name: "whitespace-only string is empty",
			result: map[string]interface{}{
				"content": "   ",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolResultEmpty(tt.result)
			if result != tt.expected {
				t.Errorf("isToolResultEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFinalizeChatMessage(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "non-empty content returns as-is",
			content:  "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "empty content returns fallback",
			content:  "",
			expected: "I apologize, but I wasn't able to generate a proper response. Could you please try rephrasing your question?",
		},
		{
			name:     "whitespace-only content returns as-is",
			content:  "   ",
			expected: "   ",
		},
	}

	s := &ChatService{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.finalizeChatMessage(tt.content)
			if result != tt.expected {
				t.Errorf("finalizeChatMessage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
