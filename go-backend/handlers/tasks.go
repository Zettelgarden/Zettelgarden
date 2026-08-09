package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func (s *Handler) QueryTask(userID int, id int) (models.Task, error) {
	task, err := services.GetTask(s.GetDB(), userID, id)
	if err != nil {
		return models.Task{}, err
	}

	// Load tags for the task
	tags, err := s.QueryTagsForTask(userID, task.ID)
	if err == nil {
		task.Tags = tags
	}

	// Convert task times to user's timezone
	userTimezone, err := s.GetUserTimezone(userID)
	if err == nil {
		services.ConvertTaskTimesToUserTimezone(&task, userTimezone)
	}

	return task, nil
}

func (s *Handler) QueryTasks(userID int, includeCompleted bool) ([]models.Task, error) {
	userTimezone, tzErr := s.GetUserTimezone(userID)
	if tzErr != nil {
		userTimezone = "UTC" // Fallback to UTC on error
	}
	tasks, err := services.GetTasks(s.GetDB(), userID, includeCompleted, userTimezone)
	if err != nil {
		return []models.Task{}, err
	}

	// Load tags for each task
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
	}

	// Convert task times to user's timezone
	if tzErr == nil {
		for i := range tasks {
			services.ConvertTaskTimesToUserTimezone(&tasks[i], userTimezone)
		}
	}

	return tasks, nil
}

func (s *Handler) QueryTasksPaginated(userID int, limit, offset int, includeCompleted bool, cardID *int, priority *string, timezone string) ([]models.Task, int, error) {
	tasks, total, err := services.GetTasksPaginated(s.GetDB(), userID, limit, offset, includeCompleted, cardID, priority, nil, nil, nil, timezone)
	if err != nil {
		return []models.Task{}, 0, err
	}

	// Load tags for each task
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
	}

	// Convert task times to user's timezone
	if timezone == "" {
		timezone = "UTC"
	}
	for i := range tasks {
		services.ConvertTaskTimesToUserTimezone(&tasks[i], timezone)
	}

	return tasks, total, nil
}
func (s *Handler) QueryTasksByCard(userID int, cardPK int) ([]models.Task, error) {
	tasks, err := services.GetTasksByCard(s.GetDB(), userID, cardPK)
	if err != nil {
		return []models.Task{}, err
	}

	// Load tags for each task
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
	}

	// Convert task times to user's timezone
	userTimezone, err := s.GetUserTimezone(userID)
	if err == nil {
		for i := range tasks {
			services.ConvertTaskTimesToUserTimezone(&tasks[i], userTimezone)
		}
	}

	return tasks, nil
}

func (s *Handler) GetTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid task id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	task, err := s.QueryTask(userID, id)
	if err != nil {
		log.Printf("Error querying task %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Handler) GetTasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	completed := r.URL.Query().Get("completed")
	includeCompleted := false
	if completed == "true" {
		includeCompleted = true
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20 // default limit
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	var cardID *int
	if cardIDStr := r.URL.Query().Get("card_id"); cardIDStr != "" {
		if id, err := strconv.Atoi(cardIDStr); err == nil {
			cardID = &id
		}
	}

	var priority *string
	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		priority = &priorityStr
	}

	var status *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = &statusStr
	}

	var scheduledDate *time.Time
	if scheduledDateStr := r.URL.Query().Get("scheduled_date"); scheduledDateStr != "" {
		if t, err := time.Parse("2006-01-02", scheduledDateStr); err == nil {
			scheduledDate = &t
		}
	}

	var completedDate *time.Time
	if completedDateStr := r.URL.Query().Get("completed_date"); completedDateStr != "" {
		if t, err := time.Parse("2006-01-02", completedDateStr); err == nil {
			completedDate = &t
		}
	}

	userTimezone, err := s.GetUserTimezone(userID)
	if err != nil {
		userTimezone = "UTC" // Fallback to UTC on error
	}
	tasks, total, err := services.GetTasksPaginated(s.GetDB(), userID, limit, offset, includeCompleted, cardID, priority, scheduledDate, completedDate, status, userTimezone)
	if err != nil {
		log.Printf("Error querying tasks for user %d: %v", userID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load tags for each task (keeping existing N+1 pattern for now)
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
	}

	// Convert task times to user's timezone
	if userTimezone == "" {
		userTimezone = "UTC"
	}
	for i := range tasks {
		services.ConvertTaskTimesToUserTimezone(&tasks[i], userTimezone)
	}

	response := models.TasksResponse{
		Tasks:  tasks,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTask applies a task update on db (the caller's transaction) and
// returns the id of a recurring task created by the update, if any. Tag
// re-derivation from the title is the CALLER's job after commit (it derives
// separate rows with their own emits; bead Zettelgarden-5ry keeps the primary
// task write + emit atomic in the caller's tx).
func (s *Handler) UpdateTask(db models.Database, userID int, id int, task models.Task) (int, error) {
	recurringTaskID, err := services.UpdateTask(db, userID, id, task)
	if err != nil {
		return 0, err
	}
	return recurringTaskID, nil
}

func (s *Handler) UpdateTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid task id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		log.Printf("Error decoding task update request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Normalize empty priority to nil to avoid storing ""
	if task.Priority != nil && *task.Priority == "" {
		task.Priority = nil
	}

	// If marking task as complete, validate subtasks first
	if task.IsComplete {
		// Check for force parameter
		force := r.URL.Query().Get("force") == "true"

		// Load existing task to get subtasks
		existingTask, err := s.QueryTask(userID, id)
		if err != nil {
			log.Printf("Error getting task for subtask validation: %v", err)
			// Continue without validation if we can't load the task
		} else {
			// Load subtasks
			existingTask.Subtasks, _ = services.GetSubtasks(s.GetDB(), userID, id)

			// Validate completion
			if validationErr := services.ValidateTaskCompletion(&existingTask, force); validationErr != nil {
				if incompleteErr, ok := validationErr.(*services.IncompleteSubtaskError); ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":            "incomplete_subtasks",
						"message":          validationErr.Error(),
						"incomplete_count": incompleteErr.IncompleteCount,
						"total_subtasks":   incompleteErr.TotalSubtasks,
					})
					return
				}
				http.Error(w, validationErr.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	// Note: Frontend sends times as ISO 8601 UTC strings (toISOString()),
	// so JSON parsing already correctly sets them as UTC. No conversion needed.

	// Task UPDATE + sync_log emit in one transaction (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	recurringTaskID, err := s.UpdateTask(tx, userID, id, task)
	if err != nil {
		s.rollbackTx(tx)
		log.Printf("Error updating task %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("update task: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	// Post-commit tag re-derivation (derived rows with their own emits).
	s.AddTagsFromTask(userID, id)
	if recurringTaskID > 0 {
		s.AddTagsFromTask(userID, recurringTaskID)
	}

	response := models.GenericResponse{
		Message: "success",
		Error:   false,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreateTask inserts a task on db (the caller's transaction) and returns its
// id. Tag re-derivation from the title is the CALLER's job after commit (see
// UpdateTask; bead Zettelgarden-5ry).
func (s *Handler) CreateTask(db models.Database, task models.Task) (int, error) {
	return services.CreateTask(db, task)
}

func (s *Handler) CreateTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		log.Printf("Error decoding create task request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Normalize empty priority to nil to avoid storing ""
	if task.Priority != nil && *task.Priority == "" {
		task.Priority = nil
	}

	// Ensure the user ID is set correctly
	task.UserID = userID

	// Note: Frontend sends times as ISO 8601 UTC strings (toISOString()),
	// so JSON parsing already correctly sets them as UTC. No conversion needed.

	// Task INSERT + sync_log emit in one transaction (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	taskID, err := s.CreateTask(tx, task)
	if err != nil {
		s.rollbackTx(tx)
		log.Printf("Error creating task: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("create task: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	// Post-commit tag re-derivation.
	s.AddTagsFromTask(task.UserID, taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": taskID})
}

// DeleteTask soft-deletes a task on db (the caller's transaction).
func (s *Handler) DeleteTask(db models.Database, userID int, id int) error {
	return services.DeleteTask(db, userID, id)
}

func (s *Handler) DeleteTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid task id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Task DELETE + sync_log emit in one transaction (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	err = s.DeleteTask(tx, userID, id)
	if err != nil {
		s.rollbackTx(tx)
		log.Printf("Error deleting task %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("delete task: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Handler) GetTaskAuditEventsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	taskID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify the user owns this task
	_, err = s.QueryTask(userID, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	events, err := services.GetAuditEvents(s.GetDB(), "task", taskID)
	if err != nil {
		log.Printf("Error getting audit events: %v", err)
		http.Error(w, "Error retrieving audit events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// AddTaskDependencyRoute adds a blocking task dependency
func (s *Handler) AddTaskDependencyRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	taskID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Parse request body to get blocking_task_id
	var requestBody struct {
		BlockingTaskID int `json:"blocking_task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Verify both tasks exist and belong to the user
	_, err = s.QueryTask(userID, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	_, err = s.QueryTask(userID, requestBody.BlockingTaskID)
	if err != nil {
		http.Error(w, "Blocking task not found", http.StatusNotFound)
		return
	}

	// Prevent self-blocking
	if taskID == requestBody.BlockingTaskID {
		http.Error(w, "A task cannot block itself", http.StatusBadRequest)
		return
	}

	// Insert the dependency
	_, err = s.GetDB().Exec(`
		INSERT INTO task_dependencies (task_id, blocking_task_id)
		VALUES ($1, $2)
		ON CONFLICT (task_id, blocking_task_id) DO NOTHING
	`, taskID, requestBody.BlockingTaskID)

	if err != nil {
		log.Printf("Error adding task dependency: %v", err)
		http.Error(w, "Error adding dependency", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GenericResponse{
		Message: "Dependency added successfully",
		Error:   false,
	})
}

// RemoveTaskDependencyRoute removes a blocking task dependency
func (s *Handler) RemoveTaskDependencyRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	taskID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	blockingTaskID, err := strconv.Atoi(mux.Vars(r)["blocking_id"])
	if err != nil {
		http.Error(w, "Invalid blocking task ID", http.StatusBadRequest)
		return
	}

	// Verify the task belongs to the user
	_, err = s.QueryTask(userID, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Remove the dependency
	result, err := s.GetDB().Exec(`
		DELETE FROM task_dependencies
		WHERE task_id = $1 AND blocking_task_id = $2
	`, taskID, blockingTaskID)

	if err != nil {
		log.Printf("Error removing task dependency: %v", err)
		http.Error(w, "Error removing dependency", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Dependency not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserTimezone returns the timezone string for a user
func (s *Handler) GetUserTimezone(userID int) (string, error) {
	var timezone string
	err := s.GetDB().QueryRow("SELECT timezone FROM users WHERE id = $1", userID).Scan(&timezone)
	if err != nil {
		log.Printf("Failed to get timezone for user %d: %v", userID, err)
		return "UTC", err // Return UTC as fallback on error
	}

	if timezone == "" {
		timezone = "UTC" // Default fallback
	}

	return timezone, nil
}

// CompleteAndScheduleTaskRoute handles completing a task and creating a new one scheduled for X days later
func (s *Handler) CompleteAndScheduleTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid task id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Parse request body to get days parameter
	var requestBody struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.Printf("Error decoding complete and schedule request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate days parameter
	if requestBody.Days <= 0 {
		http.Error(w, "Days must be greater than 0", http.StatusBadRequest)
		return
	}

	// Get the complete and default status names
	completeStatus, err := services.GetCompleteTaskStatus(s.GetDB(), userID)
	if err != nil {
		log.Printf("Error getting complete status: %v", err)
		http.Error(w, "Error getting complete status", http.StatusInternalServerError)
		return
	}

	defaultStatus, err := services.GetDefaultTaskStatus(s.GetDB(), userID)
	if err != nil {
		log.Printf("Error getting default status: %v", err)
		http.Error(w, "Error getting default status", http.StatusInternalServerError)
		return
	}

	// Call the service to complete the task and create a new one (both the
	// completion UPDATE + emit and the new task INSERT + emit in one tx, 5ry)
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	newTaskID, err := services.CompleteAndScheduleTask(tx, userID, id, requestBody.Days, completeStatus.Name, defaultStatus.Name)
	if err != nil {
		s.rollbackTx(tx)
		log.Printf("Error completing and scheduling task %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("complete and schedule task: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	// Post-commit tag re-derivation for the new task.
	if newTaskID > 0 {
		s.AddTagsFromTask(userID, newTaskID)
	}

	response := models.GenericResponse{
		Message: "success",
		Error:   false,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ===== Subtask Handlers =====

// CreateSubtaskRoute handles POST /api/tasks/:id/subtasks
// Creates a new task as a subtask of the specified parent
func (s *Handler) CreateSubtaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get parent task ID from URL
	vars := mux.Vars(r)
	parentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid parent task ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var input models.Task
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding create subtask request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Normalize empty priority to nil
	if input.Priority != nil && *input.Priority == "" {
		input.Priority = nil
	}

	// Get parent task
	parent, err := s.QueryTask(userID, parentID)
	if err != nil {
		http.Error(w, "Parent task not found", http.StatusNotFound)
		return
	}

	// Prepare subtask with inheritance
	subtask := services.PrepareSubtask(&parent, input)

	// Create the subtask (INSERT + sync_log emit in one tx, then derived tags).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	taskID, err := s.CreateTask(tx, subtask)
	if err != nil {
		s.rollbackTx(tx)
		log.Printf("Error creating subtask: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("create subtask: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}
	s.AddTagsFromTask(subtask.UserID, taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": taskID})
}

// SetTaskParentRoute handles PATCH /api/tasks/:id/parent
// Sets or clears the parent of a task
func (s *Handler) SetTaskParentRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get task ID from URL
	vars := mux.Vars(r)
	taskID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var input struct {
		ParentTaskID *int `json:"parent_task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Get the task
	task, err := s.QueryTask(userID, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Load subtasks for validation (if task is a parent)
	task.Subtasks, _ = services.GetSubtasks(s.GetDB(), userID, taskID)

	// If setting a parent (not clearing)
	if input.ParentTaskID != nil {
		// Get parent task
		parent, err := s.QueryTask(userID, *input.ParentTaskID)
		if err != nil {
			http.Error(w, "Parent task not found", http.StatusNotFound)
			return
		}

		// Validate the assignment
		if err := services.ValidateParentAssignment(&task, &parent); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Update the task's parent (UPDATE + sync_log emit in one tx, 5ry)
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	if err := services.UpdateTaskParent(tx, userID, taskID, input.ParentTaskID); err != nil {
		s.rollbackTx(tx)
		log.Printf("Error updating task parent: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("update task parent: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	// Return updated task
	updatedTask, _ := s.QueryTask(userID, taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTask)
}

// GetSubtasksRoute handles GET /api/tasks/:id/subtasks
// Returns all subtasks for a parent task
func (s *Handler) GetSubtasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	vars := mux.Vars(r)
	parentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify parent task exists and belongs to user
	_, err = s.QueryTask(userID, parentID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	subtasks, err := services.GetSubtasks(s.GetDB(), userID, parentID)
	if err != nil {
		log.Printf("Error getting subtasks: %v", err)
		http.Error(w, "Failed to get subtasks", http.StatusInternalServerError)
		return
	}

	// Count complete
	completeCount := 0
	for _, s := range subtasks {
		if s.IsComplete {
			completeCount++
		}
	}

	// Load tags for each subtask
	for i := range subtasks {
		tags, err := s.QueryTagsForTask(userID, subtasks[i].ID)
		if err == nil {
			subtasks[i].Tags = tags
		}
	}

	response := map[string]interface{}{
		"subtasks":       subtasks,
		"total":          len(subtasks),
		"complete_count": completeCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ReorderTasksRoute handles PUT /api/tasks/reorder
// Accepts a batch of { id, sort_order } updates
func (s *Handler) ReorderTasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var payload struct {
		Orders []struct {
			ID        int `json:"id"`
			SortOrder int `json:"sort_order"`
		} `json:"orders"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding reorder request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if len(payload.Orders) == 0 {
		http.Error(w, "No orders provided", http.StatusBadRequest)
		return
	}

	// Reorder is already transactional via services.ReorderTasks on the passed
	// handle; wrap it so every sort_order UPDATE + its sync_log emit commit
	// atomically in production (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	if err := services.ReorderTasks(tx, userID, payload.Orders); err != nil {
		s.rollbackTx(tx)
		log.Printf("Error reordering tasks: %v", err)
		http.Error(w, "Failed to reorder tasks", http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("reorder tasks: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "success",
		"error":   false,
	})
}
