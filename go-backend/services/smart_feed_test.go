package services

import (
	"testing"

	"go-backend/models"
	"github.com/stretchr/testify/assert"
)

func TestCalculateFeedVolumeScores(t *testing.T) {
	// This requires a test database connection
	// For now, test the scoring logic directly
	t.Run("zero articles gets max score", func(t *testing.T) {
		score := calculateVolumeScore(0)
		assert.Equal(t, 100.0, score)
	})

	t.Run("1 article per day gets high score", func(t *testing.T) {
		score := calculateVolumeScore(30) // 30 articles in 30 days = 1/day
		assert.Equal(t, 90.0, score)
	})

	t.Run("5 articles per day gets medium score", func(t *testing.T) {
		score := calculateVolumeScore(150) // 150 articles in 30 days = 5/day
		assert.Equal(t, 50.0, score)
	})

	t.Run("10+ articles per day gets zero score", func(t *testing.T) {
		score := calculateVolumeScore(300) // 300 articles in 30 days = 10/day
		assert.Equal(t, 0.0, score)
	})

	t.Run("score floors at zero", func(t *testing.T) {
		score := calculateVolumeScore(600) // 20 articles per day
		assert.Equal(t, 0.0, score)
	})
}

func TestCalculateInteractionBonus(t *testing.T) {
	t.Run("zero conversions gets zero bonus", func(t *testing.T) {
		bonus := calculateInteractionBonus(0)
		assert.Equal(t, 0.0, bonus)
	})

	t.Run("1 conversion gets 10 points", func(t *testing.T) {
		bonus := calculateInteractionBonus(1)
		assert.Equal(t, 10.0, bonus)
	})

	t.Run("5 conversions gets 50 points (max)", func(t *testing.T) {
		bonus := calculateInteractionBonus(5)
		assert.Equal(t, 50.0, bonus)
	})

	t.Run("10+ conversions caps at 50 points", func(t *testing.T) {
		bonus := calculateInteractionBonus(10)
		assert.Equal(t, 50.0, bonus)
	})
}

func TestGetPriorityFeeds(t *testing.T) {
	// This would require DB setup, so we'll test the function structure
	// For now, just ensure it compiles
	t.Run("function exists", func(t *testing.T) {
		// This is a compile-time check
		var _ func(models.Database, int) (map[int]bool, error) = getPriorityFeeds
	})
}

func TestGenerateScoreReason(t *testing.T) {
	t.Run("priority feed gets priority reason", func(t *testing.T) {
		reason := generateScoreReason(50.0, 0.0, true, 0.0)
		assert.Equal(t, "Priority feed", reason)
	})

	t.Run("low volume feed gets volume reason", func(t *testing.T) {
		reason := generateScoreReason(90.0, 0.0, false, 1.0)
		assert.Equal(t, "Low-volume feed (~1.0 article/day)", reason)
	})

	t.Run("medium volume feed gets volume reason", func(t *testing.T) {
		reason := generateScoreReason(60.0, 0.0, false, 4.0)
		assert.Equal(t, "Medium-volume feed (~4.0 articles/day)", reason)
	})

	t.Run("high volume feed gets volume reason", func(t *testing.T) {
		reason := generateScoreReason(30.0, 0.0, false, 7.0)
		assert.Equal(t, "High-volume feed (~7.0 articles/day)", reason)
	})

	t.Run("interaction bonus gets conversion reason", func(t *testing.T) {
		// Volume score of 0 means no volume data, so interaction reason comes first
		reason := generateScoreReason(0.0, 20.0, false, 0.0)
		assert.Equal(t, "You convert 4% of articles", reason)
	})

	t.Run("volume score takes precedence over interaction", func(t *testing.T) {
		// When there's volume data, it comes before interaction
		reason := generateScoreReason(50.0, 20.0, false, 5.0)
		assert.Equal(t, "Medium-volume feed (~5.0 articles/day)", reason)
	})

	t.Run("truly new feed with no data", func(t *testing.T) {
		// 0 volume score means no articles, 0 interaction means no conversions
		// But volume score of 0 doesn't match any volume conditions
		// So we get the interaction reason (which is > 0 check fails)
		reason := generateScoreReason(0.0, 0.0, false, 0.0)
		assert.Equal(t, "New feed", reason)
	})
}

func TestListSmartRSSArticles(t *testing.T) {
	// This would require DB setup, so we'll test the function structure
	// For now, just ensure it compiles
	t.Run("function exists", func(t *testing.T) {
		// This is a compile-time check
		var _ func(models.Database, int, map[string]interface{}) ([]models.RSSArticleWithScore, int, error) = ListSmartRSSArticles
	})
}
