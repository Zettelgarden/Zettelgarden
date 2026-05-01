package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"time"
)

func CreateHabit(db models.Database, userID int, params models.CreateHabitParams) (int, error) {
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

func GetHabit(db models.Database, userID int, habitID int) (models.Habit, error) {
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

func GetHabits(db models.Database, userID int) ([]models.Habit, error) {
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

func DeleteHabit(db models.Database, userID int, habitID int) error {
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

func UpdateHabit(db models.Database, userID int, habitID int, params models.UpdateHabitParams) (models.Habit, error) {
	// First verify the habit exists and belongs to the user
	habit, err := GetHabit(db, userID, habitID)
	if err != nil {
		return habit, err
	}

	// Build dynamic update query based on provided fields
	query := "UPDATE habits SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	if params.Title != nil {
		query += fmt.Sprintf(", title = $%d", argIndex)
		args = append(args, *params.Title)
		argIndex++
	}
	if params.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *params.Description)
		argIndex++
	}
	if params.Frequency != nil {
		query += fmt.Sprintf(", frequency = $%d", argIndex)
		args = append(args, *params.Frequency)
		argIndex++
	}
	if params.CustomDays != nil {
		query += fmt.Sprintf(", custom_days = $%d", argIndex)
		args = append(args, *params.CustomDays)
		argIndex++
	}
	if params.Icon != nil {
		query += fmt.Sprintf(", icon = $%d", argIndex)
		args = append(args, *params.Icon)
		argIndex++
	}
	if params.Color != nil {
		query += fmt.Sprintf(", color = $%d", argIndex)
		args = append(args, *params.Color)
		argIndex++
	}
	if params.Position != nil {
		query += fmt.Sprintf(", position = $%d", argIndex)
		args = append(args, *params.Position)
		argIndex++
	}
	if params.LinkedTaskID != nil {
		query += fmt.Sprintf(", linked_task_id = $%d", argIndex)
		args = append(args, *params.LinkedTaskID)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND user_id = $%d", argIndex, argIndex+1)
	args = append(args, habitID, userID)

	_, err = db.Exec(query, args...)
	if err != nil {
		return habit, fmt.Errorf("failed to update habit: %w", err)
	}

	// Return the updated habit
	return GetHabit(db, userID, habitID)
}

func CheckinHabit(db models.Database, userID int, habitID int, params models.CheckinHabitParams, timezone string) (int, error) {
	// Verify habit exists and belongs to user
	_, err := GetHabit(db, userID, habitID)
	if err != nil {
		return 0, fmt.Errorf("habit not found: %w", err)
	}

	// Check for duplicate today - use user's local date for comparison
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Failed to load timezone %s, falling back to UTC: %v", timezone, err)
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
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

func DeleteHabitLog(db models.Database, userID int, habitID int, logID int) error {
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

func GetHabitLogs(db models.Database, userID int, habitID int, limit, offset int) ([]models.HabitLog, int, error) {
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

func CalculateHabitStats(db models.Database, userID int, habitID int, timezone string) (models.HabitStats, error) {
	var stats models.HabitStats

	// Total completions
	db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&stats.TotalCompletions)

	// Last completion
	var lastCompleted sql.NullTime
	db.QueryRow("SELECT MAX(completed_at) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&lastCompleted)
	if lastCompleted.Valid {
		stats.LastCompletedAt = &lastCompleted.Time
	}

	stats.CurrentStreak = calculateStreak(db, userID, habitID, timezone)
	stats.LongestStreak = calculateLongestStreak(db, userID, habitID, timezone)
	stats.CompletionRate7d = calculateCompletionRate(db, userID, habitID, 7, timezone)
	stats.CompletionRate30d = calculateCompletionRate(db, userID, habitID, 30, timezone)

	return stats, nil
}

func calculateStreak(db models.Database, userID int, habitID int, timezone string) int {
	query := `SELECT DISTINCT DATE(completed_at AT TIME ZONE $1) as date FROM habit_logs
              WHERE habit_id = $2 AND user_id = $3 ORDER BY date DESC`
	rows, err := db.Query(query, timezone, habitID, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if rows.Scan(&date) == nil {
			dates = append(dates, date)
		}
	}

	if len(dates) == 0 {
		return 0
	}

	streak := 1
	for i := 0; i < len(dates)-1; i++ {
		diff := dates[i].Sub(dates[i+1]).Hours() / 24
		if diff <= 1.0 {
			streak++
		} else {
			break
		}
	}
	return streak
}

func calculateLongestStreak(db models.Database, userID int, habitID int, timezone string) int {
	query := `SELECT DISTINCT DATE(completed_at AT TIME ZONE $1) as date FROM habit_logs
              WHERE habit_id = $2 AND user_id = $3 ORDER BY date ASC`
	rows, err := db.Query(query, timezone, habitID, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if rows.Scan(&date) == nil {
			dates = append(dates, date)
		}
	}

	if len(dates) == 0 {
		return 0
	}

	longest := 1
	current := 1
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1]).Hours() / 24
		if diff <= 1.0 {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 1
		}
	}
	return longest
}

func calculateCompletionRate(db models.Database, userID int, habitID int, days int, timezone string) float64 {
	query := `SELECT COUNT(DISTINCT DATE(completed_at AT TIME ZONE $1)) FROM habit_logs
              WHERE habit_id = $2 AND user_id = $3 AND completed_at >= NOW() - INTERVAL '1 day' * $4`
	var completedDays int
	db.QueryRow(query, timezone, habitID, userID, days).Scan(&completedDays)
	return float64(completedDays) / float64(days)
}

type HabitWithCheckin struct {
	models.Habit
	IsDueToday     bool `json:"is_due_today"`
	IsOverdue      bool `json:"is_overdue"`
	CheckedInToday bool `json:"checked_in_today"`
	TodayLogID     *int `json:"today_log_id,omitempty"`
}

func GetTodaysHabits(db models.Database, userID int, timezone string) ([]HabitWithCheckin, error) {
	habits, err := GetHabits(db, userID)
	if err != nil {
		return nil, err
	}

	// Use user's local time for determining "today" and weekday
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Failed to load timezone %s, falling back to UTC: %v", timezone, err)
		loc = time.UTC
	}
	nowInTimezone := time.Now().In(loc)

	var result []HabitWithCheckin
	today := nowInTimezone.Format("2006-01-02")
	currentWeekday := int(nowInTimezone.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // Sunday is 0, convert to 7
	}

	for _, habit := range habits {
		isDueToday := isHabitDueToday(&habit, currentWeekday)

		// Find the most recent due date (today or earlier, up to 7 days back)
		mostRecentDue := findMostRecentDueDate(&habit, currentWeekday, nowInTimezone)
		if mostRecentDue == "" {
			continue
		}

		// Check if there's a check-in on or after the most recent due date.
		// Using >= so that checking in today for a habit due yesterday counts as fulfilled.
		var logID sql.NullInt64
		checkQuery := `SELECT id FROM habit_logs WHERE habit_id = $1 AND user_id = $2
                       AND DATE(completed_at AT TIME ZONE $3) >= $4
                       ORDER BY completed_at DESC LIMIT 1`
		err := db.QueryRow(checkQuery, habit.ID, userID, timezone, mostRecentDue).Scan(&logID)
		fulfilledSinceDue := (err == nil)

		// Show habit if it's due today, OR if it has an unfulfilled past due date
		if isDueToday || !fulfilledSinceDue {
			var hc HabitWithCheckin
			hc.Habit = habit
			hc.IsDueToday = isDueToday
			hc.IsOverdue = !isDueToday && !fulfilledSinceDue

			// Check specifically if checked in today (for the check-in button state)
			var todayLogID sql.NullInt64
			todayCheckQuery := `SELECT id FROM habit_logs WHERE habit_id = $1 AND user_id = $2
                               AND DATE(completed_at AT TIME ZONE $3) = $4 LIMIT 1`
			db.QueryRow(todayCheckQuery, habit.ID, userID, timezone, today).Scan(&todayLogID)
			hc.CheckedInToday = (todayLogID.Valid)
			if todayLogID.Valid {
				id := int(todayLogID.Int64)
				hc.TodayLogID = &id
			}

			result = append(result, hc)
		}
	}
	return result, nil
}

// findMostRecentDueDate returns the date string of the most recent day this habit was due,
// looking back from today up to 7 days. Returns "" if no due date is found.
func findMostRecentDueDate(habit *models.Habit, currentWeekday int, today time.Time) string {
	switch habit.Frequency {
	case models.FrequencyDaily:
		return today.Format("2006-01-02")
	case models.FrequencyWeekly, models.FrequencyCustom:
		if habit.CustomDays != nil {
			var customDays []int
			json.Unmarshal([]byte(*habit.CustomDays), &customDays)
			// Check today, then go back up to 6 more days
			for offset := 0; offset <= 6; offset++ {
				checkDate := today.AddDate(0, 0, -offset)
				checkWeekday := int(checkDate.Weekday())
				if checkWeekday == 0 {
					checkWeekday = 7
				}
				for _, day := range customDays {
					if day == checkWeekday {
						return checkDate.Format("2006-01-02")
					}
				}
			}
		}
		return ""
	default:
		return today.Format("2006-01-02")
	}
}

func isHabitDueToday(habit *models.Habit, currentWeekday int) bool {
	switch habit.Frequency {
	case models.FrequencyDaily:
		return true
	case models.FrequencyWeekly, models.FrequencyCustom:
		if habit.CustomDays != nil {
			var customDays []int
			json.Unmarshal([]byte(*habit.CustomDays), &customDays)
			for _, day := range customDays {
				if day == currentWeekday {
					return true
				}
			}
		}
		return false
	default:
		return true
	}
}
