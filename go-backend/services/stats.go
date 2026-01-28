package services

import (
	"fmt"
	"go-backend/models"
	"log"
	"time"
)

// GetDailyStats retrieves aggregated activity statistics for a date range
// Returns stats for all days in the range, with 0 counts for days with no activity
func GetDailyStats(db models.Database, userID int, startDate, endDate time.Time, timezone string) ([]models.DailyStats, error) {
	// Use CTE to generate date series and join with activity data
	query := `
	WITH date_series AS (
		SELECT generate_series(
			$2::date,
			$3::date,
			'1 day'::interval
		)::date AS day
	),
	cards_per_day AS (
		SELECT
			DATE(created_at AT TIME ZONE $4) AS day,
			COUNT(*) AS cards_created
		FROM cards
		WHERE user_id = $1
			AND is_deleted = FALSE
			AND created_at >= $2
			AND created_at < $3 + INTERVAL '1 day'
		GROUP BY DATE(created_at AT TIME ZONE $4)
	),
	tasks_created_per_day AS (
		SELECT
			DATE(created_at AT TIME ZONE $4) AS day,
			COUNT(*) AS tasks_created
		FROM tasks
		WHERE user_id = $1
			AND is_deleted = FALSE
			AND created_at >= $2
			AND created_at < $3 + INTERVAL '1 day'
		GROUP BY DATE(created_at AT TIME ZONE $4)
	),
	tasks_completed_per_day AS (
		SELECT
			DATE(completed_at AT TIME ZONE $4) AS day,
			COUNT(*) AS tasks_completed
		FROM tasks
		WHERE user_id = $1
			AND is_deleted = FALSE
			AND completed_at IS NOT NULL
			AND completed_at >= $2
			AND completed_at < $3 + INTERVAL '1 day'
		GROUP BY DATE(completed_at AT TIME ZONE $4)
	)
	SELECT
		ds.day,
		COALESCE(c.cards_created, 0) AS cards_created,
		COALESCE(tc.tasks_created, 0) AS tasks_created,
		COALESCE(tcomp.tasks_completed, 0) AS tasks_completed
	FROM date_series ds
	LEFT JOIN cards_per_day c ON ds.day = c.day
	LEFT JOIN tasks_created_per_day tc ON ds.day = tc.day
	LEFT JOIN tasks_completed_per_day tcomp ON ds.day = tcomp.day
	ORDER BY ds.day
	`

	rows, err := db.Query(query, userID, startDate, endDate, timezone)
	if err != nil {
		log.Printf("Error querying daily stats: %v", err)
		return []models.DailyStats{}, fmt.Errorf("unable to fetch daily stats")
	}
	defer rows.Close()

	var stats []models.DailyStats
	for rows.Next() {
		var stat models.DailyStats
		if err := rows.Scan(
			&stat.Date,
			&stat.CardsCreated,
			&stat.TasksCreated,
			&stat.TasksCompleted,
		); err != nil {
			log.Printf("Error scanning daily stats row: %v", err)
			continue
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating daily stats rows: %v", err)
		return []models.DailyStats{}, fmt.Errorf("error reading stats data")
	}

	return stats, nil
}

// GetTasksCompletedOnDate retrieves all tasks completed on a specific date
func GetTasksCompletedOnDate(db models.Database, userID int, date time.Time, timezone string) ([]models.Task, error) {
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
		created_at, updated_at, completed_at, title, priority, is_complete
	FROM tasks
	WHERE user_id = $1
		AND is_deleted = FALSE
		AND DATE(completed_at AT TIME ZONE $3) = DATE($2)
	ORDER BY completed_at DESC
	`

	rows, err := db.Query(query, userID, date, timezone)
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

// GetCardsCreatedOnDate retrieves all cards created on a specific date
func GetCardsCreatedOnDate(db models.Database, userID int, date time.Time, timezone string) ([]models.PartialCard, error) {
	query := `
	SELECT id, card_id, title, created_at, updated_at, parent_id, user_id
	FROM cards
	WHERE user_id = $1
		AND is_deleted = FALSE
		AND DATE(created_at AT TIME ZONE $3) = DATE($2)
	ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID, date, timezone)
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
