package services

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
)

// GetTaskStatuses retrieves all task statuses for a user, ordered by position
func GetTaskStatuses(db models.DBTX, userID int) ([]models.TaskStatus, error) {
	var statuses []models.TaskStatus
	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state, created_at, updated_at
		FROM task_statuses
		WHERE user_id = $1
		ORDER BY position ASC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying task statuses: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status models.TaskStatus
		err := rows.Scan(
			&status.ID,
			&status.UserID,
			&status.Name,
			&status.DisplayName,
			&status.Color,
			&status.Icon,
			&status.Position,
			&status.IsDefault,
			&status.IsCompleteState,
			&status.CreatedAt,
			&status.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning task status: %v", err)
			return nil, err
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetTaskStatus retrieves a single task status by ID
func GetTaskStatus(db models.DBTX, userID int, statusID int) (models.TaskStatus, error) {
	var status models.TaskStatus
	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state, created_at, updated_at
		FROM task_statuses
		WHERE id = $1 AND user_id = $2
	`
	err := db.QueryRow(query, statusID, userID).Scan(
		&status.ID,
		&status.UserID,
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
		&status.CreatedAt,
		&status.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error getting task status: %v", err)
		return models.TaskStatus{}, err
	}
	return status, nil
}

// GetTaskStatusByName retrieves a task status by name
func GetTaskStatusByName(db models.DBTX, userID int, name string) (models.TaskStatus, error) {
	var status models.TaskStatus
	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state, created_at, updated_at
		FROM task_statuses
		WHERE user_id = $1 AND name = $2
	`
	err := db.QueryRow(query, userID, name).Scan(
		&status.ID,
		&status.UserID,
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
		&status.CreatedAt,
		&status.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error getting task status by name: %v", err)
		return models.TaskStatus{}, err
	}
	return status, nil
}

// CreateTaskStatus creates a new task status
func CreateTaskStatus(db models.DBTX, userID int, params models.CreateTaskStatusParams) (int, error) {
	// If this is being set as default, unset other defaults
	if params.IsDefault {
		_, err := db.Exec(`UPDATE task_statuses SET is_default = FALSE WHERE user_id = $1`, userID)
		if err != nil {
			log.Printf("Error unsetting other defaults: %v", err)
			return 0, err
		}
	}

	// If this is being set as complete state, unset other complete states
	if params.IsCompleteState {
		_, err := db.Exec(`UPDATE task_statuses SET is_complete_state = FALSE WHERE user_id = $1`, userID)
		if err != nil {
			log.Printf("Error unsetting other complete states: %v", err)
			return 0, err
		}
	}

	var statusID int
	query := `
		INSERT INTO task_statuses (user_id, name, display_name, color, icon, position, is_default, is_complete_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err := db.QueryRow(
		query,
		userID,
		params.Name,
		params.DisplayName,
		params.Color,
		params.Icon,
		params.Position,
		params.IsDefault,
		params.IsCompleteState,
	).Scan(&statusID)

	if err != nil {
		log.Printf("Error creating task status: %v", err)
		return 0, err
	}

	return statusID, nil
}

// UpdateTaskStatus updates an existing task status
func UpdateTaskStatus(db models.DBTX, userID int, statusID int, params models.UpdateTaskStatusParams) error {
	// Get current status
	currentStatus, err := GetTaskStatus(db, userID, statusID)
	if err != nil {
		return fmt.Errorf("task status not found")
	}

	// If setting as default, unset other defaults
	if params.IsDefault != nil && *params.IsDefault {
		_, err := db.Exec(`UPDATE task_statuses SET is_default = FALSE WHERE user_id = $1 AND id != $2`, userID, statusID)
		if err != nil {
			log.Printf("Error unsetting other defaults: %v", err)
			return err
		}
	}

	// If setting as complete state, unset other complete states
	if params.IsCompleteState != nil && *params.IsCompleteState {
		_, err := db.Exec(`UPDATE task_statuses SET is_complete_state = FALSE WHERE user_id = $1 AND id != $2`, userID, statusID)
		if err != nil {
			log.Printf("Error unsetting other complete states: %v", err)
			return err
		}
	}

	// Build dynamic update query
	query := `UPDATE task_statuses SET updated_at = NOW()`
	args := []interface{}{}
	argCount := 1

	if params.DisplayName != nil {
		query += fmt.Sprintf(`, display_name = $%d`, argCount)
		args = append(args, *params.DisplayName)
		argCount++
	}
	if params.Color != nil {
		query += fmt.Sprintf(`, color = $%d`, argCount)
		args = append(args, *params.Color)
		argCount++
	}
	if params.Icon != nil {
		query += fmt.Sprintf(`, icon = $%d`, argCount)
		args = append(args, *params.Icon)
		argCount++
	}
	if params.Position != nil {
		query += fmt.Sprintf(`, position = $%d`, argCount)
		args = append(args, *params.Position)
		argCount++
	}
	if params.IsDefault != nil {
		query += fmt.Sprintf(`, is_default = $%d`, argCount)
		args = append(args, *params.IsDefault)
		argCount++
	}
	if params.IsCompleteState != nil {
		query += fmt.Sprintf(`, is_complete_state = $%d`, argCount)
		args = append(args, *params.IsCompleteState)
		argCount++
	}

	query += fmt.Sprintf(` WHERE id = $%d AND user_id = $%d`, argCount, argCount+1)
	args = append(args, statusID, userID)

	_, err = db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating task status: %v", err)
		return err
	}

	// If we just unset the default or complete state, ensure there's still at least one
	if params.IsDefault != nil && !*params.IsDefault && currentStatus.IsDefault {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND is_default = TRUE`, userID).Scan(&count)
		if err == nil && count == 0 {
			return fmt.Errorf("cannot unset default status: at least one status must be marked as default")
		}
	}

	if params.IsCompleteState != nil && !*params.IsCompleteState && currentStatus.IsCompleteState {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND is_complete_state = TRUE`, userID).Scan(&count)
		if err == nil && count == 0 {
			return fmt.Errorf("cannot unset complete state: at least one status must be marked as complete")
		}
	}

	return nil
}

// DeleteTaskStatus deletes a task status
func DeleteTaskStatus(db models.DBTX, userID int, statusID int) error {
	// Get the status to be deleted
	status, err := GetTaskStatus(db, userID, statusID)
	if err != nil {
		return fmt.Errorf("task status not found")
	}

	// Check if this is the last default status
	if status.IsDefault {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND is_default = TRUE`, userID).Scan(&count)
		if err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("cannot delete the last default status")
		}
	}

	// Check if this is the last complete state status
	if status.IsCompleteState {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND is_complete_state = TRUE`, userID).Scan(&count)
		if err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("cannot delete the last complete status")
		}
	}

	// Get the default status to reassign tasks to
	defaultStatus, err := GetDefaultTaskStatus(db, userID)
	if err != nil {
		return fmt.Errorf("no default status found for reassignment")
	}

	// Reassign all tasks with this status to the default status
	_, err = db.Exec(`UPDATE tasks SET status = $1 WHERE user_id = $2 AND status = $3`, defaultStatus.Name, userID, status.Name)
	if err != nil {
		log.Printf("Error reassigning tasks: %v", err)
		return err
	}

	// Delete the status
	_, err = db.Exec(`DELETE FROM task_statuses WHERE id = $1 AND user_id = $2`, statusID, userID)
	if err != nil {
		log.Printf("Error deleting task status: %v", err)
		return err
	}

	return nil
}

// GetDefaultTaskStatus retrieves the default task status for a user
func GetDefaultTaskStatus(db models.DBTX, userID int) (models.TaskStatus, error) {
	var status models.TaskStatus
	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state, created_at, updated_at
		FROM task_statuses
		WHERE user_id = $1 AND is_default = TRUE
		LIMIT 1
	`
	err := db.QueryRow(query, userID).Scan(
		&status.ID,
		&status.UserID,
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
		&status.CreatedAt,
		&status.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error getting default task status: %v", err)
		return models.TaskStatus{}, err
	}
	return status, nil
}

// GetCompleteTaskStatus retrieves the complete task status for a user
func GetCompleteTaskStatus(db models.DBTX, userID int) (models.TaskStatus, error) {
	var status models.TaskStatus
	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state, created_at, updated_at
		FROM task_statuses
		WHERE user_id = $1 AND is_complete_state = TRUE
		LIMIT 1
	`
	err := db.QueryRow(query, userID).Scan(
		&status.ID,
		&status.UserID,
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
		&status.CreatedAt,
		&status.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error getting complete task status: %v", err)
		return models.TaskStatus{}, err
	}
	return status, nil
}

// ReorderTaskStatuses updates the position of multiple statuses
// Note: This function begins its own transaction, so it needs *sql.DB, not models.DBTX
func ReorderTaskStatuses(db *sql.DB, userID int, statusIDs []int) error {
	// Use a transaction to ensure atomicity
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First, set all positions to negative values to avoid conflicts
	_, err = tx.Exec(`
		UPDATE task_statuses
		SET position = -position - 1
		WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Printf("Error setting temporary positions: %v", err)
		return err
	}

	// Now update each status with its new position
	for i, statusID := range statusIDs {
		_, err := tx.Exec(`
			UPDATE task_statuses
			SET position = $1, updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, i, statusID, userID)
		if err != nil {
			log.Printf("Error reordering task status %d: %v", statusID, err)
			return err
		}
	}

	return tx.Commit()
}

// ValidateTaskStatus checks if a status name is valid for a user
func ValidateTaskStatus(db models.DBTX, userID int, statusName string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND name = $2`, userID, statusName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
