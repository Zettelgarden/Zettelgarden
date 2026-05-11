package template

import (
	"regexp"
	"strconv"
	"testing"
)

// TestNextChildIDRegex tests the regex used to detect the PREFIX.SEP.NUMBER pattern
// in existing child card_ids. This validates the core logic used when children
// don't use the parent's card_id as a prefix.
func TestNextChildIDRegex(t *testing.T) {
	re := regexp.MustCompile(`^(.+?)([./\-])(\d+)$`)

	tests := []struct {
		input     string
		wantMatch bool
		prefix    string
		sep       string
		num       int
	}{
		// Standard dot-separated IDs
		{"2483.82", true, "2483", ".", 82},
		{"2483.1", true, "2483", ".", 1},
		{"999.42", true, "999", ".", 42},

		// Slash-separated IDs
		{"SP24/B", false, "", "", 0}, // no trailing number
		{"2483/5", true, "2483", "/", 5},

		// Hyphen-separated IDs
		{"ABC-7", true, "ABC", "-", 7},

		// IDs that should NOT match
		{"2483", false, "", "", 0},       // no separator
		{"", false, "", "", 0},           // empty
		{"ABC.", false, "", "", 0},        // no trailing number
		{"ABC.12X", false, "", "", 0},     // non-numeric suffix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match := re.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if len(match) != 4 {
					t.Fatalf("expected match, got no match for %q", tt.input)
				}
				if match[1] != tt.prefix {
					t.Errorf("prefix: got %q, want %q", match[1], tt.prefix)
				}
				if match[2] != tt.sep {
					t.Errorf("sep: got %q, want %q", match[2], tt.sep)
				}
				num, _ := strconv.Atoi(match[3])
				if num != tt.num {
					t.Errorf("num: got %d, want %d", num, tt.num)
				}
			} else {
				if len(match) == 4 {
					t.Errorf("expected no match, but got prefix=%q sep=%q num=%q", match[1], match[2], match[3])
				}
			}
		})
	}
}

// TestNextChildIDSuffixRegex tests both numeric and alphabetic suffix matching.
func TestNextChildIDSuffixRegex(t *testing.T) {
	numRe := regexp.MustCompile(`^[.\\/-]+(\d+)$`)
	alphaRe := regexp.MustCompile(`^[.\\/-]+([A-Z])$`)

	t.Run("numeric suffixes", func(t *testing.T) {
		cases := []struct {
			suffix string
			match  bool
			num    string
		}{
			{".1", true, "1"},
			{".82", true, "82"},
			{"/5", true, "5"},
			{"-3", true, "3"},
			{".10", true, "10"},
			{"/A", false, ""}, // alphabetic should not match numeric regex
			{"", false, ""},
		}
		for _, tt := range cases {
			t.Run(tt.suffix, func(t *testing.T) {
				match := numRe.FindStringSubmatch(tt.suffix)
				if tt.match {
					if len(match) != 2 {
						t.Fatalf("expected match for %q, got none", tt.suffix)
					}
					if match[1] != tt.num {
						t.Errorf("got %q, want %q", match[1], tt.num)
					}
				} else if len(match) == 2 {
					t.Errorf("expected no match for %q", tt.suffix)
				}
			})
		}
	})

	t.Run("alphabetic suffixes", func(t *testing.T) {
		cases := []struct {
			suffix string
			match  bool
			letter string
		}{
			{"/A", true, "A"},
			{"/B", true, "B"},
			{"/Z", true, "Z"},
			{".A", true, "A"},
			{"-C", true, "C"},
			{"/a", false, ""}, // lowercase should not match
			{"/AA", false, ""}, // two letters should not match
			{"/1", false, ""}, // numeric should not match alpha regex
			{"", false, ""},
		}
		for _, tt := range cases {
			t.Run(tt.suffix, func(t *testing.T) {
				match := alphaRe.FindStringSubmatch(tt.suffix)
				if tt.match {
					if len(match) != 2 {
						t.Fatalf("expected match for %q, got none", tt.suffix)
					}
					if match[1] != tt.letter {
						t.Errorf("got %q, want %q", match[1], tt.letter)
					}
				} else if len(match) == 2 {
					t.Errorf("expected no match for %q", tt.suffix)
				}
			})
		}
	})
}

// TestNextChildIDPrefixMatching validates the parent-prefix-matching regex
// used in the first pass (when children use parent's card_id as prefix).
func TestNextChildIDPrefixMatching(t *testing.T) {
	re := regexp.MustCompile(`^[.\\/-]+(\d+)`)

	tests := []struct {
		suffix string
		match  bool
		num    int
	}{
		{".1", true, 1},
		{".82", true, 82},
		{"/5", true, 5},
		{"-3", true, 3},
		{".10", true, 10},
	}

	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			match := re.FindStringSubmatch(tt.suffix)
			if tt.match {
				if len(match) != 2 {
					t.Fatalf("expected match for %q, got none", tt.suffix)
				}
				num, _ := strconv.Atoi(match[1])
				if num != tt.num {
					t.Errorf("got %d, want %d", num, tt.num)
				}
			} else if len(match) == 2 {
				t.Errorf("expected no match for %q", tt.suffix)
			}
		})
	}
}
