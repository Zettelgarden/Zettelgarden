package backlink

import (
	"testing"
)

func TestExtractBacklinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		// Wiki-link syntax
		{
			name:     "basic wiki-link",
			input:    "This has [[link1]] and [[link2]]",
			expected: []string{"link1", "link2"},
		},
		{
			name:     "wiki-link with display text",
			input:    "See [[42|Meeting Notes]] for details",
			expected: []string{"42"},
		},
		{
			name:     "wiki-link with child card id",
			input:    "Related to [[1.3]]",
			expected: []string{"1.3"},
		},
		{
			name:     "empty wiki-link excluded",
			input:    "This has [[]] in it",
			expected: []string{},
		},
		{
			name:     "wiki-link trims whitespace",
			input:    "See [[ 42 ]] here",
			expected: []string{"42"},
		},

		// Legacy syntax
		{
			name:     "legacy basic link",
			input:    "This has [link1] and [link2]",
			expected: []string{"link1", "link2"},
		},
		{
			name:     "legacy duplicates preserved",
			input:    "[a] [b] [a] [c] [b] [a]",
			expected: []string{"a", "b", "a", "c", "b", "a"},
		},
		{
			name:     "legacy whitespace in brackets",
			input:    "This has [  link with spaces  ] and [another]",
			expected: []string{"  link with spaces  ", "another"},
		},

		// Markdown link exclusion
		{
			name:     "markdown link not matched",
			input:    "This has [text](url)",
			expected: []string{},
		},
		{
			name:     "markdown link with card-like text not matched",
			input:    "Click [42](https://example.com)",
			expected: []string{},
		},

		// Mixed syntax
		{
			name:     "wiki and legacy in same body",
			input:    "Old [legacy_id] and new [[wiki_id]] links",
			expected: []string{"wiki_id", "legacy_id"},
		},
		{
			name:     "wiki-link and markdown link both present",
			input:    "See [[42]] and [click here](https://example.com)",
			expected: []string{"42"},
		},
		{
			name:     "all three: wiki, legacy, markdown",
			input:    "[[wiki]] [legacy] and [md](url)",
			expected: []string{"wiki", "legacy"},
		},

		// Edge cases
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "no links at all",
			input:    "Just plain text with no links",
			expected: []string{},
		},
		{
			name:     "unclosed bracket",
			input:    "This has [unclosed",
			expected: []string{},
		},
		{
			name:     "unopened bracket",
			input:    "This has unclosed]",
			expected: []string{},
		},
		{
			name:     "empty brackets",
			input:    "This has []",
			expected: []string{},
		},
		{
			name:     "wiki-link not double-counted as legacy",
			input:    "[[42]] should only produce one result",
			expected: []string{"42"},
		},
		{
			name:     "multiple wiki-links with display text",
			input:    "[[a|First]] and [[b|Second]] and [[c]]",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "legacy link not confused by adjacent brackets",
			input:    "Text with [link] and more",
			expected: []string{"link"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBacklinks(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractBacklinks() = %v, expected %v", result, tt.expected)
				return
			}
			for i, r := range result {
				if i >= len(tt.expected) {
					break
				}
				if r != tt.expected[i] {
					t.Errorf("ExtractBacklinks()[%d] = %q, want %q", i, r, tt.expected[i])
				}
			}
		})
	}
}

func TestIsMarkdownLink(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		match    string
		expected bool
	}{
		{
			name:     "followed by parenthesis",
			text:     "[text](url)",
			match:    "[text]",
			expected: true,
		},
		{
			name:     "not followed by parenthesis",
			text:     "[text] something",
			match:    "[text]",
			expected: false,
		},
		{
			name:     "match at end of text",
			text:     "see [text]",
			match:    "[text]",
			expected: false,
		},
		{
			name:     "match not in text",
			text:     "no brackets here",
			match:    "[text]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMarkdownLink(tt.text, tt.match)
			if result != tt.expected {
				t.Errorf("IsMarkdownLink() = %v, want %v", result, tt.expected)
			}
		})
	}
}
