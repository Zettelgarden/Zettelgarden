// Package task provides task-related data access and business logic
// for the Zettelgarden tool registry.
//
// This package contains functions for managing tasks and task statuses,
// including CRUD operations, recurring task handling, and dependency management.
package task

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go-backend/models"
)

// GetTask retrieves a single task by ID for a specific user
func GetTask(db models.Database, userID int, id int) (models.Task, error) {
	var task models.Task

	err := db.QueryRow(`
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, description, priority, status, is_complete,
	reminder_time, reminder_sent
	FROM
	tasks
	WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, id, userID).Scan(
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
		log.Printf("err %v", err)
		return models.Task{}, fmt.Errorf("unable to access task")
	}
	if task.CardPK > 0 {
		card, err := getPartialCard(db, userID, task.CardPK)
		if err == nil {
			task.Card = card
		}
	}

	// Load dependencies
	if err := LoadTaskDependencies(db, &task); err != nil {
		log.Printf("Error loading task dependencies: %v", err)
	}

	return task, nil
}

// GetTasksPaginated retrieves tasks for a user with pagination and filtering
func GetTasksPaginated(db models.Database, userID int, limit, offset int, includeCompleted bool, cardID *int, priority *string, scheduledDate *time.Time, completedDate *time.Time, status *string, timezone string) ([]models.Task, int, error) {
	var tasks []models.Task
	var args []interface{}
	argIndex := 1

	// Build base query
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, description, priority, status, is_complete,
	reminder_time, reminder_sent
	FROM tasks
	WHERE user_id = $` + fmt.Sprintf("%d", argIndex) + ` AND is_deleted = FALSE`
	args = append(args, userID)
	argIndex++

	// Add filters
	if !includeCompleted {
		query += ` AND is_complete = FALSE`
	}
	if cardID != nil {
		query += ` AND card_pk = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *cardID)
		argIndex++
	}
	if priority != nil {
		query += ` AND priority = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *priority)
		argIndex++
	}
	if status != nil {
		query += ` AND status = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *status)
		argIndex++
	}
	if scheduledDate != nil {
		query += ` AND DATE(scheduled_date AT TIME ZONE $` + fmt.Sprintf("%d", argIndex) + `) = DATE($` + fmt.Sprintf("%d", argIndex+1) + `)`
		args = append(args, timezone, *scheduledDate)
		argIndex += 2
	}
	if completedDate != nil {
		query += ` AND DATE(completed_at AT TIME ZONE $` + fmt.Sprintf("%d", argIndex) + `) = DATE($` + fmt.Sprintf("%d", argIndex+1) + `)`
		args = append(args, timezone, *completedDate)
		argIndex += 2
	}

	// Add ordering and pagination
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	args = append(args, limit, offset)

	// Execute main query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("err %v", err)
		return []models.Task{}, 0, err
	}
	defer rows.Close()

	// First pass: scan all tasks into the slice
	// We can't do nested queries while iterating over rows with PostgreSQL transactions
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
			log.Printf("err %v", err)
			return []models.Task{}, 0, fmt.Errorf("unable to access task")
		}
		tasks = append(tasks, task)
	}

	// Second pass: load dependencies for each task after rows are closed
	for i := range tasks {
		if tasks[i].CardPK > 0 {
			card, err := getPartialCard(db, userID, tasks[i].CardPK)
			if err == nil {
				tasks[i].Card = card
			}
		}

		// Load dependencies
		if err := LoadTaskDependencies(db, &tasks[i]); err != nil {
			log.Printf("Error loading task dependencies: %v", err)
		}
	}

	// Get total count with same filters
	countQuery := `SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND is_deleted = FALSE`
	countArgs := []interface{}{userID}
	argIndex = 2

	if !includeCompleted {
		countQuery += ` AND is_complete = FALSE`
	}
	if cardID != nil {
		countQuery += ` AND card_pk = $` + fmt.Sprintf("%d", argIndex)
		countArgs = append(countArgs, *cardID)
		argIndex++
	}
	if priority != nil {
		countQuery += ` AND priority = $` + fmt.Sprintf("%d", argIndex)
		countArgs = append(countArgs, *priority)
		argIndex++
	}
	if status != nil {
		countQuery += ` AND status = $` + fmt.Sprintf("%d", argIndex)
		countArgs = append(countArgs, *status)
		argIndex++
	}
	if scheduledDate != nil {
		countQuery += ` AND DATE(scheduled_date AT TIME ZONE $` + fmt.Sprintf("%d", argIndex) + `) = DATE($` + fmt.Sprintf("%d", argIndex+1) + `)`
		countArgs = append(countArgs, timezone, *scheduledDate)
		argIndex += 2
	}
	if completedDate != nil {
		countQuery += ` AND DATE(completed_at AT TIME ZONE $` + fmt.Sprintf("%d", argIndex) + `) = DATE($` + fmt.Sprintf("%d", argIndex+1) + `)`
		countArgs = append(countArgs, timezone, *completedDate)
	}

	var total int
	err = db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		log.Printf("err counting tasks: %v", err)
		return []models.Task{}, 0, err
	}

	return tasks, total, nil
}

// GetTasks retrieves all tasks for a user, optionally including completed tasks
func GetTasks(db models.Database, userID int, includeCompleted bool, timezone string) ([]models.Task, error) {
	tasks, _, err := GetTasksPaginated(db, userID, 1000, 0, includeCompleted, nil, nil, nil, nil, nil, timezone)
	return tasks, err
}

// GetTasksByCard retrieves all tasks associated with a specific card
func GetTasksByCard(db models.Database, userID int, cardPK int) ([]models.Task, error) {
	var tasks []models.Task
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, description, priority, status, is_complete,
	reminder_time, reminder_sent
	FROM
	tasks
	WHERE user_id = $1 AND is_deleted = FALSE AND card_pk = $2
`
	rows, err := db.Query(query, userID, cardPK)
	if err != nil {
		log.Printf("err %v", err)
		return []models.Task{}, err
	}
	defer rows.Close()

	// First pass: scan all tasks into the slice
	// We can't do nested queries while iterating over rows with PostgreSQL transactions
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
			log.Printf("err %v", err)
			return []models.Task{}, fmt.Errorf("unable to access task")
		}
		tasks = append(tasks, task)
	}

	// Second pass: load dependencies for each task after rows are closed
	for i := range tasks {
		if tasks[i].CardPK > 0 {
			card, err := getPartialCard(db, userID, tasks[i].CardPK)
			if err == nil {
				tasks[i].Card = card
			}
		}

		// Load dependencies
		if err := LoadTaskDependencies(db, &tasks[i]); err != nil {
			log.Printf("Error loading task dependencies: %v", err)
		}
	}

	// Note: Tag loading will need to be handled by the handler that calls this
	// since QueryTagsForTask is in the handlers package
	return tasks, nil
}

// GetTasksNeedingReminders retrieves all tasks that need reminder emails sent
func GetTasksNeedingReminders(db models.Database) ([]models.Task, error) {
	var tasks []models.Task
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, description, priority, status, is_complete,
	reminder_time, reminder_sent
	FROM tasks
	WHERE reminder_time <= NOW()
		AND reminder_sent = FALSE
		AND is_complete = FALSE
		AND is_deleted = FALSE
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying tasks needing reminders: %v", err)
		return []models.Task{}, err
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
			log.Printf("Error scanning task needing reminder: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// UpdateTask updates an existing task
// Returns the ID of a newly created recurring task, or 0 if none was created
func UpdateTask(db models.Database, userID int, id int, task models.Task) (int, error) {
	return UpdateTaskWithRecurring(db, userID, id, task, true)
}

// UpdateTaskWithRecurring updates an existing task with control over recurring task creation
// checkRecurring: if false, skip creating recurring tasks (used when task was already rescheduled)
// Returns the ID of a newly created recurring task, or 0 if none was created
func UpdateTaskWithRecurring(db models.Database, userID int, id int, task models.Task, checkRecurring bool) (int, error) {
	oldTask, err := GetTask(db, userID, id)
	if err != nil {
		return 0, fmt.Errorf("unable to query task: %v", err)
	}

	// Default status if empty - use the user's default status
	if task.Status == "" {
		defaultStatus, err := GetDefaultTaskStatus(db, userID)
		if err != nil {
			// Fallback to "todo" if no default status found
			task.Status = "todo"
		} else {
			task.Status = defaultStatus.Name
		}
	}

	// Bidirectional sync between is_complete and status
	// If they disagree, update the status to match is_complete
	statusConfig, err := GetTaskStatusByName(db, userID, task.Status)
	if err == nil {
		if task.IsComplete != statusConfig.IsCompleteState {
			if task.IsComplete {
				// Caller wants to mark complete; update status to the complete status
				if completeStatus, csErr := GetCompleteTaskStatus(db, userID); csErr == nil {
					task.Status = completeStatus.Name
				}
			} else {
				// Caller wants to mark incomplete; update status to the default status
				if defaultStatus, dsErr := GetDefaultTaskStatus(db, userID); dsErr == nil {
					task.Status = defaultStatus.Name
				}
			}
		}
	} else {
		// Fallback to old behavior if status not found
		if task.Status == "done" {
			task.IsComplete = true
		} else if task.IsComplete {
			task.Status = "done"
		}
	}

	var completedAt *time.Time
	var recurringTaskID int
	if task.IsComplete && !oldTask.IsComplete {
		now := time.Now()
		completedAt = &now
		if checkRecurring {
			recurringTaskID, err = checkRecurringTasks(db, task)
			if err != nil {
				log.Printf("err %v", err)
			}
		}
	} else if oldTask.IsComplete {
		completedAt = oldTask.CompletedAt
	} else {
		completedAt = nil
	}

	// Determine if we should reset reminder_sent
	reminderSent := oldTask.ReminderSent
	if task.ReminderTime != nil && oldTask.ReminderTime != nil {
		// If reminder_time changed, reset reminder_sent to FALSE
		if !task.ReminderTime.Equal(*oldTask.ReminderTime) {
			reminderSent = false
		}
	} else if task.ReminderTime != nil && oldTask.ReminderTime == nil {
		// New reminder time set, ensure reminder_sent is FALSE
		reminderSent = false
	} else if task.ReminderTime == nil && oldTask.ReminderTime != nil {
		// Reminder time removed, reset reminder_sent
		reminderSent = false
	}

	_, err = db.Exec(`
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
			reminder_sent = $11
		WHERE id = $12 AND user_id = $13 AND is_deleted = FALSE
	`, task.CardPK, task.ScheduledDate, task.DueDate, completedAt, task.Title, task.Description, task.Priority, task.Status, task.IsComplete, task.ReminderTime, reminderSent, id, userID)

	if err != nil {
		log.Printf("error: %v", err)
		return 0, fmt.Errorf("unable to update task")
	}

	newTask, err := GetTask(db, userID, id)
	if err != nil {
		log.Printf("Error querying updated task for audit: %v", err)
	} else {
		err = createAuditEvent(db, userID, id, "task", "update", oldTask, newTask)
		if err != nil {
			log.Printf("Error creating audit event: %v", err)
		}
	}

	return recurringTaskID, nil
}

// CreateTask creates a new task
func CreateTask(db models.Database, task models.Task) (int, error) {
	var taskID int

	// Log the priority value for debugging
	if task.Priority != nil {
		log.Printf("Priority value: %s", *task.Priority)
	} else {
		log.Printf("Priority is nil")
	}

	// Default status if empty - use the user's default status
	if task.Status == "" {
		defaultStatus, err := GetDefaultTaskStatus(db, task.UserID)
		if err != nil {
			// Fallback to "todo" if no default status found
			task.Status = "todo"
		} else {
			task.Status = defaultStatus.Name
		}
	}

	// Sync is_complete with status based on user's configured statuses
	// Check if the status is marked as a complete state
	statusConfig, statusErr := GetTaskStatusByName(db, task.UserID, task.Status)
	if statusErr == nil {
		// Sync is_complete with the status's is_complete_state
		task.IsComplete = statusConfig.IsCompleteState
	} else {
		// Fallback to old behavior if status not found
		if task.Status == "done" {
			task.IsComplete = true
		} else if task.IsComplete {
			task.Status = "done"
		}
	}

	err := db.QueryRow(`
	INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, priority, status, is_complete, is_deleted, reminder_time, reminder_sent)
	VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7, $8, $9, $10, FALSE, $11, FALSE)
	RETURNING id
	`, task.CardPK, task.UserID, task.ScheduledDate, task.DueDate, task.CompletedAt, task.Title, task.Description, task.Priority, task.Status, task.IsComplete, task.ReminderTime).Scan(&taskID)

	if err != nil {
		log.Printf("err %v", err)
		return 0, fmt.Errorf("unable to create task")
	}

	newTask, err := GetTask(db, task.UserID, taskID)
	if err != nil {
		log.Printf("Error querying new task for audit: %v", err)
	} else {
		err = createAuditEvent(db, task.UserID, taskID, "task", "create", nil, newTask)
		if err != nil {
			log.Printf("Error creating audit event: %v", err)
		}
	}

	return taskID, nil
}

// DeleteTask soft deletes a task
func DeleteTask(db models.Database, userID int, id int) error {
	oldTask, err := GetTask(db, userID, id)
	if err != nil {
		return fmt.Errorf("unable to query task: %v", err)
	}

	_, err = db.Exec(`
	UPDATE tasks SET is_deleted = TRUE
	WHERE id = $1 AND user_id = $2
	`, id, userID)

	if err != nil {
		log.Printf("err %v", err)
		return fmt.Errorf("unable to delete task")
	}

	err = createAuditEvent(db, userID, id, "task", "delete", oldTask, nil)
	if err != nil {
		log.Printf("Error creating audit event: %v", err)
	}

	return nil
}

// ParseRecurringTasks parses a task title to extract recurring task information
func ParseRecurringTasks(title string) (models.RecurringTask, bool) {
	patterns := []struct {
		regex       *regexp.Regexp
		frequency   string
		getInterval func([]string) int
	}{
		{
			regex:       regexp.MustCompile(`(?i)every day|daily`),
			frequency:   "daily",
			getInterval: func([]string) int { return 1 },
		},
		{
			regex:     regexp.MustCompile(`(?i)every (\d+) days?`),
			frequency: "daily",
			getInterval: func(matches []string) int {
				interval, _ := strconv.Atoi(matches[1])
				return interval
			},
		},
		// Weekly patterns
		{
			regex:       regexp.MustCompile(`(?i)every week|weekly`),
			frequency:   "weekly",
			getInterval: func([]string) int { return 7 },
		},
		{
			regex:     regexp.MustCompile(`(?i)every (\d+) weeks?`),
			frequency: "weekly",
			getInterval: func(matches []string) int {
				interval, _ := strconv.Atoi(matches[1])
				return interval
			},
		},
		// Monthly patterns
		{
			regex:       regexp.MustCompile(`(?i)every month|monthly`),
			frequency:   "monthly",
			getInterval: func([]string) int { return 30 },
		},
		{
			regex:     regexp.MustCompile(`(?i)every (\d+) months?`),
			frequency: "monthly",
			getInterval: func(matches []string) int {
				interval, _ := strconv.Atoi(matches[1])
				return interval
			},
		},
	}

	lowercaseTitle := strings.ToLower(title)

	for _, pattern := range patterns {
		matches := pattern.regex.FindStringSubmatch(lowercaseTitle)
		if matches != nil {
			return models.RecurringTask{
				Frequency: pattern.frequency,
				Interval:  pattern.getInterval(matches),
			}, true
		}
	}

	return models.RecurringTask{}, false
}

// checkRecurringTasks handles creating recurring tasks when a task is completed
// Returns the new task ID if a recurring task was created, 0 otherwise
func checkRecurringTasks(db models.Database, task models.Task) (int, error) {
	recurringTask, found := ParseRecurringTasks(task.Title)
	if !found {
		return 0, nil
	}
	var scheduledDate time.Time
	now := time.Now()
	scheduledDate = now.AddDate(0, 0, recurringTask.Interval)

	newTask := models.Task{
		CardPK:        task.CardPK,
		UserID:        task.UserID,
		ScheduledDate: &scheduledDate,
		DueDate:       &scheduledDate,
		CompletedAt:   nil,
		Title:         task.Title,
		Description:   task.Description,
		Priority:      task.Priority,
		Status:        "todo",
		IsComplete:    false,
	}
	taskID, err := CreateTask(db, newTask)
	if err != nil {
		return 0, err
	}
	return taskID, nil
}

// LoadTaskDependencies loads the blocked_by and blocks relationships for a task
func LoadTaskDependencies(db models.Database, task *models.Task) error {
	// Load tasks that block this task (blocked_by)
	blockedByQuery := `
		SELECT t.id, t.title, t.is_complete, t.status
		FROM tasks t
		INNER JOIN task_dependencies td ON t.id = td.blocking_task_id
		WHERE td.task_id = $1 AND t.is_deleted = FALSE
		ORDER BY t.created_at DESC
	`
	blockedByRows, err := db.Query(blockedByQuery, task.ID)
	if err != nil {
		log.Printf("Error loading blocked_by tasks: %v", err)
		return err
	}
	defer blockedByRows.Close()

	task.BlockedBy = []models.PartialTask{}
	for blockedByRows.Next() {
		var pt models.PartialTask
		if err := blockedByRows.Scan(&pt.ID, &pt.Title, &pt.IsComplete, &pt.Status); err != nil {
			log.Printf("Error scanning blocked_by task: %v", err)
			continue
		}
		task.BlockedBy = append(task.BlockedBy, pt)
	}

	// Close blockedByRows before opening another query on the same transaction
	blockedByRows.Close()

	// Load tasks that this task blocks (blocks)
	blocksQuery := `
		SELECT t.id, t.title, t.is_complete, t.status
		FROM tasks t
		INNER JOIN task_dependencies td ON t.id = td.task_id
		WHERE td.blocking_task_id = $1 AND t.is_deleted = FALSE
		ORDER BY t.created_at DESC
	`
	blocksRows, err := db.Query(blocksQuery, task.ID)
	if err != nil {
		log.Printf("Error loading blocks tasks: %v", err)
		return err
	}
	defer blocksRows.Close()

	task.Blocks = []models.PartialTask{}
	for blocksRows.Next() {
		var pt models.PartialTask
		if err := blocksRows.Scan(&pt.ID, &pt.Title, &pt.IsComplete, &pt.Status); err != nil {
			log.Printf("Error scanning blocks task: %v", err)
			continue
		}
		task.Blocks = append(task.Blocks, pt)
	}

	return nil
}

// CompleteAndScheduleTask completes the current task and creates a new one scheduled for X days later
// Returns the ID of the newly created task, or 0 if none was created
func CompleteAndScheduleTask(db models.Database, userID int, id int, days int, completeStatusName string, defaultStatusName string) (int, error) {
	// Get the original task
	oldTask, err := GetTask(db, userID, id)
	if err != nil {
		return 0, fmt.Errorf("unable to query task: %v", err)
	}

	// Calculate the new scheduled date (X days from now)
	now := time.Now()
	newScheduledDate := now.AddDate(0, 0, days)

	// Create the new task first
	newTask := models.Task{
		CardPK:        oldTask.CardPK,
		UserID:        oldTask.UserID,
		ScheduledDate: &newScheduledDate,
		DueDate:       &newScheduledDate, // Also set due date to the same day
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
		// Calculate the duration from scheduled date to reminder time
		duration := oldTask.ScheduledDate.Sub(*oldTask.ReminderTime)
		newReminderTime := newScheduledDate.Add(duration)
		newTask.ReminderTime = &newReminderTime
	}

	newTaskID, err := CreateTask(db, newTask)
	if err != nil {
		return 0, fmt.Errorf("unable to create new task: %v", err)
	}

	// Now update the original task to be complete
	// Pass checkRecurring=false to prevent creating duplicate recurring tasks
	// since we already created a new task above
	oldTask.IsComplete = true
	oldTask.Status = completeStatusName
	_, err = UpdateTaskWithRecurring(db, userID, id, oldTask, false)
	if err != nil {
		log.Printf("Warning: completed task creation but failed to mark original task as complete: %v", err)
		// Don't return error here since we successfully created the new task
	}

	return newTaskID, nil
}

// GetTaskStatuses retrieves all task statuses for a user, ordered by position
func GetTaskStatuses(db models.Database, userID int) ([]models.TaskStatus, error) {
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
func GetTaskStatus(db models.Database, userID int, statusID int) (models.TaskStatus, error) {
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
func GetTaskStatusByName(db models.Database, userID int, name string) (models.TaskStatus, error) {
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
func CreateTaskStatus(db models.Database, userID int, params models.CreateTaskStatusParams) (int, error) {
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
func UpdateTaskStatus(db models.Database, userID int, statusID int, params models.UpdateTaskStatusParams) error {
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
func DeleteTaskStatus(db models.Database, userID int, statusID int) error {
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
func GetDefaultTaskStatus(db models.Database, userID int) (models.TaskStatus, error) {
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
func GetCompleteTaskStatus(db models.Database, userID int) (models.TaskStatus, error) {
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
// Note: This function begins its own transaction, so it needs *sql.DB, not models.Database
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
func ValidateTaskStatus(db models.Database, userID int, statusName string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM task_statuses WHERE user_id = $1 AND name = $2`, userID, statusName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// getPartialCard retrieves a partial card by ID
// This is a local copy of the function from services/cards.go to avoid circular imports.
func getPartialCard(db models.Database, userID, id int) (models.PartialCard, error) {
	var card models.PartialCard

	err := db.QueryRow(`
	SELECT
	id, card_id, user_id, title, parent_id, created_at, updated_at
	FROM cards
	WHERE is_deleted = FALSE AND id = $1 AND user_id = $2
	`, id, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		log.Printf("query partial by id err %v", err)
		return models.PartialCard{}, fmt.Errorf("something went wrong")
	}
	return card, nil
}

// createAuditEvent creates an audit event for tracking changes
// This is a local copy of the function from services/logs.go to avoid circular imports.
func createAuditEvent(db models.Database, userID int, entityID int, entityType string, action string, oldState interface{}, newState interface{}) error {
	changes := make(map[string]models.FieldChange)

	// If we have both states, compute the differences
	if oldState != nil && newState != nil {
		oldVal := reflect.ValueOf(oldState)
		newVal := reflect.ValueOf(newState)

		// Handle pointer types
		if oldVal.Kind() == reflect.Ptr {
			oldVal = oldVal.Elem()
		}
		if newVal.Kind() == reflect.Ptr {
			newVal = newVal.Elem()
		}

		// Only process if both are structs
		if oldVal.Kind() == reflect.Struct && newVal.Kind() == reflect.Struct {
			for i := 0; i < oldVal.NumField(); i++ {
				field := oldVal.Type().Field(i)
				oldField := oldVal.Field(i)
				newField := newVal.Field(i)

				// Skip certain fields
				if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
					continue
				}

				// Convert interface values to comparable types
				oldValue := oldField.Interface()
				newValue := newField.Interface()

				// Only record if values are different
				if !reflect.DeepEqual(oldValue, newValue) {
					changes[field.Name] = models.FieldChange{
						From: oldValue,
						To:   newValue,
					}
				}
			}
		}
	}

	details := models.Details{
		ChangeType: action,
		Changes:    changes,
	}

	// For create/delete actions, store the full state
	if action == "create" && newState != nil {
		details.CustomData = map[string]interface{}{
			"initial_state": newState,
		}
	} else if action == "delete" && oldState != nil {
		details.CustomData = map[string]interface{}{
			"final_state": oldState,
		}
	}

	_, err := db.Exec(`
		INSERT INTO audit_events (user_id, entity_id, entity_type, action, details)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, entityID, entityType, action, details)

	if err != nil {
		log.Printf("Error creating audit event: %v", err)
		return err
	}

	return nil
}
