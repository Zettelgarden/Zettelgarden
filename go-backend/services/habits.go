package services

import (
	"database/sql"
	"fmt"
	"time"
	"go-backend/models"
)

func CreateHabit(db *sql.DB, userID int, params models.CreateHabitParams) (int, error) {
	var position int
	if params.Position != nil {
		position = *params.Position
	} else {
		err := db.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM habits WHERE user_id = $1", userID).Scan(&position)
		if err != nil {
			position = 0
		}
	}

	query := `INSERT INTO habits (user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	var id int
	err := db.QueryRow(query, userID, params.Title, params.Description, params.Frequency,
		params.CustomDays, params.Icon, params.Color, position, params.LinkedTaskID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create habit: %w", err)
	}
	return id, nil
}

func GetHabit(db *sql.DB, userID int, habitID int) (models.Habit, error) {
	var habit models.Habit
	var description, customDays, icon, color sql.NullString
	var linkedTaskID sql.NullInt64

	query := `SELECT id, user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id, created_at, updated_at
              FROM habits WHERE id = $1 AND user_id = $2`
	err := db.QueryRow(query, habitID, userID).Scan(
		&habit.ID, &habit.UserID, &habit.Title, &description, &habit.Frequency,
		&customDays, &icon, &color, &habit.Position, &linkedTaskID,
		&habit.CreatedAt, &habit.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return habit, fmt.Errorf("habit not found")
	}
	if err != nil {
		return habit, fmt.Errorf("failed to get habit: %w", err)
	}

	if description.Valid {
		habit.Description = &description.String
	}
	if customDays.Valid {
		habit.CustomDays = &customDays.String
	}
	if icon.Valid {
		habit.Icon = &icon.String
	}
	if color.Valid {
		habit.Color = &color.String
	}
	if linkedTaskID.Valid {
		id := int(linkedTaskID.Int64)
		habit.LinkedTaskID = &id
	}
	return habit, nil
}

func GetHabits(db *sql.DB, userID int) ([]models.Habit, error) {
	query := `SELECT id, user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id, created_at, updated_at
              FROM habits WHERE user_id = $1 ORDER BY position ASC`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get habits: %w", err)
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var habit models.Habit
		var description, customDays, icon, color sql.NullString
		var linkedTaskID sql.NullInt64

		err := rows.Scan(&habit.ID, &habit.UserID, &habit.Title, &description, &habit.Frequency,
			&customDays, &icon, &color, &habit.Position, &linkedTaskID,
			&habit.CreatedAt, &habit.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan habit: %w", err)
		}

		if description.Valid {
			habit.Description = &description.String
		}
		if customDays.Valid {
			habit.CustomDays = &customDays.String
		}
		if icon.Valid {
			habit.Icon = &icon.String
		}
		if color.Valid {
			habit.Color = &color.String
		}
		if linkedTaskID.Valid {
			id := int(linkedTaskID.Int64)
			habit.LinkedTaskID = &id
		}
		habits = append(habits, habit)
	}
	return habits, nil
}

func DeleteHabit(db *sql.DB, userID int, habitID int) error {
	result, err := db.Exec("DELETE FROM habits WHERE id = $1 AND user_id = $2", habitID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete habit: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("habit not found")
	}
	return nil
}

func CheckinHabit(db *sql.DB, userID int, habitID int, params models.CheckinHabitParams, timezone string) (int, error) {
	// Verify habit exists and belongs to user
	_, err := GetHabit(db, userID, habitID)
	if err != nil {
		return 0, fmt.Errorf("habit not found: %w", err)
	}

	// Check for duplicate today
	today := time.Now().UTC().Format("2006-01-02")
	var existingLogID int
	checkQuery := `SELECT id FROM habit_logs WHERE habit_id = $1 AND user_id = $2
                   AND DATE(completed_at AT TIME ZONE $3) = $4`
	err = db.QueryRow(checkQuery, habitID, userID, timezone, today).Scan(&existingLogID)
	if err == nil {
		return 0, fmt.Errorf("already checked in today")
	}

	// Create log entry
	query := `INSERT INTO habit_logs (habit_id, user_id, completed_at, notes) VALUES ($1, $2, $3, $4) RETURNING id`
	var logID int
	completedAt := time.Now().UTC()
	err = db.QueryRow(query, habitID, userID, completedAt, params.Notes).Scan(&logID)
	if err != nil {
		return 0, fmt.Errorf("failed to create habit log: %w", err)
	}
	return logID, nil
}

func DeleteHabitLog(db *sql.DB, userID int, habitID int, logID int) error {
	result, err := db.Exec(`DELETE FROM habit_logs WHERE id = $1 AND habit_id = $2 AND user_id = $3`, logID, habitID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete habit log: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("habit log not found")
	}
	return nil
}

func GetHabitLogs(db *sql.DB, userID int, habitID int, limit, offset int) ([]models.HabitLog, int, error) {
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count habit logs: %w", err)
	}

	query := `SELECT id, habit_id, user_id, completed_at, notes, created_at FROM habit_logs
              WHERE habit_id = $1 AND user_id = $2 ORDER BY completed_at DESC LIMIT $3 OFFSET $4`
	rows, err := db.Query(query, habitID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get habit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.HabitLog
	for rows.Next() {
		var log models.HabitLog
		var notes sql.NullString
		err := rows.Scan(&log.ID, &log.HabitID, &log.UserID, &log.CompletedAt, &notes, &log.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan habit log: %w", err)
		}
		if notes.Valid {
			log.Notes = &notes.String
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}
