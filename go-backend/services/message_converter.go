package services

import (
	"encoding/json"

	openai "github.com/sashabaranov/go-openai"
	"go-backend/models"
)

// MessageConverter handles conversion between our chat messages and OpenAI format
type MessageConverter struct {
	// Configuration options
	ClearOldToolsAfter int // Clear tool results older than N messages
}

// NewMessageConverter creates a new message converter
func NewMessageConverter() *MessageConverter {
	return &MessageConverter{
		ClearOldToolsAfter: 10,
	}
}

// NewMessageConverterWithConfig creates a new message converter with custom configuration
func NewMessageConverterWithConfig(clearOldToolsAfter int) *MessageConverter {
	return &MessageConverter{
		ClearOldToolsAfter: clearOldToolsAfter,
	}
}

// ShouldClearToolResult determines if a tool result should be cleared to save context
func (mc *MessageConverter) ShouldClearToolResult(msg models.ChatMessage, messageIndex int, totalMessages int) bool {
	// Only clear tool messages
	if msg.Role != "tool" {
		return false
	}

	// Don't clear if it's in the last N messages (keep recent context)
	messagesFromEnd := totalMessages - messageIndex
	if messagesFromEnd <= mc.ClearOldToolsAfter {
		return false
	}

	return true
}

// ToOpenAI converts our chat messages to OpenAI format
func (mc *MessageConverter) ToOpenAI(messages []models.ChatMessage, systemPrompt string) []openai.ChatCompletionMessage {
	openaiMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for i, msg := range messages {
		// Clear old tool results to reduce context pollution
		if mc.ShouldClearToolResult(msg, i, len(messages)) {
			// Replace with minimal placeholder
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:       "tool",
				ToolCallID: *msg.ToolCallID,
				Content:    "[Result cleared to save context]",
			})
			continue
		}

		var content string
		if msg.Content != nil {
			content = *msg.Content
		}

		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			var toolCalls []openai.ToolCall
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// Handle tool call responses
		if msg.ToolCallID != nil {
			openaiMsg.ToolCallID = *msg.ToolCallID
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	return openaiMessages
}
