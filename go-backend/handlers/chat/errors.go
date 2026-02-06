// Package chat provides error handling utilities for the chat service.
package chat

import (
	"context"
	"fmt"
	"go-backend/models"
	"go-backend/services"
)

// ErrorType classifies different types of errors that can occur in the chat service.
type ErrorType int

const (
	// ErrorTypeUnknown is for unclassified errors.
	ErrorTypeUnknown ErrorType = iota

	// ErrorTypeContextLength indicates the conversation exceeded the model's context limit.
	ErrorTypeContextLength

	// ErrorTypeEmptyResponse indicates the LLM returned an empty response.
	ErrorTypeEmptyResponse

	// ErrorTypeNoChoices indicates the LLM returned no choices in the response.
	ErrorTypeNoChoices

	// ErrorTypeToolExecution indicates a tool execution failure.
	ErrorTypeToolExecution

	// ErrorTypeStreamDisconnection indicates the client disconnected during streaming.
	ErrorTypeStreamDisconnection

	// ErrorTypeRateLimit indicates the API rate limit was exceeded.
	ErrorTypeRateLimit

	// ErrorTypeAuthentication indicates an authentication failure with the LLM provider.
	ErrorTypeAuthentication
)

// ChatError wraps an error with additional context about the error type.
type ChatError struct {
	Err        error
	Type       ErrorType
	Message    string
	StatusCode int
}

// Error implements the error interface.
func (e *ChatError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *ChatError) Unwrap() error {
	return e.Err
}

// GetUserFacingMessage returns a user-friendly error message based on the error type.
func (e *ChatError) GetUserFacingMessage() string {
	switch e.Type {
	case ErrorTypeContextLength:
		return "I apologize, but this conversation has become too long for me to process. Please consider starting a new conversation or summarizing the key points you'd like to continue discussing."
	case ErrorTypeEmptyResponse:
		return "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
	case ErrorTypeNoChoices:
		return "I'm having trouble connecting to my language model right now. Please try again."
	case ErrorTypeStreamDisconnection:
		return "The connection was interrupted. Please try again."
	case ErrorTypeRateLimit:
		return "I'm receiving too many requests right now. Please wait a moment and try again."
	case ErrorTypeAuthentication:
		return "There's an issue with my language service configuration. Please contact support."
	case ErrorTypeToolExecution:
		return "I encountered an issue while processing your request. Please try again."
	default:
		return "Something went wrong. Please try again."
	}
}

// ClassifyError analyzes an error and returns its classification.
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	if services.IsContextLengthError(err) {
		return ErrorTypeContextLength
	}

	// Check for context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return ErrorTypeStreamDisconnection
	}

	errMsg := err.Error()
	switch {
	case stringsContains(errMsg, "empty response"):
		return ErrorTypeEmptyResponse
	case stringsContains(errMsg, "no choices"):
		return ErrorTypeNoChoices
	case stringsContains(errMsg, "rate limit") || stringsContains(errMsg, "quota exceeded"):
		return ErrorTypeRateLimit
	case stringsContains(errMsg, "authentication") || stringsContains(errMsg, "unauthorized") || stringsContains(errMsg, "invalid api key"):
		return ErrorTypeAuthentication
	case stringsContains(errMsg, "tool") || stringsContains(errMsg, "function call"):
		return ErrorTypeToolExecution
	default:
		return ErrorTypeUnknown
	}
}

// HandleLLMError processes an LLM error and returns an appropriate chat message.
func (s *Service) HandleLLMError(
	ctx context.Context,
	err error,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
) (*models.ChatMessage, error) {
	return s.handleLLMError(ctx, err, userID, conversation, assistantMessageID)
}

// handleLLMError processes an LLM error and returns an appropriate chat message.
func (s *Service) handleLLMError(
	ctx context.Context,
	err error,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
) (*models.ChatMessage, error) {
	s.logError("executing LLM error for conversation %s: %v", conversation.ID, err)

	errorType := ClassifyError(err)
	chatErr := &ChatError{
		Err:     err,
		Type:    errorType,
		Message: fmt.Sprintf("LLM error for conversation %s", conversation.ID),
	}

	userMessage := chatErr.GetUserFacingMessage()

	return &models.ChatMessage{
		ID:      assistantMessageID,
		Content: &userMessage,
	}, nil
}

// GetUserFacingMessage returns a user-friendly message for an error.
func GetUserFacingMessage(err error, customMessage string) string {
	if customMessage != "" {
		return customMessage
	}

	chatErr := &ChatError{
		Err:  err,
		Type: ClassifyError(err),
	}

	return chatErr.GetUserFacingMessage()
}

// NewChatError creates a new ChatError with the given parameters.
func NewChatError(err error, errorType ErrorType, message string) *ChatError {
	return &ChatError{
		Err:     err,
		Type:    errorType,
		Message: message,
	}
}

// stringsContains is a simple case-insensitive string contains check.
func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 &&
		(s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

// containsMiddle checks if substr is in the middle of s.
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
