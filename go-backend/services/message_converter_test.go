package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	openai "github.com/sashabaranov/go-openai"

	"go-backend/models"
)

func TestMessageConverter_ShouldClearToolResult(t *testing.T) {
	t.Run("should clear tool messages older than N messages", func(t *testing.T) {
		converter := NewMessageConverterWithConfig(5) // Clear after 5 messages

		// Create a tool message at index 0 of 15 total messages
		toolMsg := models.ChatMessage{
			ID:     "msg-1",
			Role:   "tool",
			Content: stringPtr("Tool result content"),
		}

		// At index 0 of 15, messagesFromEnd = 15, which is > 5, so should clear
		shouldClear := converter.ShouldClearToolResult(toolMsg, 0, 15)
		assert.True(t, shouldClear, "Tool message at index 0 of 15 should be cleared when threshold is 5")
	})

	t.Run("should NOT clear recent tool messages (last N)", func(t *testing.T) {
		converter := NewMessageConverterWithConfig(10) // Clear after 10 messages

		// Test cases for recent messages (within last 10)
		testCases := []struct {
			name          string
			messageIndex  int
			totalMessages int
			shouldClear   bool
		}{
			{
				name:          "last message (index 14 of 15)",
				messageIndex:  14,
				totalMessages: 15,
				shouldClear:   false, // messagesFromEnd = 1 <= 10
			},
			{
				name:          "5th from end (index 10 of 15)",
				messageIndex:  10,
				totalMessages: 15,
				shouldClear:   false, // messagesFromEnd = 5 <= 10
			},
			{
				name:          "10th from end (index 5 of 15)",
				messageIndex:  5,
				totalMessages: 15,
				shouldClear:   false, // messagesFromEnd = 10 <= 10
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				toolMsg := models.ChatMessage{
					ID:     "msg-1",
					Role:   "tool",
					Content: stringPtr("Tool result"),
				}

				shouldClear := converter.ShouldClearToolResult(toolMsg, tc.messageIndex, tc.totalMessages)
				assert.Equal(t, tc.shouldClear, shouldClear)
			})
		}
	})

	t.Run("should NOT clear non-tool messages", func(t *testing.T) {
		converter := NewMessageConverter()

		// Test various non-tool roles
		roles := []string{"user", "assistant", "system"}

		for _, role := range roles {
			t.Run("role_"+role, func(t *testing.T) {
				msg := models.ChatMessage{
					ID:     "msg-1",
					Role:   role,
					Content: stringPtr("Some content"),
				}

				// Even at index 0 of 100, non-tool messages should never be cleared
				shouldClear := converter.ShouldClearToolResult(msg, 0, 100)
				assert.False(t, shouldClear, "Non-tool messages should never be cleared")
			})
		}
	})

	t.Run("should clear tool messages exactly at threshold boundary", func(t *testing.T) {
		converter := NewMessageConverterWithConfig(10)

		// At index 4 of 15: messagesFromEnd = 11 > 10, so should clear
		toolMsg := models.ChatMessage{
			ID:     "msg-1",
			Role:   "tool",
			Content: stringPtr("Tool result"),
		}

		shouldClear := converter.ShouldClearToolResult(toolMsg, 4, 15)
		assert.True(t, shouldClear, "Tool message at threshold boundary should be cleared")
	})

	t.Run("default configuration uses threshold of 10", func(t *testing.T) {
		converter := NewMessageConverter() // Default is 10

		// Test with default config
		toolMsg := models.ChatMessage{
			ID:     "msg-1",
			Role:   "tool",
			Content: stringPtr("Tool result"),
		}

		// At index 9 of 20: messagesFromEnd = 11 > 10, should clear
		shouldClear := converter.ShouldClearToolResult(toolMsg, 9, 20)
		assert.True(t, shouldClear)

		// At index 10 of 20: messagesFromEnd = 10 <= 10, should not clear
		shouldClear = converter.ShouldClearToolResult(toolMsg, 10, 20)
		assert.False(t, shouldClear)
	})
}

func TestMessageConverter_ToOpenAI(t *testing.T) {
	t.Run("converts system prompt correctly", func(t *testing.T) {
		converter := NewMessageConverter()
		systemPrompt := "You are a helpful assistant."

		messages := []models.ChatMessage{}
		result := converter.ToOpenAI(messages, systemPrompt)

		require.Len(t, result, 1, "Should have system message")
		assert.Equal(t, openai.ChatMessageRoleSystem, result[0].Role)
		assert.Equal(t, systemPrompt, result[0].Content)
	})

	t.Run("converts user and assistant messages", func(t *testing.T) {
		converter := NewMessageConverter()
		systemPrompt := "System prompt"

		content1 := "Hello, how are you?"
		content2 := "I'm doing well, thank you!"

		messages := []models.ChatMessage{
			{
				ID:      "msg-1",
				Role:    "user",
				Content: &content1,
			},
			{
				ID:      "msg-2",
				Role:    "assistant",
				Content: &content2,
			},
		}

		result := converter.ToOpenAI(messages, systemPrompt)

		require.Len(t, result, 3) // System + 2 messages

		// First message after system is user
		assert.Equal(t, "user", result[1].Role)
		assert.Equal(t, content1, result[1].Content)

		// Second message is assistant
		assert.Equal(t, "assistant", result[2].Role)
		assert.Equal(t, content2, result[2].Content)
	})

	t.Run("handles tool calls with proper serialization", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Let me search for that information."
		toolCallID := "call_abc123"

		messages := []models.ChatMessage{
			{
				ID:      "msg-1",
				Role:    "assistant",
				Content: &content,
				ToolCalls: []models.ChatToolCall{
					{
						ID:   toolCallID,
						Type: "function",
						Function: models.ChatToolCallFunction{
							Name: "search_cards",
							Arguments: map[string]interface{}{
								"query":     "test query",
								"max_results": 10,
							},
						},
					},
				},
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		require.Len(t, result[1].ToolCalls, 1)

		toolCall := result[1].ToolCalls[0]
		assert.Equal(t, toolCallID, toolCall.ID)
		assert.Equal(t, openai.ToolTypeFunction, toolCall.Type)
		assert.Equal(t, "search_cards", toolCall.Function.Name)
		assert.Contains(t, toolCall.Function.Arguments, "test query")
		assert.Contains(t, toolCall.Function.Arguments, "max_results")
	})

	t.Run("handles tool call responses with ToolCallID", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Search found 5 results."
		toolCallID := "call_xyz789"

		messages := []models.ChatMessage{
			{
				ID:         "msg-1",
				Role:       "tool",
				Content:    &content,
				ToolCallID: &toolCallID,
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		assert.Equal(t, "tool", result[1].Role)
		assert.Equal(t, toolCallID, result[1].ToolCallID)
		assert.Equal(t, content, result[1].Content)
	})

	t.Run("clears old tool results and replaces with placeholder", func(t *testing.T) {
		converter := NewMessageConverterWithConfig(5) // Clear after 5 messages

		toolCallID := "call_old123"
		content := "Old tool result that should be cleared"

		// Create 10 messages with a tool message at the beginning
		messages := make([]models.ChatMessage, 10)
		messages[0] = models.ChatMessage{
			ID:         "msg-1",
			Role:       "tool",
			Content:    &content,
			ToolCallID: &toolCallID,
		}

		// Fill the rest with user messages
		for i := 1; i < 10; i++ {
			c := "Message"
			messages[i] = models.ChatMessage{
				ID:      "msg-" + string(rune('1'+i)),
				Role:    "user",
				Content: &c,
			}
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 11) // System + 10 messages

		// First tool message should be cleared with placeholder
		assert.Equal(t, "tool", result[1].Role)
		assert.Equal(t, toolCallID, result[1].ToolCallID)
		assert.Equal(t, "[Result cleared to save context]", result[1].Content)
	})

	t.Run("keeps recent tool results", func(t *testing.T) {
		converter := NewMessageConverterWithConfig(5)

		content := "Recent tool result"
		toolCallID := "call_recent123"

		messages := []models.ChatMessage{
			// 5 user messages
			{ID: "msg-1", Role: "user", Content: stringPtr("Msg 1")},
			{ID: "msg-2", Role: "user", Content: stringPtr("Msg 2")},
			{ID: "msg-3", Role: "user", Content: stringPtr("Msg 3")},
			{ID: "msg-4", Role: "user", Content: stringPtr("Msg 4")},
			{ID: "msg-5", Role: "user", Content: stringPtr("Msg 5")},
			// Recent tool message (within last 5)
			{
				ID:         "msg-6",
				Role:       "tool",
				Content:    &content,
				ToolCallID: &toolCallID,
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 7) // System + 6 messages

		// Tool message should keep its content
		assert.Equal(t, "tool", result[6].Role)
		assert.Equal(t, toolCallID, result[6].ToolCallID)
		assert.Equal(t, content, result[6].Content)
	})
}

func TestMessageConverter_ToOpenAI_EdgeCases(t *testing.T) {
	t.Run("handles nil content", func(t *testing.T) {
		converter := NewMessageConverter()

		messages := []models.ChatMessage{
			{
				ID:      "msg-1",
				Role:    "user",
				Content: nil, // Nil content
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		assert.Equal(t, "user", result[1].Role)
		assert.Equal(t, "", result[1].Content) // Should be empty string
	})

	t.Run("handles empty tool calls array", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Message with empty tool calls"

		messages := []models.ChatMessage{
			{
				ID:        "msg-1",
				Role:      "assistant",
				Content:   &content,
				ToolCalls: []models.ChatToolCall{}, // Empty array
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		assert.Equal(t, content, result[1].Content)
		assert.Nil(t, result[1].ToolCalls) // Should be nil, not empty slice
	})

	t.Run("handles messages without tool calls", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Simple message without tools"

		messages := []models.ChatMessage{
			{
				ID:        "msg-1",
				Role:      "user",
				Content:   &content,
				ToolCalls: nil, // No tool calls
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		assert.Equal(t, content, result[1].Content)
		assert.Nil(t, result[1].ToolCalls)
		assert.Empty(t, result[1].ToolCallID)
	})

	t.Run("handles multiple tool calls in one message", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Let me make several calls."

		messages := []models.ChatMessage{
			{
				ID:      "msg-1",
				Role:    "assistant",
				Content: &content,
				ToolCalls: []models.ChatToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: models.ChatToolCallFunction{
							Name: "search_cards",
							Arguments: map[string]interface{}{
								"query": "test",
							},
						},
					},
					{
						ID:   "call_2",
						Type: "function",
						Function: models.ChatToolCallFunction{
							Name: "get_card",
							Arguments: map[string]interface{}{
								"card_id": "123",
							},
						},
					},
				},
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		require.Len(t, result[1].ToolCalls, 2)

		// Verify first tool call
		assert.Equal(t, "call_1", result[1].ToolCalls[0].ID)
		assert.Equal(t, "search_cards", result[1].ToolCalls[0].Function.Name)

		// Verify second tool call
		assert.Equal(t, "call_2", result[1].ToolCalls[1].ID)
		assert.Equal(t, "get_card", result[1].ToolCalls[1].Function.Name)
	})

	t.Run("handles complex arguments in tool calls", func(t *testing.T) {
		converter := NewMessageConverter()

		content := "Searching with complex filters"

		messages := []models.ChatMessage{
			{
				ID:      "msg-1",
				Role:    "assistant",
				Content: &content,
				ToolCalls: []models.ChatToolCall{
					{
						ID:   "call_complex",
						Type: "function",
						Function: models.ChatToolCallFunction{
							Name: "advanced_search",
							Arguments: map[string]interface{}{
								"filters": []map[string]interface{}{
									{"field": "tags", "value": "important"},
									{"field": "date", "value": "2024-01-01"},
								},
								"options": map[string]interface{}{
									"case_sensitive": false,
									"max_results":    100,
								},
							},
						},
					},
				},
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		require.Len(t, result[1].ToolCalls, 1)

		// Verify JSON serialization of complex arguments
		argsJSON := result[1].ToolCalls[0].Function.Arguments
		assert.Contains(t, argsJSON, "filters")
		assert.Contains(t, argsJSON, "options")
		assert.Contains(t, argsJSON, "tags")
		assert.Contains(t, argsJSON, "important")
	})

	t.Run("handles empty messages array", func(t *testing.T) {
		converter := NewMessageConverter()
		systemPrompt := "System prompt"

		messages := []models.ChatMessage{}
		result := converter.ToOpenAI(messages, systemPrompt)

		require.Len(t, result, 1, "Should only have system message")
		assert.Equal(t, openai.ChatMessageRoleSystem, result[0].Role)
		assert.Equal(t, systemPrompt, result[0].Content)
	})

	t.Run("handles tool message with nil content", func(t *testing.T) {
		converter := NewMessageConverter()

		toolCallID := "call_test123"

		messages := []models.ChatMessage{
			{
				ID:         "msg-1",
				Role:       "tool",
				Content:    nil, // Nil content
				ToolCallID: &toolCallID,
			},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 2)
		assert.Equal(t, "tool", result[1].Role)
		assert.Equal(t, toolCallID, result[1].ToolCallID)
		assert.Equal(t, "", result[1].Content)
	})

	t.Run("preserves message order", func(t *testing.T) {
		converter := NewMessageConverter()

		c1, c2, c3 := "Message 1", "Message 2", "Message 3"
		toolCallID := "call_order_test"

		messages := []models.ChatMessage{
			{ID: "msg-1", Role: "user", Content: &c1},
			{
				ID:         "msg-2",
				Role:       "tool",
				Content:    &c2,
				ToolCallID: &toolCallID,
			},
			{ID: "msg-3", Role: "assistant", Content: &c3},
		}

		result := converter.ToOpenAI(messages, "System prompt")

		require.Len(t, result, 4) // System + 3 messages
		assert.Equal(t, "user", result[1].Role)
		assert.Equal(t, c1, result[1].Content)
		assert.Equal(t, "tool", result[2].Role)
		assert.Equal(t, c2, result[2].Content)
		assert.Equal(t, "assistant", result[3].Role)
		assert.Equal(t, c3, result[3].Content)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
