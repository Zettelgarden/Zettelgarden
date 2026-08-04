package handlers

import (
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
