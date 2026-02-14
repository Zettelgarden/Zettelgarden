package models

type ToolErrorType string

const (
	ToolErrorTypeNetwork    ToolErrorType = "network"
	ToolErrorTypeValidation ToolErrorType = "validation"
	ToolErrorTypeDatabase   ToolErrorType = "database"
	ToolErrorTypeNotFound   ToolErrorType = "not_found"
	ToolErrorTypePermission ToolErrorType = "permission"
	ToolErrorTypeRateLimit  ToolErrorType = "rate_limit"
	ToolErrorTypeTimeout    ToolErrorType = "timeout"
	ToolErrorTypeUnknown    ToolErrorType = "unknown"
)

// Explanation returns a human-readable explanation of what this error type means
func (t ToolErrorType) Explanation() string {
	switch t {
	case ToolErrorTypeNotFound:
		return "The resource you're looking for doesn't exist. This could mean the ID is incorrect, the resource was deleted, or it never existed."
	case ToolErrorTypeValidation:
		return "The parameters provided were incorrect. This could mean missing required fields, invalid formats, or values that don't meet the requirements."
	case ToolErrorTypeNetwork:
		return "A network connectivity issue occurred. This could be due to internet connection problems, DNS issues, or the remote service being unreachable."
	case ToolErrorTypeDatabase:
		return "A database error occurred. The database may be temporarily unavailable, overloaded, or experiencing connection issues."
	case ToolErrorTypePermission:
		return "You don't have permission to perform this action. This could be due to insufficient privileges, ownership restrictions, or account limitations."
	case ToolErrorTypeRateLimit:
		return "You've exceeded the rate limit for this service. You're making too many requests too quickly."
	case ToolErrorTypeTimeout:
		return "The operation took too long to complete. The service or resource may be slow, or the request may be too complex."
	case ToolErrorTypeUnknown:
		return "An unexpected error occurred. The system encountered an issue that doesn't fit into standard error categories."
	default:
		return "An error occurred."
	}
}

// SuggestionForLLM returns an actionable suggestion for the LLM on how to recover
func (t ToolErrorType) SuggestionForLLM() string {
	switch t {
	case ToolErrorTypeNotFound:
		return "1. Verify the resource ID is correct\n2. Try searching for the resource instead of accessing it directly\n3. Use broader search criteria\n4. Ask the user to confirm the resource exists"
	case ToolErrorTypeValidation:
		return "1. Check the required parameters for this tool\n2. Ensure all required fields are provided\n3. Verify parameter formats match expectations\n4. Review the tool's schema"
	case ToolErrorTypeNetwork:
		return "1. The network issue may be temporary - try again\n2. If this persists, the external service may be down\n3. Consider using a different tool or approach\n4. Inform the user about the connectivity issue"
	case ToolErrorTypeDatabase:
		return "1. This is usually temporary - wait a moment and try again\n2. If the problem persists, there may be a system issue\n3. Try a simpler query if possible\n4. Inform the user about the database issue"
	case ToolErrorTypePermission:
		return "1. This error cannot be retried - it will continue to fail\n2. The user may not have the required access level\n3. Consider asking the user to check their permissions\n4. Try an alternative approach that doesn't require this permission"
	case ToolErrorTypeRateLimit:
		return "1. Wait a moment before trying again\n2. Reduce the frequency of tool calls\n3. Batch operations if possible\n4. Inform the user about the rate limiting"
	case ToolErrorTypeTimeout:
		return "1. The operation may be too complex - try breaking it into smaller steps\n2. If querying data, try reducing the scope\n3. Network delays may be causing issues\n4. Try again with a simpler request"
	case ToolErrorTypeUnknown:
		return "1. Try a different approach or tool\n2. If the problem persists, inform the user\n3. Check if there are any syntax or parameter issues\n4. Consider asking the user for clarification"
	default:
		return "Try again or use a different approach."
	}
}

type ToolError struct {
	Type       ToolErrorType          `json:"type"`
	Message    string                 `json:"message"`
	Retryable  bool                   `json:"retryable"`
	ToolName   string                 `json:"tool_name"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`
	Suggestion string                 `json:"suggestion,omitempty"`
}

func (e *ToolError) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"type":        e.Type,
			"message":     e.Message,
			"retryable":   e.Retryable,
			"tool_name":   e.ToolName,
			"arguments":   e.Arguments,
			"suggestion":  e.Suggestion,
		},
	}
}

// HasError checks if a tool result map contains an error
func HasError(result map[string]interface{}) bool {
	if result == nil {
		return false
	}
	_, hasError := result["error"]
	return hasError
}
