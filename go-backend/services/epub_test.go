package services

import (
	"os"
	"path/filepath"
	"testing"

	epublib "github.com/ArcadiaLin/go-epub"
)

func TestParseEpub(t *testing.T) {
	// Create a minimal test epub file
	testEpubPath := filepath.Join("testdata", "test.epub")

	// Skip if test file doesn't exist (will be created separately)
	if _, err := os.Stat(testEpubPath); os.IsNotExist(err) {
		t.Skip("Test epub file not found, skipping")
	}

	metadata, chapters, err := ParseEpub(testEpubPath)
	if err != nil {
		t.Fatalf("ParseEpub failed: %v", err)
	}

	if metadata.Title == "" {
		t.Error("Expected non-empty title")
	}

	if len(chapters) == 0 {
		t.Error("Expected at least one chapter")
	}

	for i, chapter := range chapters {
		if chapter.Title == "" {
			t.Errorf("Chapter %d has empty title", i)
		}
		if chapter.Body == "" {
			t.Errorf("Chapter %d has empty body", i)
		}
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024", "2024"},
		{"2024-01-15", "2024"},
		{"Published: 2023", "2023"},
		{"1999-12-31T23:59:59Z", "1999"},
		{"invalid", ""},
		{"", ""},
		{"1800", ""}, // Should not match years before 1900
		{"2100", ""}, // Should not match years after 2099
	}

	for _, tt := range tests {
		result := extractYear(tt.input)
		if result != tt.expected {
			t.Errorf("extractYear(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHtmlToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "br tags preserved",
			input:    "<p>Line one<br>Line two<br>Line three</p>",
			expected: "Line one\nLine two\nLine three",
		},
		{
			name:     "paragraph breaks",
			input:    "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph\n\nSecond paragraph",
		},
		{
			name:     "html entities",
			input:    "<p>Quote: &quot;Hello&quot; &amp; goodbye</p>",
			expected: "Quote: \"Hello\" & goodbye",
		},
		{
			name:     "strips tags",
			input:    "<p><b>Bold</b> and <i>italic</i></p>",
			expected: "Bold and italic",
		},
		{
			name:     "multiple br tags",
			input:    "<p>Line 1<br/>Line 2<BR>Line 3<br />Line 4</p>",
			expected: "Line 1\nLine 2\nLine 3\nLine 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlToMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("htmlToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParagraphsToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single paragraph",
			input:    []string{"Hello world"},
			expected: "Hello world",
		},
		{
			name:     "multiple paragraphs",
			input:    []string{"First paragraph", "Second paragraph"},
			expected: "First paragraph\n\nSecond paragraph",
		},
		{
			name:     "with whitespace",
			input:    []string{"  Trimmed  ", "  Another  "},
			expected: "Trimmed\n\nAnother",
		},
		{
			name:     "with empty strings",
			input:    []string{"First", "", "Second"},
			expected: "First\n\nSecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paragraphsToMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("paragraphsToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetChapterTitle(t *testing.T) {
	tests := []struct {
		title    string
		index    int
		expected string
	}{
		{"Introduction", 1, "Introduction"},
		{"", 1, "Chapter 1"},
		{"", 5, "Chapter 5"},
	}

	for _, tt := range tests {
		chapter := &epublib.Chapter{Title: tt.title}
		result := getChapterTitle(chapter, tt.index)
		if result != tt.expected {
			t.Errorf("getChapterTitle(%q, %d) = %q, want %q", tt.title, tt.index, result, tt.expected)
		}
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are reasonable values
	if MinChapterContent <= 0 {
		t.Error("MinChapterContent should be positive")
	}
}

func TestErrNoValidChapters(t *testing.T) {
	// Verify the error is defined correctly
	if ErrNoValidChapters == nil {
		t.Error("ErrNoValidChapters should not be nil")
	}
	if ErrNoValidChapters.Error() != "no valid chapters found in epub" {
		t.Errorf("ErrNoValidChapters message = %q, want %q", ErrNoValidChapters.Error(), "no valid chapters found in epub")
	}
}
