package handlers

import (
	"testing"
)

func TestIsToolResultEmpty(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected bool
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: true,
		},
		{
			name:     "empty map",
			result:   map[string]interface{}{},
			expected: true,
		},
		{
			name:     "only empty string",
			result:   map[string]interface{}{"data": ""},
			expected: true,
		},
		{
			name:     "only whitespace string",
			result:   map[string]interface{}{"data": "   "},
			expected: true,
		},
		{
			name:     "empty array",
			result:   map[string]interface{}{"data": []interface{}{}},
			expected: true,
		},
		{
			name:     "empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{}},
			expected: true,
		},
		{
			name:     "only error field",
			result:   map[string]interface{}{"error": "some error"},
			expected: true,
		},
		{
			name:     "non-empty string",
			result:   map[string]interface{}{"data": "some value"},
			expected: false,
		},
		{
			name:     "non-empty array",
			result:   map[string]interface{}{"data": []interface{}{"item"}},
			expected: false,
		},
		{
			name:     "non-empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{"key": "value"}},
			expected: false,
		},
		{
			name:     "numeric value",
			result:   map[string]interface{}{"count": 42},
			expected: false,
		},
		{
			name:     "boolean value",
			result:   map[string]interface{}{"success": true},
			expected: false,
		},
		{
			name:     "mixed empty and non-empty",
			result:   map[string]interface{}{"empty": "", "data": "value"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolResultEmpty(tt.result)
			if result != tt.expected {
				t.Errorf("isToolResultEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}