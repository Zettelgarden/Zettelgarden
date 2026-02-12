package services

import (
	"fmt"

	"go-backend/models"
)

// calculateVolumeScore converts article count to volume score (0-100)
// score = max(0, 100 - (daily_avg x 10))
func calculateVolumeScore(articleCount int) float64 {
	if articleCount <= 0 {
		return 100.0
	}
	dailyAvg := float64(articleCount) / 30.0
	score := 100.0 - (dailyAvg * 10.0)
	if score < 0 {
		return 0.0
	}
	return score
}

// calculateFeedVolumeScores gets article counts for each feed in the last 30 days
func calculateFeedVolumeScores(db models.Database, userID int) (map[int]float64, error) {
	query := `
		SELECT feed_id, COUNT(*) as article_count
		FROM rss_articles
		WHERE user_id = $1 AND published_at > NOW() - INTERVAL '30 days'
		GROUP BY feed_id
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed volume scores: %w", err)
	}
	defer rows.Close()

	scores := make(map[int]float64)
	for rows.Next() {
		var feedID, count int
		if err := rows.Scan(&feedID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan volume score: %w", err)
		}
		scores[feedID] = calculateVolumeScore(count)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating volume scores: %w", err)
	}

	return scores, nil
}
