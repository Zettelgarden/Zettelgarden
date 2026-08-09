package handlers

import (
	"strings"
	"testing"
)

func TestBuildFileSnippet(t *testing.T) {
	content := strings.Repeat("lorem ipsum ", 50) + "quarterly report" + strings.Repeat(" dolor sit ", 50)

	cases := []struct {
		name        string
		query       string
		fileName    string
		description string
		extracted   string
		tags        []string
		wantField   string
		wantText    string
	}{
		{
			name:      "name match",
			query:     "budget",
			fileName:  "budget-2024.xlsx",
			wantField: "name",
			wantText:  "budget-2024.xlsx",
		},
		{
			name:        "description match",
			query:       "invoices",
			fileName:    "q3.pdf",
			description: "all q3 invoices attached",
			wantField:   "description",
			wantText:    "all q3 invoices attached",
		},
		{
			name:      "content match",
			query:     "quarterly",
			fileName:  "notes.md",
			extracted: content,
			wantField: "content",
			wantText:  "quarterly report",
		},
		{
			name:      "tag match",
			query:     "tax",
			fileName:  "receipt.jpg",
			tags:      []string{"taxes-2024"},
			wantField: "tag",
			wantText:  "taxes-2024",
		},
		{
			name:      "no match",
			query:     "zzzz",
			fileName:  "notes.md",
			extracted: "nothing here",
			wantField: "",
			wantText:  "",
		},
		{
			name:      "empty query",
			query:     "",
			fileName:  "notes.md",
			wantField: "",
			wantText:  "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			snippet, field := buildFileSnippet(c.query, c.fileName, c.description, c.extracted, c.tags)
			if field != c.wantField {
				t.Errorf("field = %q, want %q", field, c.wantField)
			}
			if c.wantText == "" {
				if snippet != "" {
					t.Errorf("snippet = %q, want empty", snippet)
				}
				return
			}
			if !strings.Contains(strings.ToLower(snippet), strings.ToLower(c.wantText)) {
				t.Errorf("snippet %q does not contain %q", snippet, c.wantText)
			}
		})
	}
}

func TestSnippetAroundTruncates(t *testing.T) {
	long := strings.Repeat("a", 1000) + "needle" + strings.Repeat("b", 1000)
	snippet := snippetAround(long, 1000, 6)
	if len(snippet) >= len(long) {
		t.Errorf("snippet should be truncated, got %d chars", len(snippet))
	}
	if !strings.HasPrefix(snippet, "…") || !strings.HasSuffix(snippet, "…") {
		t.Errorf("truncated snippet should have ellipses both ends, got %q", snippet)
	}
	if !strings.Contains(snippet, "needle") {
		t.Errorf("snippet should contain the match, got %q", snippet)
	}

	short := "needle here"
	s := snippetAround(short, 0, 6)
	if s != short {
		t.Errorf("short text should not be truncated, got %q", s)
	}
}

func TestIsTextContentType(t *testing.T) {
	textTypes := []string{
		"text/plain", "text/markdown", "text/html; charset=utf-8", "TEXT/CSS",
		"application/json", "application/xml", "application/javascript",
		"application/x-yaml", "application/x-sh", "application/csv",
	}
	for _, ctype := range textTypes {
		if !isTextContentType(ctype) {
			t.Errorf("expected %q to be text", ctype)
		}
	}
	binaryTypes := []string{
		"application/pdf", "image/png", "application/zip", "application/epub+zip",
		"application/octet-stream", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
	for _, ctype := range binaryTypes {
		if isTextContentType(ctype) {
			t.Errorf("expected %q to be binary", ctype)
		}
	}
}

func TestExtractTextFromFile(t *testing.T) {
	got := extractTextFromFile("text/plain", strings.NewReader("hello world"))
	if got != "hello world" {
		t.Errorf("extract = %q, want %q", got, "hello world")
	}
	got = extractTextFromFile("application/pdf", strings.NewReader("should not extract"))
	if got != "" {
		t.Errorf("binary type should return empty, got %q", got)
	}
}
