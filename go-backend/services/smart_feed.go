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

// getPriorityFeeds returns a map of feed IDs that have priority=true
func getPriorityFeeds(db models.Database, userID int) (map[int]bool, error) {
	query := `SELECT id FROM rss_feeds WHERE user_id = $1 AND priority = TRUE`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get priority feeds: %w", err)
	}
	defer rows.Close()

	priorityFeeds := make(map[int]bool)
	for rows.Next() {
		var feedID int
		if err := rows.Scan(&feedID); err != nil {
			return nil, fmt.Errorf("failed to scan priority feed: %w", err)
		}
		priorityFeeds[feedID] = true
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating priority feeds: %w", err)
	}

	return priorityFeeds, nil
}

// generateScoreReason creates a human-readable explanation for the score
func generateScoreReason(volumeScore, interactionBonus float64, isPriority bool, dailyAvg float64) string {
	reasons := []string{}

	if isPriority {
		reasons = append(reasons, "Priority feed")
	}

	if volumeScore >= 80 {
		reasons = append(reasons, fmt.Sprintf("Low-volume feed (~%.1f article/day)", dailyAvg))
	} else if volumeScore >= 50 {
		reasons = append(reasons, fmt.Sprintf("Medium-volume feed (~%.1f articles/day)", dailyAvg))
	} else if volumeScore > 0 {
		reasons = append(reasons, fmt.Sprintf("High-volume feed (~%.1f articles/day)", dailyAvg))
	}

	if interactionBonus > 0 {
		reasons = append(reasons, fmt.Sprintf("You convert %.0f%% of articles", interactionBonus/5))
	}

	if len(reasons) == 0 {
		return "New feed"
	}

	return reasons[0]
}
