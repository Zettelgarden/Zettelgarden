package services

import (
	"fmt"
	"go-backend/models"
	"log"
	"time"
)

// GetDailyStats retrieves aggregated activity statistics for a date range.
// Returns stats for all days in the range, with 0 counts for days with no
// activity.
//
// Originally this used a Postgres generate_series() CTE plus INTERVAL / AT
// TIME ZONE. SQLite has none of those, so to stay driver-neutral the day
// series is built in Go and activity is grouped by the UTC date prefix
// (substr of the ISO timestamp). Day boundaries are therefore UTC (the app
// stores UTC per D5); the `timezone` arg is accepted for API compatibility but
// not used for grouping. Timezone-aware grouping can be added app-side later
// if a non-UTC user needs it.
func GetDailyStats(db models.Database, userID int, startDate, endDate time.Time, timezone string) ([]models.DailyStats, error) {
	_ = timezone // UTC grouping; see note above.

	// Build the full inclusive day series, oldest first, keyed by 'YYYY-MM-DD'.
	byDay := make(map[string]*models.DailyStats)
	var ordered []string
	endExclusive := endDate.UTC().AddDate(0, 0, 1)
	for d := startDate.UTC(); !d.After(endDate.UTC()); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if _, ok := byDay[key]; !ok {
			byDay[key] = &models.DailyStats{Date: d}
			ordered = append(ordered, key)
		}
	}

	// cards created per day
	cardRows, err := db.Query(`
		SELECT substr(cast(created_at as text), 1, 10) AS day, COUNT(*) AS n
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE
			AND created_at >= $2 AND created_at < $3
		GROUP BY day`, userID, startDate, endExclusive)
	if err != nil {
		log.Printf("Error querying daily card stats: %v", err)
		return []models.DailyStats{}, fmt.Errorf("unable to fetch daily stats")
	}
	for cardRows.Next() {
		var day string
		var n int
		if err := cardRows.Scan(&day, &n); err != nil {
			log.Printf("Error scanning daily card stats row: %v", err)
			continue
		}
		if s, ok := byDay[day]; ok {
			s.CardsCreated = n
		}
	}
	cardRows.Close()

	// tasks created per day
	taskCreatedRows, err := db.Query(`
		SELECT substr(cast(created_at as text), 1, 10) AS day, COUNT(*) AS n
		FROM tasks
		WHERE user_id = $1 AND is_deleted = FALSE
			AND created_at >= $2 AND created_at < $3
		GROUP BY day`, userID, startDate, endExclusive)
	if err != nil {
		log.Printf("Error querying daily task-created stats: %v", err)
		return []models.DailyStats{}, fmt.Errorf("unable to fetch daily stats")
	}
	for taskCreatedRows.Next() {
		var day string
		var n int
		if err := taskCreatedRows.Scan(&day, &n); err != nil {
			log.Printf("Error scanning daily task-created stats row: %v", err)
			continue
		}
		if s, ok := byDay[day]; ok {
			s.TasksCreated = n
		}
	}
	taskCreatedRows.Close()

	// tasks completed per day
	taskCompletedRows, err := db.Query(`
		SELECT substr(cast(completed_at as text), 1, 10) AS day, COUNT(*) AS n
		FROM tasks
		WHERE user_id = $1 AND is_deleted = FALSE
			AND completed_at IS NOT NULL
			AND completed_at >= $2 AND completed_at < $3
		GROUP BY day`, userID, startDate, endExclusive)
	if err != nil {
		log.Printf("Error querying daily task-completed stats: %v", err)
		return []models.DailyStats{}, fmt.Errorf("unable to fetch daily stats")
	}
	for taskCompletedRows.Next() {
		var day string
		var n int
		if err := taskCompletedRows.Scan(&day, &n); err != nil {
			log.Printf("Error scanning daily task-completed stats row: %v", err)
			continue
		}
		if s, ok := byDay[day]; ok {
			s.TasksCompleted = n
		}
	}
	taskCompletedRows.Close()

	stats := make([]models.DailyStats, 0, len(ordered))
	for _, day := range ordered {
		stats = append(stats, *byDay[day])
	}
	return stats, nil
}

// GetTasksCompletedOnDate retrieves all tasks completed on a specific date.
// The target date is compared as a 'YYYY-MM-DD' string against the UTC date
// prefix of completed_at (cross-driver; see GetDailyStats note).
func GetTasksCompletedOnDate(db models.Database, userID int, date time.Time, timezone string) ([]models.Task, error) {
	_ = timezone
	day := date.UTC().Format("2006-01-02")
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
		created_at, updated_at, completed_at, title, priority, is_complete
	FROM tasks
	WHERE user_id = $1
		AND is_deleted = FALSE
		AND substr(cast(completed_at as text), 1, 10) = $2
	ORDER BY completed_at DESC
	`

	rows, err := db.Query(query, userID, day)
	if err != nil {
		log.Printf("Error querying tasks for date: %v", err)
		return []models.Task{}, fmt.Errorf("unable to fetch tasks for date")
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID,
			&task.CardPK,
			&task.UserID,
			&task.ScheduledDate,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.CompletedAt,
			&task.Title,
			&task.Priority,
			&task.IsComplete,
		); err != nil {
			log.Printf("Error scanning task row: %v", err)
			continue
		}

		// Load associated card if exists (following pattern from services/tasks.go)
		if task.CardPK > 0 {
			card, err := GetPartialCard(db, userID, task.CardPK)
			if err == nil {
				task.Card = card
			}
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating task rows: %v", err)
		return []models.Task{}, fmt.Errorf("error reading tasks data")
	}

	return tasks, nil
}

// GetCardsCreatedOnDate retrieves all cards created on a specific date.
// The target date is compared as a 'YYYY-MM-DD' string against the UTC date
// prefix of created_at (cross-driver; see GetDailyStats note).
func GetCardsCreatedOnDate(db models.Database, userID int, date time.Time, timezone string) ([]models.PartialCard, error) {
	_ = timezone
	day := date.UTC().Format("2006-01-02")
	query := `
	SELECT id, card_id, title, created_at, updated_at, parent_id, user_id
	FROM cards
	WHERE user_id = $1
		AND is_deleted = FALSE
		AND substr(cast(created_at as text), 1, 10) = $2
	ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID, day)
	if err != nil {
		log.Printf("Error querying cards for date: %v", err)
		return []models.PartialCard{}, fmt.Errorf("unable to fetch cards for date")
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var card models.PartialCard
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.Title,
			&card.CreatedAt,
			&card.UpdatedAt,
			&card.ParentID,
			&card.UserID,
		); err != nil {
			log.Printf("Error scanning card row: %v", err)
			continue
		}

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating card rows: %v", err)
		return []models.PartialCard{}, fmt.Errorf("error reading cards data")
	}

	return cards, nil
}
