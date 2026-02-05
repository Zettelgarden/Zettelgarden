// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import (
	"encoding/json"
	"fmt"
)

// Parameter extraction helpers

// getIntParam extracts an integer parameter from args
func getIntParam(args map[string]interface{}, key string) (int, error) {
	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s parameter is required", key)
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid %s format: %v", key, err)
		}
		return int(i), nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

// getOptionalIntParam extracts an optional integer parameter from args
func getOptionalIntParam(args map[string]interface{}, key string) (int, bool, error) {
	val, ok := args[key]
	if !ok {
		return 0, false, nil
	}
	switch v := val.(type) {
	case float64:
		return int(v), true, nil
	case int:
		return v, true, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false, fmt.Errorf("invalid %s format: %v", key, err)
		}
		return int(i), true, nil
	default:
		return 0, false, fmt.Errorf("%s must be an integer", key)
	}
}

// getStringParam extracts a string parameter from args
func getStringParam(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s parameter is required", key)
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("%s must be a string", key)
}

// getOptionalStringParam extracts an optional string parameter from args
func getOptionalStringParam(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key]
	if !ok {
		return "", false
	}
	if str, ok := val.(string); ok {
		return str, true
	}
	return "", false
}

// getBoolParam extracts a boolean parameter from args
func getBoolParam(args map[string]interface{}, key string) (bool, error) {
	val, ok := args[key]
	if !ok {
		return false, fmt.Errorf("%s parameter is required", key)
	}
	if b, ok := val.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("%s must be a boolean", key)
}

// getOptionalBoolParam extracts an optional boolean parameter from args
func getOptionalBoolParam(args map[string]interface{}, key string) bool {
	val, ok := args[key]
	if !ok {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}
