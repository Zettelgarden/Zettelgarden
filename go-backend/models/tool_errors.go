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
