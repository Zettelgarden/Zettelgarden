package services

import (
	"testing"

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
