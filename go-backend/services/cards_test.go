package services

import (
	"reflect"
	"testing"
)

func TestExtractBacklinks(t *testing.T) {
	text := "This is a sample text with [link1] and [another link]."
	expected := []string{"link1", "another link"}
	result := ExtractBacklinks(text)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestGetParentCardId(t *testing.T) {
	// Test cases for new dual-format function
	testCases := []struct {
		name     string
		cardID   string
		expected string
	}{
		// Old format (alternating separators) test cases
		{"Old format - complex hierarchy", "SP24/P.19", "SP24/P"},
		{"Old format - simple hierarchy", "1957/A.135", "1957/A"},
		{"Old format - deep hierarchy", "1957/A.135/B.2", "1957/A.135/B"},
		{"Old format - very deep", "SP170/A.1/A.1/A.1/A.1", "SP170/A.1/A.1/A.1/A"},
		{"Old format - root card", "SP24", "SP24"},
		{"Old format - root card numeric", "1957", "1957"},
		{"Old format - single level", "1", "1"},

		// New format test cases
		{"New format - dot separators", "cardA.1.2", "cardA.1"},
		{"New format - slash separators", "cardA/1/2", "cardA/1"},
		{"New format - dash separators", "cardA-1-2", "cardA-1"},
		{"New format - mixed separators", "cardA.1/2", "cardA.1"},
		{"New format - mixed separators 2", "cardA/1.2", "cardA/1"},
		{"New format - mixed separators 3", "cardA-1.2", "cardA-1"},
		{"New format - single level", "cardA.1", "cardA"},
		{"New format - single level slash", "cardA/1", "cardA"},
		{"New format - root card", "cardA", "cardA"},
		{"New format - complex name", "myProject.v2.1", "myProject.v2"},
		{"New format - deep hierarchy", "project.1.2.3", "project.1.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DiscoverParentId(tc.cardID)
			if result != tc.expected {
				t.Errorf("function returned wrong result for %q, got %v want %v", tc.cardID, result, tc.expected)
			}
		})
	}
}
