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
