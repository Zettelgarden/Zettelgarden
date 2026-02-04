package services

import (
	"fmt"
	"go-backend/models"
	"regexp"
	"strings"
)

// isNetworkError checks if an error is network-related
func isNetworkError(err error) bool {
	errStr := strings.ToLower(err.Error())
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"network",
		"dial tcp",
		"no such host",
		"timeout",
		"deadline exceeded",
		"temporary failure",
		"EOF",
	}
	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isDatabaseError checks if an error is database-related
func isDatabaseError(err error) bool {
	errStr := strings.ToLower(err.Error())
	dbPatterns := []string{
		"sql",
		"database",
		"db.",
		"pq:",
		"mysql",
		"postgres",
		"sqlite",
	}
	for _, pattern := range dbPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isTimeoutError checks if an error is a timeout
func isTimeoutError(err error) bool {
	errStr := strings.ToLower(err.Error())
	timeoutPatterns := []string{
		"timeout",
		"deadline exceeded",
		"context deadline exceeded",
	}
	for _, pattern := range timeoutPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isNotFoundError checks if an error is a not found error
func isNotFoundError(err error) bool {
	errStr := strings.ToLower(err.Error())
	notFoundPatterns := []string{
		"not found",
		"no rows in result set",
		"no such",
		"does not exist",
		"404",
	}
	for _, pattern := range notFoundPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isValidationError checks if an error is validation-related
func isValidationError(err error) bool {
	errStr := strings.ToLower(err.Error())
	validationPatterns := []string{
		"required",
		"invalid",
		"validation",
		"missing",
		"malformed",
		"cannot be blank",
		"must be",
	}
	for _, pattern := range validationPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isPermissionError checks if an error is permission-related
func isPermissionError(err error) bool {
	errStr := strings.ToLower(err.Error())
	permissionPatterns := []string{
		"permission",
		"unauthorized",
		"access denied",
		"forbidden",
		"403",
		"401",
	}
	for _, pattern := range permissionPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isRateLimitError checks if an error is rate limit related
func isRateLimitError(err error) bool {
	errStr := strings.ToLower(err.Error())
	rateLimitPatterns := []string{
		"rate limit",
		"too many requests",
		"429",
		"quota exceeded",
	}
	for _, pattern := range rateLimitPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// getValidationSuggestion returns a helpful suggestion based on validation errors
func getValidationSuggestion(toolName string, args map[string]interface{}, err error) string {
	errStr := strings.ToLower(err.Error())

	// Check for missing required parameters
	requiredRe := regexp.MustCompile(`required.*?(\w+)`)
	matches := requiredRe.FindStringSubmatch(errStr)
	if len(matches) > 1 {
		return fmt.Sprintf("The required parameter '%s' is missing. Please provide it.", matches[1])
	}

	// Check for invalid format
	if strings.Contains(errStr, "invalid") {
		if strings.Contains(errStr, "card") && strings.Contains(errStr, "id") {
			return "The card ID must be a valid number."
		}
		if strings.Contains(errStr, "query") {
			return "The search query must be at least 2 characters long."
		}
		return "One or more parameters have invalid values. Please check your input."
	}

	// Default suggestion
	return "Please check that all required parameters are provided and have valid values."
}

// ClassifyToolError analyzes an error and returns a structured ToolError
func ClassifyToolError(toolName string, args map[string]interface{}, err error) *models.ToolError {
	if err == nil {
		return nil
	}

	toolErr := &models.ToolError{
		ToolName:  toolName,
		Arguments: args,
		Message:   err.Error(),
	}

	// Classify the error type
	switch {
	case isRateLimitError(err):
		toolErr.Type = models.ToolErrorTypeRateLimit
		toolErr.Retryable = true
		toolErr.Suggestion = "You've exceeded the rate limit. Please wait a moment and try again."
	case isTimeoutError(err):
		toolErr.Type = models.ToolErrorTypeTimeout
		toolErr.Retryable = true
		toolErr.Suggestion = "The operation timed out. Please try again."
	case isNetworkError(err):
		toolErr.Type = models.ToolErrorTypeNetwork
		toolErr.Retryable = true
		toolErr.Suggestion = "Network error occurred. Please check your internet connection and try again."
	case isDatabaseError(err):
		toolErr.Type = models.ToolErrorTypeDatabase
		toolErr.Retryable = true
		toolErr.Suggestion = "Database temporarily unavailable. Please try again in a moment."
	case isNotFoundError(err):
		toolErr.Type = models.ToolErrorTypeNotFound
		toolErr.Retryable = false
		toolErr.Suggestion = "The requested resource was not found. Please verify the resource exists."
	case isPermissionError(err):
		toolErr.Type = models.ToolErrorTypePermission
		toolErr.Retryable = false
		toolErr.Suggestion = "You don't have permission to perform this action. Please check your account permissions."
	case isValidationError(err):
		toolErr.Type = models.ToolErrorTypeValidation
		toolErr.Retryable = false
		toolErr.Suggestion = getValidationSuggestion(toolName, args, err)
	default:
		toolErr.Type = models.ToolErrorTypeUnknown
		toolErr.Retryable = true
		toolErr.Suggestion = "An unexpected error occurred. Please try again."
	}

	return toolErr
}

// IsRetryableError checks if an error is retryable based on its type
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable error types
	if isNetworkError(err) || isDatabaseError(err) || isTimeoutError(err) || isRateLimitError(err) {
		return true
	}

	// Not found, permission, and validation errors are not retryable
	return false
}

// WrapToolError wraps a raw error in a ToolError for SSE events
func WrapToolError(toolName string, args map[string]interface{}, err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	toolErr := ClassifyToolError(toolName, args, err)
	if toolErr == nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return toolErr.ToMap()
}

// UnwrapToolError extracts a ToolError from a result map
func UnwrapToolError(result map[string]interface{}) *models.ToolError {
	if result == nil {
		return nil
	}

	errorData, ok := result["error"]
	if !ok {
		return nil
	}

	errorMap, ok := errorData.(map[string]interface{})
	if !ok {
		// Old format: just an error string
		if errStr, ok := errorData.(string); ok {
			return &models.ToolError{
				Type:      models.ToolErrorTypeUnknown,
				Message:   errStr,
				Retryable: true,
				Suggestion: "An error occurred. Please try again.",
			}
		}
		return nil
	}

	toolErr := &models.ToolError{}

	if t, ok := errorMap["type"].(string); ok {
		toolErr.Type = models.ToolErrorType(t)
	}
	if msg, ok := errorMap["message"].(string); ok {
		toolErr.Message = msg
	}
	if retryable, ok := errorMap["retryable"].(bool); ok {
		toolErr.Retryable = retryable
	}
	if toolName, ok := errorMap["tool_name"].(string); ok {
		toolErr.ToolName = toolName
	}
	if args, ok := errorMap["arguments"].(map[string]interface{}); ok {
		toolErr.Arguments = args
	}
	if suggestion, ok := errorMap["suggestion"].(string); ok {
		toolErr.Suggestion = suggestion
	}

	return toolErr
}

// ExecuteToolWithErrorClassification wraps tool execution with error classification
func ExecuteToolWithErrorClassification(executeFunc func() (map[string]interface{}, error), toolName string, args map[string]interface{}) map[string]interface{} {
	result, err := executeFunc()
	if err != nil {
		return WrapToolError(toolName, args, err)
	}
	return result
}

// ToLLMMessage generates a helpful error message for the LLM
// This is designed to help the agent recover from errors more intelligently
func ToLLMMessage(toolName string, args map[string]interface{}, err error) string {
	if err == nil {
		return ""
	}

	toolErr := ClassifyToolError(toolName, args, err)
	if toolErr == nil {
		return fmt.Sprintf("Tool '%s' encountered an error: %s", toolName, err.Error())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Tool Error**: %s\n\n", toolName))
	sb.WriteString(fmt.Sprintf("**Error Type**: %s\n\n", toolErr.Type))
	sb.WriteString(fmt.Sprintf("**What Happened**: %s\n\n", toolErr.Type.Explanation()))
	sb.WriteString(fmt.Sprintf("**Details**: %s\n\n", toolErr.Message))
	sb.WriteString(fmt.Sprintf("**How to Fix This**:\n%s", toolErr.Type.SuggestionForLLM()))

	return sb.String()
}
