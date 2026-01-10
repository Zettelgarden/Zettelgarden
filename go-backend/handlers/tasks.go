package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func (s *Handler) QueryTask(userID int, id int) (models.Task, error) {
	task, err := services.GetTask(s.DB, userID, id)
	if err != nil {
		return models.Task{}, err
	}

	// Load tags for the task
	tags, err := s.QueryTagsForTask(userID, task.ID)
	if err == nil {
		task.Tags = tags
	}

	return task, nil
}

func (s *Handler) QueryTasks(userID int, includeCompleted bool) ([]models.Task, error) {
	tasks, err := services.GetTasks(s.DB, userID, includeCompleted)
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

	return tasks, nil
}

func (s *Handler) QueryTasksPaginated(userID int, limit, offset int, includeCompleted bool, cardID *int, priority *string) ([]models.Task, int, error) {
	tasks, total, err := services.GetTasksPaginated(s.DB, userID, limit, offset, includeCompleted, cardID, priority, nil, nil, nil)
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

	return tasks, total, nil
}
func (s *Handler) QueryTasksByCard(userID int, cardPK int) ([]models.Task, error) {
	tasks, err := services.GetTasksByCard(s.DB, userID, cardPK)
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

	return tasks, nil
}

func (s *Handler) GetTaskRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	task, err := s.QueryTask(userID, id)
	if err != nil {
		log.Printf("asdas")
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

	tasks, total, err := services.GetTasksPaginated(s.DB, userID, limit, offset, includeCompleted, cardID, priority, scheduledDate, completedDate, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Batch load tags for all tasks
	taskIDs := make([]int, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}

	// Load tags for each task (keeping existing N+1 pattern for now)
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
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

func (s *Handler) UpdateTask(userID int, id int, task models.Task) error {
	recurringTaskID, err := services.UpdateTask(s.DB, userID, id, task)
	if err != nil {
		return err
	}

	// Add tags after updating the task
	s.AddTagsFromTask(userID, id)

	// If a recurring task was created, process its tags too
	if recurringTaskID > 0 {
		s.AddTagsFromTask(userID, recurringTaskID)
	}

	return nil
}

func (s *Handler) UpdateTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// First read the request body into a map to extract the priority
	var requestData map[string]interface{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Create a new reader with the same body data for the next decode
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		log.Printf("error unmarshaling request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Now decode into the Task struct
	var task models.Task
	if err := json.Unmarshal(bodyBytes, &task); err != nil {
		log.Printf("error unmarshaling to task: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Handle priority conversion from string to *string
	if priorityVal, ok := requestData["priority"]; ok {
		if priorityStr, ok := priorityVal.(string); ok && priorityStr != "" {
			task.Priority = &priorityStr
			log.Printf("Set priority from request: %s", priorityStr)
		} else {
			task.Priority = nil
			log.Printf("Priority was empty or not a string, setting to nil")
		}
	}

	err = s.UpdateTask(userID, id, task)
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	response := models.GenericResponse{
		Message: "success",
		Error:   false,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Handler) CreateTask(task models.Task) (int, error) {
	taskID, err := services.CreateTask(s.DB, task)
	if err != nil {
		return 0, err
	}

	// Add tags after creating the task
	s.AddTagsFromTask(task.UserID, taskID)
	return taskID, nil
}

func (s *Handler) CreateTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// First read the request body into a map to extract the priority
	var requestData map[string]interface{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Create a new reader with the same body data for the next decode
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		log.Printf("error unmarshaling request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Now decode into the Task struct
	var task models.Task
	if err := json.Unmarshal(bodyBytes, &task); err != nil {
		log.Printf("error unmarshaling to task: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Handle priority conversion from string to *string
	if priorityVal, ok := requestData["priority"]; ok {
		if priorityStr, ok := priorityVal.(string); ok && priorityStr != "" {
			task.Priority = &priorityStr
			log.Printf("Set priority from request: %s", priorityStr)
		} else {
			task.Priority = nil
			log.Printf("Priority was empty or not a string, setting to nil")
		}
	}

	log.Printf("creating task with priority: %v", task.Priority)
	// Ensure the user ID is set correctly
	task.UserID = userID

	taskID, err := s.CreateTask(task)
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": taskID})
}

func (s *Handler) DeleteTask(userID int, id int) error {
	return services.DeleteTask(s.DB, userID, id)
}

func (s *Handler) DeleteTaskRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	err = s.DeleteTask(userID, id)
	if err != nil {
		log.Printf("error %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
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

	events, err := services.GetAuditEvents(s.DB, "task", taskID)
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
	_, err = s.DB.Exec(`
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
	result, err := s.DB.Exec(`
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
