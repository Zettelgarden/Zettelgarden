// Package chat provides message helper utilities for the chat service.
package chat

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// FinalizeChatMessage creates a finalized chat message with all required fields.
func FinalizeChatMessage(
	messageID string,
	content *string,
	toolCalls []openai.ToolCall,
	role string,
) models.ChatMessage {
	return models.ChatMessage{
		ID:             messageID,
		Content:        content,
		ToolCalls:      convertToolCalls(toolCalls),
		Role:           role,
		CreatedAt:      time.Now(),
		SequenceNumber: 0, // Will be set by caller
	}
}

// ValidateMessage checks if a message has valid content and/or tool calls.
func ValidateMessage(content string, toolCalls []openai.ToolCall) error {
	hasContent := strings.TrimSpace(content) != ""
	hasToolCalls := len(toolCalls) > 0

	if !hasContent && !hasToolCalls {
		return fmt.Errorf("message must have either content or tool calls")
	}

	return nil
}

// IsToolResultEmpty checks if a tool result is effectively empty.
func IsToolResultEmpty(result interface{}) bool {
	if result == nil {
		return true
	}

	// Check for empty maps
	if resultMap, ok := result.(map[string]interface{}); ok {
		if len(resultMap) == 0 {
			return true
		}

		// Check for error field with empty content
		if errMsg, hasError := resultMap["error"]; hasError {
			if errStr, ok := errMsg.(string); ok && strings.TrimSpace(errStr) == "" {
				// If error field exists but is empty, check for actual data
				delete(resultMap, "error")
				return len(resultMap) == 0
			}
		}
	}

	// Check for empty strings
	if resultStr, ok := result.(string); ok {
		return strings.TrimSpace(resultStr) == ""
	}

	// Check for empty slices
	if resultSlice, ok := result.([]interface{}); ok {
		return len(resultSlice) == 0
	}

	return false
}

// ConvertToolCalls converts OpenAI tool calls to the internal model format.
func ConvertToolCalls(toolCalls []openai.ToolCall) []models.ChatToolCall {
	return convertToolCalls(toolCalls)
}

// convertToolCalls converts OpenAI tool calls to the internal model format.
func convertToolCalls(toolCalls []openai.ToolCall) []models.ChatToolCall {
	var result []models.ChatToolCall
	for _, tc := range toolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result = append(result, models.ChatToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: models.ChatToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		})
	}
	return result
}

// SerializeToolCalls converts tool calls to JSON string format.
func SerializeToolCalls(toolCalls []openai.ToolCall) (*string, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	toolCallsData, err := json.Marshal(toolCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool calls: %w", err)
	}

	toolCallsStr := string(toolCallsData)
	return &toolCallsStr, nil
}

// DeserializeToolCalls converts JSON string to tool calls.
func DeserializeToolCalls(toolCallsJSON *string) ([]openai.ToolCall, error) {
	if toolCallsJSON == nil || *toolCallsJSON == "" {
		return nil, nil
	}

	var toolCalls []openai.ToolCall
	err := json.Unmarshal([]byte(*toolCallsJSON), &toolCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
	}

	return toolCalls, nil
}

// ExtractToolCallArguments parses the arguments from a tool call.
func ExtractToolCallArguments(toolCall openai.ToolCall) (map[string]interface{}, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
	}
	return args, nil
}

// FormatToolResultForDisplay formats a tool result for user display.
func FormatToolResultForDisplay(result interface{}) string {
	if result == nil {
		return "Tool returned no result"
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("Tool result: %v", result)
	}

	return string(resultJSON)
}

// SanitizeMessageContent removes potentially harmful content from messages.
func SanitizeMessageContent(content string) string {
	// Remove null bytes
	content = strings.ReplaceAll(content, "\x00", "")

	// Trim whitespace
	content = strings.TrimSpace(content)

	// Limit length to prevent abuse
	const maxContentLength = 100000
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "... (truncated)"
	}

	return content
}

// ValidateMessageRole checks if a message role is valid.
func ValidateMessageRole(role string) error {
	validRoles := map[string]bool{
		"system":    true,
		"user":      true,
		"assistant": true,
		"tool":      true,
	}

	if !validRoles[role] {
		return fmt.Errorf("invalid message role: %s", role)
	}

	return nil
}

// BuildToolResponseMessage creates a tool response message for the LLM.
func BuildToolResponseMessage(toolCallID string, result interface{}) openai.ChatCompletionMessage {
	resultJSON, _ := json.Marshal(result)

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		ToolCallID: toolCallID,
		Content:    string(resultJSON),
	}
}
