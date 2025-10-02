package handlers

import (
	"go-backend/models"
	"testing"

	openai "github.com/sashabaranov/go-openai"
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

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		messages []openai.ChatCompletionMessage
		minCount int // minimum expected tokens
		maxCount int // maximum expected tokens
	}{
		{
			name:     "empty messages",
			messages: []openai.ChatCompletionMessage{},
			minCount: 0,
			maxCount: 10,
		},
		{
			name: "single short message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
			},
			minCount: 10,
			maxCount: 20,
		},
		{
			name: "multiple messages",
			messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "What is the weather?"},
				{Role: "assistant", Content: "I don't have access to weather information."},
			},
			minCount: 40,
			maxCount: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimateTokenCount(tt.messages)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("estimateTokenCount() = %v, expected between %v and %v", count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestGetModelContextLimit(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedLimit int
	}{
		{
			name:          "gemini flash",
			model:         "google/gemini-2.5-flash",
			expectedLimit: 1000000,
		},
		{
			name:          "gemini pro",
			model:         "google/gemini-2.5-pro",
			expectedLimit: 2000000,
		},
		{
			name:          "gpt-4o-mini",
			model:         "openai/gpt-4o-mini",
			expectedLimit: 128000,
		},
		{
			name:          "claude sonnet",
			model:         "anthropic/claude-sonnet-4",
			expectedLimit: 200000,
		},
		{
			name:          "unknown model",
			model:         "unknown/model",
			expectedLimit: 100000, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := getModelContextLimit(tt.model)
			if limit != tt.expectedLimit {
				t.Errorf("getModelContextLimit() = %v, expected %v", limit, tt.expectedLimit)
			}
		})
	}
}