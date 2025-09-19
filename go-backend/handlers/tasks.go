package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func (s *Handler) QueryTask(userID int, id int) (models.Task, error) {
	var task models.Task

	err := s.DB.QueryRow(`
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, priority, is_complete
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
		&task.Priority,
		&task.IsComplete,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.Task{}, fmt.Errorf("unable to access task")
	}
	if task.CardPK > 0 {
		card, err := s.QueryPartialCardByID(userID, task.CardPK)
		if err != nil {
			task.Card = card
		}
	}
	return task, nil
}

func (s *Handler) QueryTasks(userID int, includeCompleted bool) ([]models.Task, error) {
	var tasks []models.Task
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, priority, is_complete
	FROM
	tasks
	WHERE user_id = $1 AND is_deleted = FALSE
	`
	if !includeCompleted {
		query += " AND is_complete = FALSE"
	}

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		log.Printf("err %v", err)
		return []models.Task{}, err
	}
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
			log.Printf("err %v", err)
			return []models.Task{}, fmt.Errorf("unable to access task")
		}
		if task.CardPK > 0 {
			card, err := s.QueryPartialCardByID(userID, task.CardPK)
			if err == nil {
				task.Card = card
			}
		}
		tags, err := s.QueryTagsForTask(userID, task.ID)
		if err == nil {
			task.Tags = tags
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
func (s *Handler) QueryTasksByCard(userID int, cardPK int) ([]models.Task, error) {
	var tasks []models.Task
	query := `
	SELECT id, card_pk, user_id, scheduled_date, due_date,
	created_at, updated_at, completed_at, title, priority, is_complete
	FROM
	tasks
	WHERE user_id = $1 AND is_deleted = FALSE AND card_pk = $2
`
	rows, err := s.DB.Query(query, userID, cardPK)
	if err != nil {
		log.Printf("err %v", err)
		return []models.Task{}, err
	}
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
			log.Printf("err %v", err)
			return []models.Task{}, fmt.Errorf("unable to access task")
		}
		if task.CardPK > 0 {
			card, err := s.QueryPartialCardByID(userID, task.CardPK)
			if err == nil {
				task.Card = card
			}
		}
		tags, err := s.QueryTagsForTask(userID, task.ID)
		if err == nil {
			task.Tags = tags
		}
		tasks = append(tasks, task)
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

	completed := r.URL.Query().Get("completed")
	includeCompleted := false
	if completed == "true" {
		includeCompleted = true
	}

	tasks, err := s.QueryTasks(userID, includeCompleted)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)

}

func (s *Handler) UpdateTask(userID int, id int, task models.Task) error {
	oldTask, err := s.QueryTask(userID, id)
	if err != nil {
		return fmt.Errorf("unable to query task: %v", err)
	}

	var completedAt *time.Time
	if task.IsComplete && !oldTask.IsComplete {
		now := time.Now()
		completedAt = &now
		err = s.checkRecurringTasks(task)
		if err != nil {
			log.Printf("err %v", err)
		}
	} else if oldTask.IsComplete {
		completedAt = oldTask.CompletedAt
	} else {
		completedAt = nil
	}

	_, err = s.DB.Exec(`
		UPDATE tasks SET
			card_pk = $1,
			scheduled_date = $2,
			updated_at = NOW(),
			completed_at = $3,
			title = $4,
			priority = $5,
			is_complete = $6
		WHERE id = $7 AND user_id = $8 AND is_deleted = FALSE
	`, task.CardPK, task.ScheduledDate, completedAt, task.Title, task.Priority, task.IsComplete, id, userID)

	if err != nil {
		log.Printf("error: %v", err)
		return fmt.Errorf("unable to update task")
	}

	newTask, err := s.QueryTask(userID, id)
	if err != nil {
		log.Printf("Error querying updated task for audit: %v", err)
	} else {
		err = services.CreateAuditEvent(s.DB, userID, id, "task", "update", oldTask, newTask)
		if err != nil {
			log.Printf("Error creating audit event: %v", err)
		}
	}

	s.AddTagsFromTask(userID, id)
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
	var taskID int

	// Log the priority value for debugging
	if task.Priority != nil {
		log.Printf("Priority value: %s", *task.Priority)
	} else {
		log.Printf("Priority is nil")
	}

	err := s.DB.QueryRow(`
	INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, priority, is_complete, is_deleted)
	VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7, $8, FALSE)
	RETURNING id
	`, task.CardPK, task.UserID, task.ScheduledDate, task.DueDate, task.CompletedAt, task.Title, task.Priority, task.IsComplete).Scan(&taskID)

	if err != nil {
		log.Printf("err %v", err)
		return 0, fmt.Errorf("unable to create task")
	}

	newTask, err := s.QueryTask(task.UserID, taskID)
	if err != nil {
		log.Printf("Error querying new task for audit: %v", err)
	} else {
		err = services.CreateAuditEvent(s.DB, task.UserID, taskID, "task", "create", nil, newTask)
		if err != nil {
			log.Printf("Error creating audit event: %v", err)
		}
	}

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
	oldTask, err := s.QueryTask(userID, id)
	if err != nil {
		return fmt.Errorf("unable to query task: %v", err)
	}

	_, err = s.DB.Exec(`
	UPDATE tasks SET is_deleted = TRUE
	WHERE id = $1 AND user_id = $2
	`, id, userID)

	if err != nil {
		log.Printf("err %v", err)
		return fmt.Errorf("unable to delete task")
	}

	err = services.CreateAuditEvent(s.DB, userID, id, "task", "delete", oldTask, nil)
	if err != nil {
		log.Printf("Error creating audit event: %v", err)
	}

	return nil
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

func parseRecurringTasks(title string) (models.RecurringTask, bool) {
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
func (s *Handler) checkRecurringTasks(task models.Task) error {
	recurringTask, found := parseRecurringTasks(task.Title)
	if !found {
		return nil
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
		Priority:      task.Priority,
		IsComplete:    false,
	}
	_, err := s.CreateTask(newTask)
	if err != nil {
		return err
	}
	return nil

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
