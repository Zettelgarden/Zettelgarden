package handlers

import (
	"go-backend/tests"
	"strings"
	"testing"
)

func TestParseSearchText(t *testing.T) {
	var output SearchParams
	input := "hello #test"
	output = ParseSearchText(input)

	if len(output.Terms) != 1 {
		t.Errorf("wrong number of search terms returned, got %v want %v", len(output.Terms), 0)
	}
	if output.Terms[0] != "hello" {
		t.Errorf("wrong term returned, got %v want %v", output.Terms[0], "hello")
	}
	if len(output.Tags) != 1 {
		t.Errorf("wrong number of search tags returned, got %v want %v", len(output.Tags), 0)
	}
	if output.Tags[0] != "test" {
		t.Errorf("wrong tag returned, got %v want %v", output.Tags[0], "test")
	}

	input = "hello #test world #another"
	output = ParseSearchText(input)

	if len(output.Terms) != 2 {
		t.Errorf("wrong number of search terms returned, got %v want %v", len(output.Terms), 0)
	}
	if output.Terms[0] != "hello" {
		t.Errorf("wrong term returned, got %v want %v", output.Terms[0], "hello")
	}
	if output.Terms[1] != "world" {
		t.Errorf("wrong term returned, got %v want %v", output.Terms[1], "world")
	}
	if len(output.Tags) != 2 {
		t.Errorf("wrong number of search tags returned, got %v want %v", len(output.Tags), 0)
	}
	if output.Tags[0] != "test" {
		t.Errorf("wrong tag returned, got %v want %v", output.Tags[0], "test")
	}
	if output.Tags[1] != "another" {
		t.Errorf("wrong tag returned, got %v want %v", output.Tags[1], "another")
	}
}

func TestBuildPartialCardSqlSearchTermString(t *testing.T) {
	input := "hello world"
	expectedOutput := " AND ((card_id ILIKE '%hello%' OR title ILIKE '%hello%') AND (card_id ILIKE '%world%' OR title ILIKE '%world%'))"
	output := BuildPartialCardSqlSearchTermString(input, false)
	if output != expectedOutput {
		t.Errorf("wrong string returned, got %v want %v", output, expectedOutput)
	}

	input = "hello #world"
	expectedOutput = `
	AND (EXISTS (
	            SELECT 1 FROM card_tags
	            JOIN tags ON card_tags.tag_id = tags.id
	            WHERE card_tags.card_pk = c.id AND tags.name = 'world' AND tags.is_deleted = FALSE
	        )) AND ((card_id ILIKE '%hello%' OR title ILIKE '%hello%'))
			`
	expectedOutput = strings.ReplaceAll(expectedOutput, " ", "")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\n", "")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\t", "")
	output = BuildPartialCardSqlSearchTermString(input, false)
	output = strings.ReplaceAll(output, " ", "")
	output = strings.ReplaceAll(output, "\n", "")
	if output != expectedOutput {
		t.Errorf("wrong string returned, got %v want %v", output, expectedOutput)
	}

}

func TestBuildPartialCardSqlSearchTermStringNegate(t *testing.T) {

	input := "hello !world"
	expectedOutput := " AND ((card_id ILIKE '%hello%' OR title ILIKE '%hello%')) AND (NOT (card_id ILIKE '%world%' OR title ILIKE '%world%'))"
	output := BuildPartialCardSqlSearchTermString(input, false)
	if output != expectedOutput {
		t.Errorf("wrong string returned, got %v want %v", output, expectedOutput)
	}

	input = "hello !#world"
	expectedOutput = `
AND ((card_id ILIKE '%hello%' OR title ILIKE '%hello%')) AND (NOT EXISTS (
            SELECT 1 FROM card_tags
            JOIN tags ON card_tags.tag_id = tags.id
            WHERE card_tags.card_pk = c.id AND tags.name = 'world' AND tags.is_deleted = FALSE
        ))
`
	expectedOutput = strings.ReplaceAll(expectedOutput, " ", "")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\n", "")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\t", "")
	output = BuildPartialCardSqlSearchTermString(input, false)
	output = strings.ReplaceAll(output, " ", "")
	output = strings.ReplaceAll(output, "\n", "")
	if output != expectedOutput {
		t.Errorf("wrong string returned, got %v want %v", output, expectedOutput)
	}
}

func TestBuildPartialCardSqlSearchTermStringWithEntities(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		fullText bool
		contains []string // Strings that should be present in the output
	}{
		{
			name:     "single entity",
			input:    "@[John Smith]",
			fullText: false,
			contains: []string{
				"EXISTS(SELECT1FROMentity_card_junctionecj",
				"JOINentitieseONecj.entity_id=e.id",
				"WHEREecj.card_pk=c.idANDe.name='JohnSmith'",
			},
		},
		{
			name:     "negated entity",
			input:    "!@[Project Alpha]",
			fullText: false,
			contains: []string{
				"NOTEXISTS(SELECT1FROMentity_card_junctionecj",
				"JOINentitieseONecj.entity_id=e.id",
				"WHEREecj.card_pk=c.idANDe.name='ProjectAlpha'",
			},
		},
		{
			name:     "mixed entities and terms",
			input:    "hello @[John Smith] !@[Project Beta]",
			fullText: false,
			contains: []string{
				"card_idILIKE'%hello%'ORtitleILIKE'%hello%'",
				"EXISTS(SELECT1FROMentity_card_junctionecj",
				"WHEREecj.card_pk=c.idANDe.name='JohnSmith'",
				"NOTEXISTS(SELECT1FROMentity_card_junctionecj",
				"WHEREecj.card_pk=c.idANDe.name='ProjectBeta'",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildPartialCardSqlSearchTermString(tc.input, tc.fullText)

			// Normalize the result by removing all whitespace
			normalizedResult := strings.ReplaceAll(result, " ", "")
			normalizedResult = strings.ReplaceAll(normalizedResult, "\n", "")
			normalizedResult = strings.ReplaceAll(normalizedResult, "\t", "")
			t.Logf("Normalized SQL: %s", normalizedResult)

			// Check that all expected strings are present in the result
			for _, str := range tc.contains {
				if !strings.Contains(normalizedResult, str) {
					t.Errorf("Expected SQL to contain '%s', but it didn't.\nGot: %s", str, result)
				}
			}
		})
	}
}

// Add an integration test that actually executes the query
func TestClassicCardSearch(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Create test data including entities
	// ... setup code ...

	testCases := []struct {
		name       string
		searchTerm string
		wantCount  int
	}{
		{
			name:       "entity search",
			searchTerm: "@[Test Entity 1]",
			wantCount:  2,
		},
		{
			name:       "tag search",
			searchTerm: "#test",
			wantCount:  1,
		},
		// ... more test cases ...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cards, err := s.ClassicCardSearch(1, SearchRequestParams{SearchTerm: tc.searchTerm})
			if err != nil {
				t.Errorf("ClassicCardSearch() error = %v", err)
				return
			}
			if len(cards) != tc.wantCount {
				t.Errorf("ClassicCardSearch() got %v cards, want %v", len(cards), tc.wantCount)
			}
		})
	}
}

func TestFullTextSearch(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	testCases := []struct {
		name       string
		params     SearchRequestParams
		wantCount  int
		wantCardID string // Optional: to check specific card is found
	}{
		{
			name: "non-full text search - should not find body content",
			params: SearchRequestParams{
				SearchTerm: "uniquebodycontent",
				FullText:   false,
			},
			wantCount: 0,
		},
		{
			name: "full text search - should find body content",
			params: SearchRequestParams{
				SearchTerm: "uniquebodycontent",
				FullText:   true,
			},
			wantCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cards, err := s.ClassicCardSearch(1, tc.params)
			if err != nil {
				t.Errorf("ClassicCardSearch() error = %v", err)
				return
			}
			if len(cards) != tc.wantCount {
				t.Errorf("ClassicCardSearch() got %v cards, want %v", len(cards), tc.wantCount)
			}
			// If we want to check for specific cards
			if tc.wantCardID != "" && len(cards) > 0 {
				found := false
				for _, card := range cards {
					if card.CardID == tc.wantCardID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ClassicCardSearch() did not find expected card with ID %s", tc.wantCardID)
				}
			}
		})
	}
}

func TestBuildPartialEntitySqlSearchTermString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single term",
			input:    "hello",
			expected: " AND ((name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%'))",
		},
		{
			name:     "multiple terms",
			input:    "hello world",
			expected: " AND ((name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%') OR (name ILIKE '%world%' OR description ILIKE '%world%' OR type ILIKE '%world%'))",
		},
		{
			name:     "negated term",
			input:    "hello !world",
			expected: " AND ((name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%')) AND (NOT (name ILIKE '%world%' OR description ILIKE '%world%' OR type ILIKE '%world%'))",
		},
		{
			name:     "multiple negated terms",
			input:    "!hello !world",
			expected: " AND (NOT (name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%')) AND (NOT (name ILIKE '%world%' OR description ILIKE '%world%' OR type ILIKE '%world%'))",
		},
		{
			name:     "single tag",
			input:    "#test",
			expected: " AND (EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'test' AND tags.is_deleted = FALSE))",
		},
		{
			name:     "negated tag",
			input:    "!#test",
			expected: " AND (NOT EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'test' AND tags.is_deleted = FALSE))",
		},
		{
			name:     "term with tag",
			input:    "hello #test",
			expected: " AND (EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'test' AND tags.is_deleted = FALSE)) AND ((name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%'))",
		},
		{
			name:     "multiple tags",
			input:    "#test #another",
			expected: " AND (EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'test' AND tags.is_deleted = FALSE)) AND (EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'another' AND tags.is_deleted = FALSE))",
		},
		{
			name:     "mixed terms, tags, and negations",
			input:    "hello #test !world !#another",
			expected: " AND (EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'test' AND tags.is_deleted = FALSE)) AND ((name ILIKE '%hello%' OR description ILIKE '%hello%' OR type ILIKE '%hello%')) AND (NOT (name ILIKE '%world%' OR description ILIKE '%world%' OR type ILIKE '%world%')) AND (NOT EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = 'another' AND tags.is_deleted = FALSE))",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := BuildPartialEntitySqlSearchTermString(tc.input)
			if output != tc.expected {
				t.Errorf("wrong string returned for %s\ngot:  %v\nwant: %v", tc.name, output, tc.expected)
			}
		})
	}
}
