package handlers

import (
	"go-backend/models"
	"go-backend/tests"
	"strings"
	"testing"
)

func TestRemoveReferences(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single reference line",
			input:    "Some text\n[1/A.1] - Card Title\nMore text",
			expected: "Some text\nMore text",
		},
		{
			name:     "multiple reference lines",
			input:    "Some text\n[1] - First Card\n[2/A] - Second Card\nMore text\n[3/B.1] - Third Card\nEnd text",
			expected: "Some text\nMore text\nEnd text",
		},
		{
			name:     "no references",
			input:    "Some text\nMore text\nEnd text",
			expected: "Some text\nMore text\nEnd text",
		},
		{
			name:     "reference at start",
			input:    "[1] - First Card\nSome text\nMore text",
			expected: "Some text\nMore text",
		},
		{
			name:     "reference at end",
			input:    "Some text\nMore text\n[1] - Last Card\n",
			expected: "Some text\nMore text",
		},
		{
			name:     "complex reference tags",
			input:    "Text\n[REF001] - Reference Card\n[MM001/A.1.2] - Complex Reference\nMore text",
			expected: "Text\nMore text",
		},
		{
			name:     "reference with special characters in title",
			input:    "Text\n[1] - Card with & special $symbols!\nMore text",
			expected: "Text\nMore text",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only references",
			input:    "[1] - First Card\n[2] - Second Card\n",
			expected: "",
		},
		{
			name:     "malformed reference (missing dash)",
			input:    "Text\n[1] Card Title\nMore text",
			expected: "Text\n[1] Card Title\nMore text",
		},
		{
			name:     "malformed reference (missing bracket)",
			input:    "Text\n1] - Card Title\nMore text",
			expected: "Text\n1] - Card Title\nMore text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeReferences(tc.input)
			if result != tc.expected {
				t.Errorf("removeReferences() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestRemoveReferencesPreservesContent(t *testing.T) {
	// Test that content with bracket patterns that aren't references is preserved
	input := `This is a test document.

Some content here with [brackets] in it.

[1] - This is a reference and should be removed
More content.

Text with [square brackets] should remain.

[2/A.1] - Another reference to remove

Final paragraph with some [notation] intact.`

	expected := `This is a test document.

Some content here with [brackets] in it.

More content.

Text with [square brackets] should remain.

Final paragraph with some [notation] intact.`

	result := removeReferences(input)
	if result != expected {
		t.Errorf("removeReferences() failed to preserve non-reference brackets.\nGot:\n%s\nWant:\n%s", result, expected)
	}
}

func TestRemoveReferencesRealWorldExample(t *testing.T) {
	// Test with a realistic card body that might contain references
	input := `# Meeting Notes

## Key Points
- Discussed project timeline
- Need to review budget

[PROJECT-001] - Budget Review Card
[MEETING/2024-01] - Previous Meeting Notes

## Action Items
1. Follow up with team
2. Schedule next meeting

[TEAM-CONTACT] - Team Contact Information

End of notes.`

	expected := `# Meeting Notes

## Key Points
- Discussed project timeline
- Need to review budget

## Action Items
1. Follow up with team
2. Schedule next meeting

End of notes.`

	result := removeReferences(input)
	if result != expected {
		t.Errorf("removeReferences() failed on real-world example.\nGot:\n%s\nWant:\n%s", result, expected)
	}
}

func TestPrepareTextForAnalysis(t *testing.T) {
	testCases := []struct {
		name     string
		title    string
		body     string
		expected string
	}{
		{
			name:     "with title and body",
			title:    "Meeting Notes",
			body:     "Discussed project timeline\n[REF-001] - Previous Meeting\nAction items listed",
			expected: "# Meeting Notes\n\nDiscussed project timeline\nAction items listed",
		},
		{
			name:     "empty title",
			title:    "",
			body:     "Just some content\n[REF-001] - Reference Card\nMore content",
			expected: "Just some content\nMore content",
		},
		{
			name:     "title with references in body",
			title:    "Important Document",
			body:     "[CARD-123] - Related Info\nMain content here\n[CARD-456] - Another Reference",
			expected: "# Important Document\n\nMain content here",
		},
		{
			name:     "only title, empty body",
			title:    "Title Only",
			body:     "",
			expected: "# Title Only",
		},
		{
			name:     "both empty",
			title:    "",
			body:     "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := prepareTextForAnalysis(tc.title, tc.body)
			if result != tc.expected {
				t.Errorf("prepareTextForAnalysis() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestPrepareTextForAnalysisPreservesMarkdown(t *testing.T) {
	title := "Research Notes"
	body := `## Introduction
This is important research.

[REF-001] - Related Study

### Findings
- Point 1
- Point 2

[REF-002] - Another Reference

## Conclusion
Final thoughts here.`

	expected := `# Research Notes

## Introduction
This is important research.

### Findings
- Point 1
- Point 2

## Conclusion
Final thoughts here.`

	result := prepareTextForAnalysis(title, body)
	if result != expected {
		t.Errorf("prepareTextForAnalysis() failed to preserve markdown structure.\nGot:\n%s\nWant:\n%s", result, expected)
	}
}

// TestSaveAnalysisSuccess tests successful save of analysis data
func TestSaveAnalysisSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	// Create a summarization record first
	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, cardPK, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{
			Section: "Section 1: Introduction",
			Theses: []models.ThesisEntry{
				{
					Thesis: "This is the first thesis",
					Facts:  []string{"fact1", "fact2"},
					Arguments: []models.Argument{
						{Argument: "argument 1", Importance: 8},
						{Argument: "argument 2", Importance: 5},
					},
				},
			},
		},
	}

	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err != nil {
		t.Fatalf("SaveAnalysis failed: %v", err)
	}

	// Verify data was saved correctly
	var sectionCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_sections
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&sectionCount)
	if err != nil {
		t.Fatalf("Failed to query sections: %v", err)
	}
	if sectionCount != 1 {
		t.Errorf("Expected 1 section, got %d", sectionCount)
	}

	var thesisCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_theses
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&thesisCount)
	if err != nil {
		t.Fatalf("Failed to query theses: %v", err)
	}
	if thesisCount != 1 {
		t.Errorf("Expected 1 thesis, got %d", thesisCount)
	}

	var argumentCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_arguments
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&argumentCount)
	if err != nil {
		t.Fatalf("Failed to query arguments: %v", err)
	}
	if argumentCount != 2 {
		t.Errorf("Expected 2 arguments, got %d", argumentCount)
	}
}

// TestSaveAnalysisEmptyThesisSkipped tests that empty theses are skipped
func TestSaveAnalysisEmptyThesisSkipped(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, cardPK, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{
			Section: "Section 1: Introduction",
			Theses: []models.ThesisEntry{
				{
					Thesis: "Valid thesis",
					Facts:  []string{"fact1"},
					Arguments: []models.Argument{
						{Argument: "argument 1", Importance: 8},
					},
				},
				{
					Thesis:    "", // Empty thesis - should be skipped
					Facts:     []string{"fact2"},
					Arguments: []models.Argument{{Argument: "argument 2", Importance: 5}},
				},
				{
					Thesis: "   ", // Whitespace-only thesis - should be skipped
					Facts:  []string{"fact3"},
					Arguments: []models.Argument{
						{Argument: "argument 3", Importance: 7},
					},
				},
			},
		},
	}

	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err != nil {
		t.Fatalf("SaveAnalysis failed: %v", err)
	}

	// Only the valid thesis should be saved
	var thesisCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_theses
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&thesisCount)
	if err != nil {
		t.Fatalf("Failed to query theses: %v", err)
	}
	if thesisCount != 1 {
		t.Errorf("Expected 1 thesis (empty ones skipped), got %d", thesisCount)
	}
}

// TestSaveAnalysisEmptySectionSkipped tests that sections with empty titles are skipped
func TestSaveAnalysisEmptySectionSkipped(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, cardPK, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{
			Section: "Section 1: Introduction",
			Theses: []models.ThesisEntry{
				{
					Thesis: "Valid thesis in valid section",
					Facts:  []string{"fact1"},
					Arguments: []models.Argument{
						{Argument: "argument 1", Importance: 8},
					},
				},
			},
		},
		{
			Section: "", // Empty section - should be skipped
			Theses: []models.ThesisEntry{
				{
					Thesis: "Thesis in empty section",
					Facts:  []string{"fact2"},
					Arguments: []models.Argument{
						{Argument: "argument 2", Importance: 5},
					},
				},
			},
		},
		{
			Section: "   ", // Whitespace-only section - should be skipped
			Theses: []models.ThesisEntry{
				{
					Thesis: "Thesis in whitespace section",
					Facts:  []string{"fact3"},
					Arguments: []models.Argument{
						{Argument: "argument 3", Importance: 7},
					},
				},
			},
		},
	}

	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err != nil {
		t.Fatalf("SaveAnalysis failed: %v", err)
	}

	// Only the valid section should be saved
	var sectionCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_sections
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&sectionCount)
	if err != nil {
		t.Fatalf("Failed to query sections: %v", err)
	}
	if sectionCount != 1 {
		t.Errorf("Expected 1 section (empty ones skipped), got %d", sectionCount)
	}
}

// TestSaveAnalysisInvalidCardPK tests that invalid cardPK values are rejected
func TestSaveAnalysisInvalidCardPK(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1

	// First create a valid summarization (with card_pk = 1)
	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, 1, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{
			Section: "Section 1: Introduction",
			Theses: []models.ThesisEntry{
				{
					Thesis: "This is a thesis",
					Facts:  []string{"fact1"},
					Arguments: []models.Argument{
						{Argument: "argument 1", Importance: 8},
					},
				},
			},
		},
	}

	// Test with cardPK = 0 (invalid)
	cardPK := 0
	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err == nil {
		t.Error("Expected error for invalid cardPK (0), but got none")
	}
	// Check that the error message mentions cardPK
	if err != nil && !strings.Contains(err.Error(), "card_pk") {
		t.Errorf("Error message should mention cardPK, got: %v", err)
	}

	// Test negative cardPK
	cardPK = -1
	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err == nil {
		t.Error("Expected error for invalid cardPK (-1), but got none")
	}
}

// TestSaveAnalysisMultipleSections tests saving multiple sections
func TestSaveAnalysisMultipleSections(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, cardPK, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{
			Section: "Section 1: Introduction",
			Theses: []models.ThesisEntry{
				{
					Thesis: "First thesis in section 1",
					Facts:  []string{"fact1"},
					Arguments: []models.Argument{
						{Argument: "argument 1", Importance: 8},
					},
				},
			},
		},
		{
			Section: "Section 2: Analysis",
			Theses: []models.ThesisEntry{
				{
					Thesis: "First thesis in section 2",
					Facts:  []string{"fact2"},
					Arguments: []models.Argument{
						{Argument: "argument 2", Importance: 7},
					},
				},
				{
					Thesis: "Second thesis in section 2",
					Facts:  []string{"fact3"},
					Arguments: []models.Argument{
						{Argument: "argument 3", Importance: 6},
						{Argument: "argument 4", Importance: 9},
					},
				},
			},
		},
	}

	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err != nil {
		t.Fatalf("SaveAnalysis failed: %v", err)
	}

	// Verify all sections were saved
	var sectionCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_sections
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&sectionCount)
	if err != nil {
		t.Fatalf("Failed to query sections: %v", err)
	}
	if sectionCount != 2 {
		t.Errorf("Expected 2 sections, got %d", sectionCount)
	}

	// Verify all theses were saved (1 in section 1, 2 in section 2)
	var thesisCount int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM summary_theses
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
	`, userID, cardPK, summarizationID).Scan(&thesisCount)
	if err != nil {
		t.Fatalf("Failed to query theses: %v", err)
	}
	if thesisCount != 3 {
		t.Errorf("Expected 3 theses, got %d", thesisCount)
	}
}

// TestSaveAnalysisSectionOrder tests that sections maintain their order
func TestSaveAnalysisSectionOrder(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	var summarizationID int
	err := s.Server.Tx.QueryRow(`
		INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'complete', NOW(), NOW())
		RETURNING id
	`, userID, cardPK, "").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("Failed to create summarization: %v", err)
	}

	analyses := []models.SectionAnalysis{
		{Section: "First Section", Theses: []models.ThesisEntry{{Thesis: "Thesis 1", Facts: []string{}, Arguments: []models.Argument{}}}},
		{Section: "Second Section", Theses: []models.ThesisEntry{{Thesis: "Thesis 2", Facts: []string{}, Arguments: []models.Argument{}}}},
		{Section: "Third Section", Theses: []models.ThesisEntry{{Thesis: "Thesis 3", Facts: []string{}, Arguments: []models.Argument{}}}},
	}

	err = s.SaveAnalysis(userID, cardPK, summarizationID, analyses)
	if err != nil {
		t.Fatalf("SaveAnalysis failed: %v", err)
	}

	// Verify section order is maintained
	rows, err := s.Server.Tx.Query(`
		SELECT section_title, section_order FROM summary_sections
		WHERE user_id = $1 AND card_pk = $2 AND summarization_id = $3
		ORDER BY section_order
	`, userID, cardPK, summarizationID)
	if err != nil {
		t.Fatalf("Failed to query sections: %v", err)
	}
	defer rows.Close()

	expectedSections := []string{"First Section", "Second Section", "Third Section"}
	var actualSections []string
	for rows.Next() {
		var title string
		var order int
		if err := rows.Scan(&title, &order); err != nil {
			t.Fatalf("Failed to scan section: %v", err)
		}
		actualSections = append(actualSections, title)
	}

	if len(actualSections) != len(expectedSections) {
		t.Fatalf("Expected %d sections, got %d", len(expectedSections), len(actualSections))
	}

	for i, expected := range expectedSections {
		if actualSections[i] != expected {
			t.Errorf("Section %d: expected %q, got %q", i, expected, actualSections[i])
		}
	}
}