package handlers

import (
	"go-backend/models"
	"testing"
)

func TestIsToolResultEmpty(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected bool
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: true,
		},
		{
			name:     "empty map",
			result:   map[string]interface{}{},
			expected: true,
		},
		{
			name:     "only empty string",
			result:   map[string]interface{}{"data": ""},
			expected: true,
		},
		{
			name:     "only whitespace string",
			result:   map[string]interface{}{"data": "   "},
			expected: true,
		},
		{
			name:     "empty array",
			result:   map[string]interface{}{"data": []interface{}{}},
			expected: true,
		},
		{
			name:     "empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{}},
			expected: true,
		},
		{
			name:     "only error field",
			result:   map[string]interface{}{"error": "some error"},
			expected: true,
		},
		{
			name:     "non-empty string",
			result:   map[string]interface{}{"data": "some value"},
			expected: false,
		},
		{
			name:     "non-empty array",
			result:   map[string]interface{}{"data": []interface{}{"item"}},
			expected: false,
		},
		{
			name:     "non-empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{"key": "value"}},
			expected: false,
		},
		{
			name:     "numeric value",
			result:   map[string]interface{}{"count": 42},
			expected: false,
		},
		{
			name:     "boolean value",
			result:   map[string]interface{}{"success": true},
			expected: false,
		},
		{
			name:     "mixed empty and non-empty",
			result:   map[string]interface{}{"empty": "", "data": "value"},
			expected: false,
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

func TestShouldClearToolResult(t *testing.T) {
	toolCallID := "call_123"
	userMessage := models.ChatMessage{Role: "user", Content: strPtr("Hello")}
	assistantMessage := models.ChatMessage{Role: "assistant", Content: strPtr("Hi")}
	toolMessage := models.ChatMessage{Role: "tool", ToolCallID: &toolCallID, Content: strPtr("Tool result")}

	tests := []struct {
		name          string
		msg           models.ChatMessage
		messageIndex  int
		totalMessages int
		expected      bool
	}{
		{
			name:          "user message should not be cleared",
			msg:           userMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "assistant message should not be cleared",
			msg:           assistantMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "recent tool message (last 10) should not be cleared",
			msg:           toolMessage,
			messageIndex:  15,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "tool message at exactly 10 from end should not be cleared",
			msg:           toolMessage,
			messageIndex:  10,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "old tool message (>10 from end) should be cleared",
			msg:           toolMessage,
			messageIndex:  5,
			totalMessages: 20,
			expected:      true,
		},
		{
			name:          "very old tool message should be cleared",
			msg:           toolMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      true,
		},
		{
			name:          "tool message in short conversation should not be cleared",
			msg:           toolMessage,
			messageIndex:  0,
			totalMessages: 8,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldClearToolResult(tt.msg, tt.messageIndex, tt.totalMessages)
			if result != tt.expected {
				t.Errorf("shouldClearToolResult() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}