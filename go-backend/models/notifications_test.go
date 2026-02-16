package models

import (
	"testing"
)

func TestNotificationImportanceScore(t *testing.T) {
	tests := []struct {
		name           string
		sourceType     string
		isUnprocessed  bool
		isStarred      bool
		isPriorityFeed bool
		expectedScore  int
	}{
		{
			name:           "unprocessed email",
			sourceType:     "email",
			isUnprocessed:  true,
			expectedScore:  10,
		},
		{
			name:           "triaged email",
			sourceType:     "email",
			isUnprocessed:  false,
			expectedScore:  5,
		},
		{
			name:           "starred article",
			sourceType:     "rss",
			isStarred:      true,
			expectedScore:  10,
		},
		{
			name:           "priority feed article",
			sourceType:     "rss",
			isPriorityFeed: true,
			expectedScore:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateImportanceScore(tt.sourceType, tt.isUnprocessed, tt.isStarred, tt.isPriorityFeed)
			if score != tt.expectedScore {
				t.Errorf("CalculateImportanceScore() = %d, want %d", score, tt.expectedScore)
			}
		})
	}
}

func TestFilterTagsForEmail(t *testing.T) {
	tags := GetFilterTagsForEmail("unprocessed", "test@example.com")
	if len(tags) == 0 {
		t.Error("Expected at least one filter tag")
	}

	// tags is now pq.StringArray which is a []string
	hasStatusTag := false
	for _, tag := range tags {
		if tag == "unprocessed" {
			hasStatusTag = true
		}
	}
	if !hasStatusTag {
		t.Error("Expected 'unprocessed' tag in filter tags")
	}
}
