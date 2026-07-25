package services

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"go-backend/models"
)

const (
	// DefaultLimit is the default number of articles to return
	DefaultLimit = 100
	// MaxLimit is the maximum number of articles allowed per request
	MaxLimit = 1000
	// VolumeDays is the number of days to look back for volume scoring
	VolumeDays = 30
	// InteractionDays is the number of days to look back for interaction scoring
	InteractionDays = 90
	// SmartFeedMaxAgeDays is the maximum age of articles to include in smart feed
	SmartFeedMaxAgeDays = 14
	// SmartFeedDecayDays is the decay constant for age-based scoring
	SmartFeedDecayDays = 7
)

// calculateVolumeScore converts article count to volume score (0-100)
// score = max(0, 100 - (daily_avg x 10))
func calculateVolumeScore(articleCount int) float64 {
	if articleCount <= 0 {
		return 100.0
	}
	dailyAvg := float64(articleCount) / float64(VolumeDays)
	score := 100.0 - (dailyAvg * 10.0)
	if score < 0 {
		return 0.0
	}
	return score
}

// calculateFeedVolumeScores gets article counts for each feed in the last VolumeDays
func calculateFeedVolumeScores(db models.Database, userID int) (map[int]float64, error) {
	// App-side cutoff (SQLite has no INTERVAL). See migration design P3.
	volumeCutoff := time.Now().UTC().AddDate(0, 0, -VolumeDays)
	query := `
		SELECT feed_id, COUNT(*) as article_count
		FROM rss_articles
		WHERE user_id = $1 AND published_at > $2
		GROUP BY feed_id
	`
	rows, err := db.Query(query, userID, volumeCutoff)
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

// calculateInteractionBonuses gets conversion counts for each feed in last InteractionDays
func calculateInteractionBonuses(db models.Database, userID int) (map[int]float64, error) {
	query := `
        SELECT f.id, COUNT(a.card_id) as conversion_count
        FROM rss_feeds f
        JOIN rss_articles a ON f.id = a.feed_id
        WHERE f.user_id = $1 AND a.card_id IS NOT NULL
          AND a.published_at > $2
        GROUP BY f.id
    `
	interactionCutoff := time.Now().UTC().AddDate(0, 0, -InteractionDays)
	rows, err := db.Query(query, userID, interactionCutoff)
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

// ListSmartRSSArticles returns articles ranked by smart scoring
func ListSmartRSSArticles(db models.Database, userID int, filters map[string]interface{}) ([]models.RSSArticleWithScore, int, error) {
	// 1. Calculate volume scores (last 30 days)
	volumeScores, err := calculateFeedVolumeScores(db, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to calculate volume scores: %w", err)
	}

	// 2. Calculate interaction bonuses (last 90 days)
	interactionBonuses, err := calculateInteractionBonuses(db, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to calculate interaction bonuses: %w", err)
	}

	// 3. Get priority feeds
	priorityFeeds, err := getPriorityFeeds(db, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get priority feeds: %w", err)
	}

	// 4. Query articles with scoring and sort
	articles, total, err := queryArticlesWithScoring(db, userID, filters, volumeScores, interactionBonuses, priorityFeeds)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}

	return articles, total, nil
}

// queryArticlesWithScoring fetches articles and calculates scores
func queryArticlesWithScoring(db models.Database, userID int, filters map[string]interface{}, volumeScores map[int]float64, interactionBonuses map[int]float64, priorityFeeds map[int]bool) ([]models.RSSArticleWithScore, int, error) {
	// Build WHERE clause separately for reuse in both count and main queries
	whereClause := "user_id = $1"
	args := []interface{}{userID}
	argPos := 2

	// Apply filters (same as ListRSSArticles)
	if folder, ok := filters["folder"].(string); ok && folder != "" {
		whereClause += fmt.Sprintf(" AND feed_id IN (SELECT id FROM rss_feeds WHERE user_id = $1 AND folder = $%d)", argPos)
		args = append(args, folder)
		argPos++
	}

	if feedID, ok := filters["feed_id"].(int); ok && feedID > 0 {
		whereClause += fmt.Sprintf(" AND feed_id = $%d", argPos)
		args = append(args, feedID)
		argPos++
	}

	if unreadOnly, ok := filters["unread"].(bool); ok && unreadOnly {
		whereClause += " AND read = false"
	}

	// Apply 14-day age cutoff for smart feed. App-side cutoff (SQLite has no
	// INTERVAL). See migration design P3.
	whereClause += fmt.Sprintf(" AND published_at > $%d", argPos)
	args = append(args, time.Now().UTC().AddDate(0, 0, -SmartFeedMaxAgeDays))
	argPos++

	// Get count first
	countQuery := "SELECT COUNT(*) FROM rss_articles WHERE " + whereClause
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %w", err)
	}

	// Build main query
	query := `
		SELECT id, user_id, feed_id, title, content, author, url,
		       published_at, fetched_at, read, card_id
		FROM rss_articles
		WHERE ` + whereClause + ` ORDER BY published_at DESC NULLS LAST, fetched_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	// Collect articles with scores
	var scoredArticles []models.RSSArticleWithScore
	for rows.Next() {
		var article models.RSSArticleWithScore
		var content, author sql.NullString
		var publishedAt sql.NullTime
		var cardID sql.NullInt64

		err := rows.Scan(
			&article.ID, &article.UserID, &article.FeedID, &article.Title,
			&content, &author, &article.URL, &publishedAt,
			&article.FetchedAt, &article.Read, &cardID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan article: %w", err)
		}

		if content.Valid {
			article.Content = &content.String
		}
		if author.Valid {
			article.Author = &author.String
		}
		if publishedAt.Valid {
			article.PublishedAt = &publishedAt.Time
		}
		if cardID.Valid {
			cardIDInt := int(cardID.Int64)
			article.CardID = &cardIDInt
		}

		// Calculate scores
		volumeScore := volumeScores[article.FeedID]
		interactionBonus := interactionBonuses[article.FeedID]
		isPriority := priorityFeeds[article.FeedID]

		priorityBonus := 0.0
		if isPriority {
			priorityBonus = 100.0
		}

		// Calculate age decay - newer articles get higher scores
		// Decay ranges from 1.0 (new) to ~0 (old), maxes at 1.0 for future dates
		ageDecay := 1.0
		if article.PublishedAt != nil {
			articleAgeDays := time.Since(*article.PublishedAt).Hours() / 24
			calculatedDecay := math.Exp(-articleAgeDays / SmartFeedDecayDays)
			// Clamp to max 1.0 - future-dated articles shouldn't get boosted beyond base score
			if calculatedDecay < ageDecay {
				ageDecay = calculatedDecay
			}
		}

		totalScore := (volumeScore + interactionBonus + priorityBonus) * ageDecay

		// Calculate daily average for reason
		dailyAvg := 0.0
		if volumeScore < 100 {
			// Reverse engineer from score
			dailyAvg = (100.0 - volumeScore) / 10.0
		}

		article.SmartScore = &models.SmartFeedScore{
			ArticleID:        article.ID,
			Score:            totalScore,
			VolumeScore:      volumeScore,
			InteractionBonus: interactionBonus,
			IsPriority:       isPriority,
			Reason:           generateScoreReason(volumeScore, interactionBonus, isPriority, dailyAvg),
		}

		scoredArticles = append(scoredArticles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating articles: %w", err)
	}

	// Sort by score DESC, then published_at DESC
	sort.Slice(scoredArticles, func(i, j int) bool {
		si := scoredArticles[i].SmartScore
		sj := scoredArticles[j].SmartScore
		if si.Score != sj.Score {
			return si.Score > sj.Score
		}
		// Tie-break by published date
		pi, pj := scoredArticles[i].PublishedAt, scoredArticles[j].PublishedAt
		if pi == nil {
			return false
		}
		if pj == nil {
			return true
		}
		return pi.After(*pj)
	})

	// Apply limit/offset after sorting
	limit := DefaultLimit
	if limitParam, ok := filters["limit"].(int); ok {
		if limitParam > 0 && limitParam <= MaxLimit {
			limit = limitParam
		} else if limitParam > MaxLimit {
			limit = MaxLimit
		}
	}
	offset := 0
	if offsetParam, ok := filters["offset"].(int); ok && offsetParam > 0 {
		offset = offsetParam
	}

	if offset >= len(scoredArticles) {
		return []models.RSSArticleWithScore{}, total, nil
	}

	end := offset + limit
	if end > len(scoredArticles) {
		end = len(scoredArticles)
	}

	return scoredArticles[offset:end], total, nil
}
