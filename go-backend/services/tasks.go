package services

import (
	"fmt"
	"go-backend/models"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GetTask retrieves a single task by ID for a specific user
func GetTask(db models.Database, userID int, id int) (models.Task, error) {
	var task models.Task

	err := db.QueryRow(`
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, description, priority, status, is_complete,
	reminder_time, reminder_sent, parent_task_id, sort_order
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
		&task.ParentTaskID,
		&task.SortOrder,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.Task{}, fmt.Errorf("unable to access task")
	}
	if task.CardPK > 0 {
		card, err := GetPartialCard(db, userID, task.CardPK)
		if err == nil {
			task.Card = card
		}
	}

	// Load dependencies
	if err := LoadTaskDependencies(db, &task); err != nil {
		log.Printf("Error loading task dependencies: %v", err)
	}

	// Load subtasks
	subtasks, err := GetSubtasks(db, userID, id)
	if err == nil {
		task.Subtasks = subtasks
	}

	// Load parent title if has parent
	if task.ParentTaskID != nil {
		var parentTitle string
		err := db.QueryRow(`SELECT title FROM tasks WHERE id = $1`, *task.ParentTaskID).Scan(&parentTitle)
		if err == nil {
			task.ParentTitle = parentTitle
		}
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
	reminder_time, reminder_sent, parent_task_id, sort_order
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
			&task.ParentTaskID,
			&task.SortOrder,
		); err != nil {
			log.Printf("err %v", err)
			return []models.Task{}, 0, fmt.Errorf("unable to access task")
		}
		tasks = append(tasks, task)
	}

	// Second pass: load dependencies for each task after rows are closed
	for i := range tasks {
		if tasks[i].CardPK > 0 {
			card, err := GetPartialCard(db, userID, tasks[i].CardPK)
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
	reminder_time, reminder_sent, parent_task_id, sort_order
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
			&task.ParentTaskID,
			&task.SortOrder,
		); err != nil {
			log.Printf("err %v", err)
			return []models.Task{}, fmt.Errorf("unable to access task")
		}
		tasks = append(tasks, task)
	}

	// Second pass: load dependencies for each task after rows are closed
	for i := range tasks {
		if tasks[i].CardPK > 0 {
			card, err := GetPartialCard(db, userID, tasks[i].CardPK)
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
	WHERE reminder_time <= CURRENT_TIMESTAMP
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
			updated_at = CURRENT_TIMESTAMP,
			completed_at = $4,
			title = $5,
			description = $6,
			priority = $7,
			status = $8,
			is_complete = $9,
			reminder_time = $10,
			reminder_sent = $11,
			sort_order = $12
		WHERE id = $13 AND user_id = $14 AND is_deleted = FALSE
	`, task.CardPK, task.ScheduledDate, task.DueDate, completedAt, task.Title, task.Description, task.Priority, task.Status, task.IsComplete, task.ReminderTime, reminderSent, task.SortOrder, id, userID)

	if err != nil {
		log.Printf("error: %v", err)
		return 0, fmt.Errorf("unable to update task")
	}

	newTask, err := GetTask(db, userID, id)
	if err != nil {
		log.Printf("Error querying updated task for audit: %v", err)
	} else {
		err = CreateAuditEvent(db, userID, id, "task", "update", oldTask, newTask)
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
	INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, priority, status, is_complete, is_deleted, reminder_time, reminder_sent, parent_task_id, sort_order)
	VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $5, $6, $7, $8, $9, $10, FALSE, $11, FALSE, $12, $13)
	RETURNING id
	`, task.CardPK, task.UserID, task.ScheduledDate, task.DueDate, task.CompletedAt, task.Title, task.Description, task.Priority, task.Status, task.IsComplete, task.ReminderTime, task.ParentTaskID, task.SortOrder).Scan(&taskID)

	if err != nil {
		log.Printf("err %v", err)
		return 0, fmt.Errorf("unable to create task")
	}

	// user_stats was trigger-maintained (0093); now maintained in Go (Phase 5).
	IncrementUserTaskCount(db, task.UserID)

	newTask, err := GetTask(db, task.UserID, taskID)
	if err != nil {
		log.Printf("Error querying new task for audit: %v", err)
	} else {
		err = CreateAuditEvent(db, task.UserID, taskID, "task", "create", nil, newTask)
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

	// user_stats was trigger-maintained (0093); now maintained in Go (Phase 5).
	DecrementUserTaskCount(db, userID)

	err = CreateAuditEvent(db, userID, id, "task", "delete", oldTask, nil)
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

// PrepareSubtask creates a subtask from a parent task, inheriting priority and tags
// Dates are NOT inherited - subtasks get their own scheduling
func PrepareSubtask(parent *models.Task, input models.Task) models.Task {
	subtask := models.Task{
		ParentTaskID: &parent.ID,
		Title:        input.Title,
		Description:  input.Description,
		Priority:     input.Priority,
		Status:       input.Status,
		UserID:       parent.UserID,
		CardPK:       parent.CardPK,
	}

	// Inherit priority from parent if not provided in input
	if subtask.Priority == nil && parent.Priority != nil {
		subtask.Priority = parent.Priority
	}

	// Inherit tags from parent
	subtask.Tags = make([]models.Tag, len(parent.Tags))
	copy(subtask.Tags, parent.Tags)

	return subtask
}

// ValidateParentAssignment checks if a task can be assigned to a parent
// Returns an error if:
// - Task is being assigned to itself
// - Parent already has a parent (would create multi-level nesting)
// - Task has children (can't make a parent into a child)
func ValidateParentAssignment(task *models.Task, parent *models.Task) error {
	// No self-reference
	if task.ID == parent.ID {
		return fmt.Errorf("task cannot be its own parent")
	}

	// Single level only: parent cannot have a parent
	if parent.ParentTaskID != nil {
		return fmt.Errorf("cannot nest more than one level deep")
	}

	// Task with children cannot become a subtask
	if len(task.Subtasks) > 0 {
		return fmt.Errorf("cannot make a parent task into a subtask")
	}

	return nil
}

// IncompleteSubtaskError is returned when trying to complete a task with incomplete subtasks
type IncompleteSubtaskError struct {
	IncompleteCount int
	TotalSubtasks   int
}

func (e *IncompleteSubtaskError) Error() string {
	return "complete all subtasks first"
}

// ValidateTaskCompletion checks if a task can be completed
// Returns an error if the task has incomplete subtasks and force is false
func ValidateTaskCompletion(task *models.Task, force bool) error {
	if force {
		return nil
	}

	incompleteCount := 0
	for _, subtask := range task.Subtasks {
		if !subtask.IsComplete {
			incompleteCount++
		}
	}

	if incompleteCount > 0 {
		return &IncompleteSubtaskError{
			IncompleteCount: incompleteCount,
			TotalSubtasks:   len(task.Subtasks),
		}
	}

	return nil
}

// GetSubtasks retrieves all subtasks for a parent task
func GetSubtasks(db models.Database, userID int, parentTaskID int) ([]models.Task, error) {
	query := `
		SELECT id, title, is_complete, status, priority, scheduled_date, due_date, 
		       description, created_at, updated_at, completed_at, user_id, card_pk
		FROM tasks
		WHERE user_id = $1 AND parent_task_id = $2 AND is_deleted = FALSE
		ORDER BY created_at
	`

	rows, err := db.Query(query, userID, parentTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subtasks []models.Task
	for rows.Next() {
		var t models.Task
		var scheduledDate, dueDate, completedAt *time.Time
		var priority, description *string

		err := rows.Scan(
			&t.ID, &t.Title, &t.IsComplete, &t.Status, &priority,
			&scheduledDate, &dueDate, &description,
			&t.CreatedAt, &t.UpdatedAt, &completedAt, &t.UserID, &t.CardPK,
		)
		if err != nil {
			return nil, err
		}

		t.Priority = priority
		t.ScheduledDate = scheduledDate
		t.DueDate = dueDate
		t.Description = description
		t.CompletedAt = completedAt

		subtasks = append(subtasks, t)
	}

	if subtasks == nil {
		subtasks = []models.Task{}
	}

	return subtasks, nil
}

// UpdateTaskParent updates the parent_task_id of a task
func UpdateTaskParent(db models.Database, userID int, taskID int, parentTaskID *int) error {
	query := `UPDATE tasks SET parent_task_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`
	result, err := db.Exec(query, parentTaskID, taskID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// PrepareSubtask creates a subtask task model with inherited properties
// This is the exported version for use by handlers

// ReorderTasks updates the sort_order for multiple tasks in a single transaction
func ReorderTasks(db models.Database, userID int, orders []struct {
	ID        int `json:"id"`
	SortOrder int `json:"sort_order"`
}) error {
	for _, item := range orders {
		_, err := db.Exec(`UPDATE tasks SET sort_order = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3 AND is_deleted = FALSE`, item.SortOrder, item.ID, userID)
		if err != nil {
			return fmt.Errorf("failed to update sort_order for task %d: %w", item.ID, err)
		}
	}
	return nil
}
