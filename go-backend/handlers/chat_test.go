package handlers

import (
	"go-backend/models"
	"go-backend/services"
	"testing"
)

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
			converter := services.NewMessageConverter()
			result := converter.ShouldClearToolResult(tt.msg, tt.messageIndex, tt.totalMessages)
			if result != tt.expected {
				t.Errorf("ShouldClearToolResult() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}