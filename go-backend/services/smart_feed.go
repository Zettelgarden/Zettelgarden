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

// calculateInteractionBonus converts conversion count to bonus (0-50)
// bonus = min(50, conversion_count x 10)
func calculateInteractionBonus(conversionCount int) float64 {
	bonus := float64(conversionCount) * 10.0
	if bonus > 50.0 {
		return 50.0
	}
	return bonus
}

// calculateInteractionBonuses gets conversion counts for each feed in last 90 days
func calculateInteractionBonuses(db models.Database, userID int) (map[int]float64, error) {
	query := `
        SELECT f.id, COUNT(a.card_id) as conversion_count
        FROM rss_feeds f
        JOIN rss_articles a ON f.id = a.feed_id
        WHERE f.user_id = $1 AND a.card_id IS NOT NULL
          AND a.published_at > NOW() - INTERVAL '90 days'
        GROUP BY f.id
    `
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction bonuses: %w", err)
	}
	defer rows.Close()

	bonuses := make(map[int]float64)
	for rows.Next() {
		var feedID, count int
		if err := rows.Scan(&feedID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan interaction bonus: %w", err)
		}
		bonuses[feedID] = calculateInteractionBonus(count)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating interaction bonuses: %w", err)
	}

	return bonuses, nil
}
