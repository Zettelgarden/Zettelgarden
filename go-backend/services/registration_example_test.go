package services

import (
	"encoding/json"
	"testing"
)

// TestBuildParams verifies the buildParams function generates correct JSON Schema
func TestBuildParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []ToolParam
		expected map[string]interface{}
	}{
		{
			name: "String parameter with required",
			params: []ToolParam{
				{Name: "query", Type: "string", Required: true, Desc: "Search query"},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			name: "Integer parameter with min/max and default",
			params: []ToolParam{
				{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50), Desc: "Max results"},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Max results",
						"default":     10,
						"minimum":     1,
						"maximum":     50,
					},
				},
			},
		},
		{
			name: "String parameter with enum and default",
			params: []ToolParam{
				{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search_type": map[string]interface{}{
						"type":    "string",
						"default": "semantic",
						"enum":    []interface{}{"text", "semantic"},
					},
				},
			},
		},
		{
			name: "Multiple parameters of different types",
			params: []ToolParam{
				{Name: "query", Type: "string", Required: true, Desc: "Search query"},
				{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50)},
				{Name: "verbose", Type: "boolean", Required: false, Default: false},
				{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":    "integer",
						"default": 10,
						"minimum": 1,
						"maximum": 50,
					},
					"verbose": map[string]interface{}{
						"type":    "boolean",
						"default": false,
					},
					"search_type": map[string]interface{}{
						"type":    "string",
						"default": "semantic",
						"enum":    []interface{}{"text", "semantic"},
					},
				},
				"required": []string{"query"},
			},
		},
		{
			name: "Array parameter with items",
			params: []ToolParam{
				{
					Name:     "tags",
					Type:     "array",
					Required: false,
					Desc:     "List of tags",
					Items: &ToolParam{
						Type: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "List of tags",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
		{
			name: "Desc takes precedence over Description",
			params: []ToolParam{
				{Name: "query", Type: "string", Required: true, Description: "Long description", Desc: "Short description"},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Short description",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildParams(tt.params)

			// Compare JSON to avoid issues with nested map comparisons
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result: %v", err)
			}
			expectedJSON, err := json.Marshal(tt.expected)
			if err != nil {
				t.Fatalf("Failed to marshal expected: %v", err)
			}

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("buildParams() mismatch\nGot:      %s\nExpected: %s", resultJSON, expectedJSON)
			}
		})
	}
}

// TestIntPtr verifies intPtr helper works correctly
func TestIntPtr(t *testing.T) {
	val := 42
	ptr := intPtr(val)
	if ptr == nil {
		t.Fatal("intPtr returned nil")
	}
	if *ptr != val {
		t.Errorf("intPtr(%d) = %d, want %d", val, *ptr, val)
	}
}

// TestRegisterTool verifies RegisterTool correctly registers a tool
func TestRegisterTool(t *testing.T) {
	registry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	handler := func(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	}

	RegisterTool(registry,
		"test_tool",
		"A test tool",
		handler,
		ToolParam{Name: "input", Type: "string", Required: true, Desc: "Input parameter"},
		ToolParam{Name: "count", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(100)},
	)

	// Verify tool was registered
	tool, exists := registry.tools["test_tool"]
	if !exists {
		t.Fatal("RegisterTool did not register the tool")
	}

	// Verify tool definition
	if tool.Definition.Function.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Definition.Function.Name)
	}

	if tool.Definition.Function.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", tool.Definition.Function.Description)
	}

	// Verify parameters
	params := tool.Definition.Function.Parameters
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		t.Fatal("Parameters is not a map")
	}
	if paramsMap["type"] != "object" {
		t.Errorf("Expected parameter type 'object', got '%v'", paramsMap["type"])
	}

	properties, ok := paramsMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters properties is not a map")
	}

	// Check input parameter
	inputParam, ok := properties["input"].(map[string]interface{})
	if !ok {
		t.Fatal("Input parameter not found")
	}
	if inputParam["type"] != "string" {
		t.Errorf("Expected input type 'string', got '%v'", inputParam["type"])
	}

	// Check count parameter
	countParam, ok := properties["count"].(map[string]interface{})
	if !ok {
		t.Fatal("Count parameter not found")
	}
	if countParam["type"] != "integer" {
		t.Errorf("Expected count type 'integer', got '%v'", countParam["type"])
	}
	if countParam["default"] != 10 {
		t.Errorf("Expected count default 10, got %v", countParam["default"])
	}

	// Verify required array
	required, ok := paramsMap["required"].([]string)
	if !ok {
		// The JSON unmarshaling might create []interface{} instead of []string
		requiredInterface, okInterface := paramsMap["required"].([]interface{})
		if !okInterface {
			t.Fatalf("Required is not an array, got %T", paramsMap["required"])
		}
		if len(requiredInterface) != 1 || requiredInterface[0] != "input" {
			t.Errorf("Expected required ['input'], got %v", requiredInterface)
		}
		return
	}
	if len(required) != 1 || required[0] != "input" {
		t.Errorf("Expected required ['input'], got %v", required)
	}
}
