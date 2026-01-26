package services

import (
	"go-backend/models"
	"reflect"
	"testing"
)

// TestRemoveFactsFromAnalyses tests the RemoveFactsFromAnalyses function
func TestRemoveFactsFromAnalyses(t *testing.T) {
	testCases := []struct {
		name     string
		input    []models.SectionAnalysis
		expected []models.SectionAnalysis
		facts    []string
	}{
		{
			name: "single section with single thesis",
			input: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis: "Thesis 1",
							Facts:  []string{"fact1", "fact2"},
							Arguments: []models.Argument{
								{Argument: "arg1", Importance: 8},
							},
						},
					},
				},
			},
			expected: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
					},
				},
			},
			facts: []string{"fact1", "fact2"},
		},
		{
			name: "multiple sections with multiple theses",
			input: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis: "Thesis 1",
							Facts:  []string{"fact1"},
							Arguments: []models.Argument{
								{Argument: "arg1", Importance: 8},
							},
						},
						{
							Thesis: "Thesis 2",
							Facts:  []string{"fact2", "fact3"},
							Arguments: []models.Argument{
								{Argument: "arg2", Importance: 5},
							},
						},
					},
				},
				{
					Section: "Section 2",
					Theses: []models.ThesisEntry{
						{
							Thesis: "Thesis 3",
							Facts:  []string{"fact4"},
							Arguments: []models.Argument{
								{Argument: "arg3", Importance: 7},
							},
						},
					},
				},
			},
			expected: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
						{
							Thesis:    "Thesis 2",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg2", Importance: 5}},
						},
					},
				},
				{
					Section: "Section 2",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 3",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg3", Importance: 7}},
						},
					},
				},
			},
			facts: []string{"fact1", "fact2", "fact3", "fact4"},
		},
		{
			name: "empty facts array",
			input: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
					},
				},
			},
			expected: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
					},
				},
			},
			facts: nil, // nil because the function returns nil when no facts are collected
		},
		{
			name:     "empty input",
			input:    []models.SectionAnalysis{},
			expected: []models.SectionAnalysis{},
			facts:    nil, // nil because the function returns nil when no facts are collected
		},
		{
			name: "nil facts in thesis",
			input: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     nil,
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
					},
				},
			},
			expected: []models.SectionAnalysis{
				{
					Section: "Section 1",
					Theses: []models.ThesisEntry{
						{
							Thesis:    "Thesis 1",
							Facts:     []string{},
							Arguments: []models.Argument{{Argument: "arg1", Importance: 8}},
						},
					},
				},
			},
			facts: nil, // nil because the function returns nil when no facts are collected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, facts := RemoveFactsFromAnalyses(tc.input)

			// Check that facts were extracted correctly
			// Handle nil vs empty slice comparison
			if !reflect.DeepEqual(facts, tc.facts) && len(facts) != len(tc.facts) {
				t.Errorf("Facts mismatch:\nGot:      %v\nExpected: %v", facts, tc.facts)
			}

			// Check that the result structure matches expected
			if len(result) != len(tc.expected) {
				t.Fatalf("Result length mismatch: got %d, expected %d", len(result), len(tc.expected))
			}

			for i, section := range result {
				if section.Section != tc.expected[i].Section {
					t.Errorf("Section %d: expected section name %q, got %q", i, tc.expected[i].Section, section.Section)
				}

				if len(section.Theses) != len(tc.expected[i].Theses) {
					t.Errorf("Section %d: expected %d theses, got %d", i, len(tc.expected[i].Theses), len(section.Theses))
				}

				for j, thesis := range section.Theses {
					if thesis.Thesis != tc.expected[i].Theses[j].Thesis {
						t.Errorf("Section %d, Thesis %d: expected thesis %q, got %q", i, j, tc.expected[i].Theses[j].Thesis, thesis.Thesis)
					}

					// Facts should always be empty in result
					if len(thesis.Facts) != 0 {
						t.Errorf("Section %d, Thesis %d: expected empty facts array, got %v", i, j, thesis.Facts)
					}

					// Arguments should be preserved
					if !reflect.DeepEqual(thesis.Arguments, tc.expected[i].Theses[j].Arguments) {
						t.Errorf("Section %d, Thesis %d: arguments not preserved", i, j)
					}
				}
			}
		})
	}
}

// TestRemoveFactsFromAnalysesPreservesOriginal tests that the original input is not modified
func TestRemoveFactsFromAnalysesPreservesOriginal(t *testing.T) {
	input := []models.SectionAnalysis{
		{
			Section: "Section 1",
			Theses: []models.ThesisEntry{
				{
					Thesis: "Thesis 1",
					Facts:  []string{"fact1", "fact2"},
					Arguments: []models.Argument{
						{Argument: "arg1", Importance: 8},
					},
				},
			},
		},
	}

	// Store original facts for comparison
	originalFacts := make([]string, len(input[0].Theses[0].Facts))
	copy(originalFacts, input[0].Theses[0].Facts)

	// Call the function
	RemoveFactsFromAnalyses(input)

	// Check that original was not modified
	if !reflect.DeepEqual(input[0].Theses[0].Facts, originalFacts) {
		t.Errorf("Original input was modified:\nExpected: %v\nGot:      %v", originalFacts, input[0].Theses[0].Facts)
	}
}
