// Package task provides task-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The task domain contains tools for managing user tasks, including creation,
// updating, completion, deletion, and scheduling.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// The task domain package follows the pattern established by memory_tools for
// splitting tools into separate domain packages. The registration is handled
// in services/task_tools.go to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions for task operations
// 2. Tool handler logic
// 3. Domain-specific business logic for task management
//
// This is a high-usage domain with 7 tools:
// - GetTasks: Retrieve task lists with filtering
// - CreateTask: Create new tasks with scheduling
// - UpdateTask: Update existing task properties
// - GetTaskByID: Retrieve a specific task
// - CompleteTask: Mark tasks as complete
// - DeleteTask: Soft delete tasks
// - CompleteAndScheduleTask: Handle recurring tasks
package task

import (
	"database/sql"
	"fmt"
	"time"

	"go-backend/models"
)

// TaskStatus represents a task status configuration.
type TaskStatus struct {
	ID             int
	Name           string
	DisplayName    string
	Color          string
	Icon           string
	Position       int
	IsDefault      bool
	IsCompleteState bool
}

// GetTasks retrieves all tasks for a user, optionally including completed tasks.
// This is the domain data access function for task listing.
func GetTasks(db *sql.DB, userID int, includeCompleted bool, timezone string) ([]models.Task, error) {
	// Use the services package function which handles complex querying
	// This is a re-export to avoid circular dependencies
	var tasks []models.Task
	var err error

	query := `
		SELECT id, card_pk, user_id, scheduled_date, due_date,
		created_at, updated_at, completed_at, title, description, priority, status, is_complete,
		reminder_time, reminder_sent
		FROM tasks
		WHERE user_id = $1 AND is_deleted = FALSE
	`
	args := []interface{}{userID}

	if !includeCompleted {
		query += ` AND is_complete = FALSE`
	}

	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

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
			&task.Description,
			&task.Priority,
			&task.Status,
			&task.IsComplete,
			&task.ReminderTime,
			&task.ReminderSent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTasksByCard retrieves all tasks associated with a specific card.
// This is the domain data access function for card-filtered task listing.
func GetTasksByCard(db *sql.DB, userID int, cardPK int) ([]models.Task, error) {
	var tasks []models.Task
	query := `
		SELECT id, card_pk, user_id, scheduled_date, due_date,
		created_at, updated_at, completed_at, title, description, priority, status, is_complete,
		reminder_time, reminder_sent
		FROM tasks
		WHERE user_id = $1 AND is_deleted = FALSE AND card_pk = $2
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to query card tasks: %w", err)
	}
	defer rows.Close()

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
			&task.Description,
			&task.Priority,
			&task.Status,
			&task.IsComplete,
			&task.ReminderTime,
			&task.ReminderSent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTask retrieves a single task by ID for a specific user.
// This is the domain data access function for single task retrieval.
func GetTask(db *sql.DB, userID int, taskID int) (models.Task, error) {
	var task models.Task

	query := `
		SELECT id, card_pk, user_id, scheduled_date, due_date,
		created_at, updated_at, completed_at, title, description, priority, status, is_complete,
		reminder_time, reminder_sent
		FROM tasks
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`

	err := db.QueryRow(query, taskID, userID).Scan(
		&task.ID,
		&task.CardPK,
		&task.UserID,
		&task.ScheduledDate,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.CompletedAt,
		&task.Title,
		&task.Description,
		&task.Priority,
		&task.Status,
		&task.IsComplete,
		&task.ReminderTime,
		&task.ReminderSent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Task{}, fmt.Errorf("task not found")
		}
		return models.Task{}, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// CreateTask creates a new task in the database.
// This is the domain data access function for task creation.
func CreateTask(db *sql.DB, task models.Task) (int, error) {
	var taskID int

	query := `
		INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at,
			completed_at, title, description, priority, status, is_complete, is_deleted,
			reminder_time, reminder_sent)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7, $8, $9, $10, FALSE, $11, FALSE)
		RETURNING id
	`

	err := db.QueryRow(
		query,
		task.CardPK,
		task.UserID,
		task.ScheduledDate,
		task.DueDate,
		task.CompletedAt,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		task.IsComplete,
		task.ReminderTime,
	).Scan(&taskID)

	if err != nil {
		return 0, fmt.Errorf("failed to create task: %w", err)
	}

	return taskID, nil
}

// UpdateTask updates an existing task in the database.
// This is the domain data access function for task updates.
func UpdateTask(db *sql.DB, userID int, taskID int, task models.Task) error {
	var completedAt *time.Time

	if task.IsComplete {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE tasks SET
			card_pk = $1,
			scheduled_date = $2,
			due_date = $3,
			updated_at = NOW(),
			completed_at = $4,
			title = $5,
			description = $6,
			priority = $7,
			status = $8,
			is_complete = $9,
			reminder_time = $10,
			reminder_sent = FALSE
		WHERE id = $11 AND user_id = $12 AND is_deleted = FALSE
	`

	_, err := db.Exec(
		query,
		task.CardPK,
		task.ScheduledDate,
		task.DueDate,
		completedAt,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		task.IsComplete,
		task.ReminderTime,
		taskID,
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask soft deletes a task by setting is_deleted flag.
// This is the domain data access function for task deletion.
func DeleteTask(db *sql.DB, userID int, taskID int) error {
	query := `UPDATE tasks SET is_deleted = TRUE WHERE id = $1 AND user_id = $2`

	_, err := db.Exec(query, taskID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// GetTaskStatusByName retrieves a task status configuration by name.
// This is a domain data access function for task status operations.
func GetTaskStatusByName(db *sql.DB, userID int, name string) (TaskStatus, error) {
	var status TaskStatus

	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state
		FROM task_statuses
		WHERE user_id = $1 AND name = $2
	`

	err := db.QueryRow(query, userID, name).Scan(
		&status.ID,
		new(interface{}), // user_id - unused
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return TaskStatus{}, fmt.Errorf("task status not found")
		}
		return TaskStatus{}, fmt.Errorf("failed to get task status: %w", err)
	}

	return status, nil
}

// GetDefaultTaskStatus retrieves the default task status for a user.
// This is a domain data access function for task status operations.
func GetDefaultTaskStatus(db *sql.DB, userID int) (TaskStatus, error) {
	var status TaskStatus

	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state
		FROM task_statuses
		WHERE user_id = $1 AND is_default = TRUE
		LIMIT 1
	`

	err := db.QueryRow(query, userID).Scan(
		&status.ID,
		new(interface{}), // user_id - unused
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return TaskStatus{}, fmt.Errorf("default task status not found")
		}
		return TaskStatus{}, fmt.Errorf("failed to get default task status: %w", err)
	}

	return status, nil
}

// GetCompleteTaskStatus retrieves the complete task status for a user.
// This is a domain data access function for task status operations.
func GetCompleteTaskStatus(db *sql.DB, userID int) (TaskStatus, error) {
	var status TaskStatus

	query := `
		SELECT id, user_id, name, display_name, color, icon, position,
		       is_default, is_complete_state
		FROM task_statuses
		WHERE user_id = $1 AND is_complete_state = TRUE
		LIMIT 1
	`

	err := db.QueryRow(query, userID).Scan(
		&status.ID,
		new(interface{}), // user_id - unused
		&status.Name,
		&status.DisplayName,
		&status.Color,
		&status.Icon,
		&status.Position,
		&status.IsDefault,
		&status.IsCompleteState,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return TaskStatus{}, fmt.Errorf("complete task status not found")
		}
		return TaskStatus{}, fmt.Errorf("failed to get complete task status: %w", err)
	}

	return status, nil
}

// CompleteAndScheduleTask completes the current task and creates a new one
// scheduled for a specified number of days later.
// This is the domain business logic for recurring task management.
func CompleteAndScheduleTask(db *sql.DB, userID int, taskID int, days int, completeStatusName string, defaultStatusName string) (int, error) {
	// Get the original task
	oldTask, err := GetTask(db, userID, taskID)
	if err != nil {
		return 0, fmt.Errorf("unable to query task: %w", err)
	}

	// Calculate the new scheduled date (X days from now)
	now := time.Now()
	newScheduledDate := now.AddDate(0, 0, days)

	// Create the new task first
	newTask := models.Task{
		CardPK:        oldTask.CardPK,
		UserID:        oldTask.UserID,
		ScheduledDate: &newScheduledDate,
		DueDate:       &newScheduledDate,
		CompletedAt:   nil,
		Title:         oldTask.Title,
		Description:   oldTask.Description,
		Priority:      oldTask.Priority,
		Status:        defaultStatusName,
		IsComplete:    false,
		ReminderTime:  nil,
	}

	// If the old task had a reminder time, calculate the new reminder time
	if oldTask.ReminderTime != nil {
		duration := oldTask.ScheduledDate.Sub(*oldTask.ReminderTime)
		newReminderTime := newScheduledDate.Add(duration)
		newTask.ReminderTime = &newReminderTime
	}

	newTaskID, err := CreateTask(db, newTask)
	if err != nil {
		return 0, fmt.Errorf("unable to create new task: %w", err)
	}

	// Now update the original task to be complete
	oldTask.IsComplete = true
	oldTask.Status = completeStatusName
	err = UpdateTask(db, userID, taskID, oldTask)
	if err != nil {
		return 0, fmt.Errorf("unable to mark original task as complete: %w", err)
	}

	return newTaskID, nil
}
