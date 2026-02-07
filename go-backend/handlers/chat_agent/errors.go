package chat_agent

import (
	"fmt"
	"go-backend/services"
	"strings"
)

// getUserFacingErrorMessage returns a user-friendly error message using the canonical error classifier
func getUserFacingErrorMessage(err error, customMessage string) string {
	// Use the canonical error classifier from services package
	errInfo := services.ClassifyToolError("", nil, err)
	if customMessage != "" {
		return customMessage
	}
	if errInfo != nil {
		return errInfo.Suggestion
	}
	// Fallback for context length errors (special case for LLM chat)
	if services.IsContextLengthError(err) {
		return "I apologize, but this conversation has become too long for me to process. Please consider starting a new conversation or summarizing the key points you'd like to continue discussing."
	}
	return "Something went wrong. Please try again."
}

// isToolResultEmpty checks if a tool result is effectively empty
func isToolResultEmpty(result map[string]interface{}) bool {
	if result == nil || len(result) == 0 {
		return true
	}

	// Check if result only contains empty values
	for key, value := range result {
		// Skip error field for this check
		if key == "error" {
			continue
		}

		// Check various types of emptiness
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return false
			}
		case []interface{}:
			if len(v) > 0 {
				return false
			}
		case map[string]interface{}:
			if len(v) > 0 {
				return false
			}
		default:
			// If there's any other non-nil value, consider it non-empty
			if v != nil {
				return false
			}
		}
	}

	return true
}

// handleLLMError creates a user-facing error message for LLM errors
func (s *ChatService) handleLLMError(err error, userID int, conversationID, assistantMessageID string, updateMessage func(userID int, conversationID, messageID string, content string) error) (string, error) {
	userMessage := getUserFacingErrorMessage(err, "")

	if updateErr := updateMessage(userID, conversationID, assistantMessageID, userMessage); updateErr != nil {
		return "", fmt.Errorf("error updating message with error: %w", updateErr)
	}
	return userMessage, nil
}
