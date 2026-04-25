package models

import (
	"testing"
)

func TestNotificationImportanceScore(t *testing.T) {
	tests := []struct {
		name           string
		sourceType     string
		isStarred      bool
		isPriorityFeed bool
		expectedScore  int
	}{
		{
			name:          "starred article",
			sourceType:    "rss",
			isStarred:     true,
			expectedScore: 10,
		},
		{
			name:           "priority feed article",
			sourceType:     "rss",
			isPriorityFeed: true,
			expectedScore:  5,
		},
		{
			name:           "priority task",
			sourceType:     "task",
			isPriorityFeed: true,
			expectedScore:  5,
		},
		{
			name:          "normal article",
			sourceType:    "rss",
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateImportanceScore(tt.sourceType, tt.isStarred, tt.isPriorityFeed)
			if score != tt.expectedScore {
				t.Errorf("CalculateImportanceScore() = %d, want %d", score, tt.expectedScore)
			}
		})
	}
}
