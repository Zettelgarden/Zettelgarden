package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CreateJobRequest represents the request to create a new job
type CreateJobRequest struct {
	JobType     string                 `json:"job_type"`
	Priority    int                    `json:"priority,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
	MaxRetries  int                    `json:"max_retries,omitempty"`
	TimeoutSecs int                    `json:"timeout_seconds,omitempty"`
}

// JobResponse represents the response for a job
type JobResponse struct {
	ID            int                    `json:"id"`
	UserID        int                    `json:"user_id"`
	JobType       string                 `json:"job_type"`
	Status        string                 `json:"status"`
	Priority      int                    `json:"priority"`
	Payload       map[string]interface{} `json:"payload"`
	Result        map[string]interface{} `json:"result,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	StartedAt     string                 `json:"started_at,omitempty"`
	CompletedAt   string                 `json:"completed_at,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	TimeoutSecs   int                    `json:"timeout_seconds"`
}

// JobsListResponse represents a paginated list of jobs
type JobsListResponse struct {
	Jobs       []JobResponse `json:"jobs"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

// CreateJobRoute creates a new job and adds it to the queue
func (h *Handler) CreateJobRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request
	var req CreateJobRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate job type
	if !isValidJobType(req.JobType) {
		http.Error(w, fmt.Sprintf("Invalid job_type: %s. Must be one of: embedding, summarization, entity_extraction, chat, memory, email", req.JobType), http.StatusBadRequest)
		return
	}

	// Validate payload
	if len(req.Payload) == 0 {
		http.Error(w, "payload cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate payload based on job type
	if err := validateJobPayload(req.JobType, req.Payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check rate limits if rate limiter is initialized
	if h.JobRateLimiter != nil {
		// Check if user is PRO (has active or trialing subscription)
		user, err := h.QueryUser(userID)
		if err != nil {
			log.Printf("Failed to query user for rate limit check: %v", err)
			http.Error(w, "Failed to check rate limits", http.StatusInternalServerError)
			return
		}

		isProUser := user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing"

		result := h.JobRateLimiter.CheckRateLimit(ctx, userID, isProUser)
		services.SetJobRateLimitHeaders(w, result)

		if !result.Allowed {
			log.Printf("[RateLimiter] Job submission rejected for user %d: %s", userID, result.Reason)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "rate_limit_exceeded",
				"message": result.Reason,
			})
			return
		}

		// Record the job submission
		h.JobRateLimiter.RecordJobSubmission(userID, isProUser)
	}

	// Create job parameters
	params := models.CreateJobParams{
		UserID:      userID,
		JobType:     models.JobType(req.JobType),
		Priority:    req.Priority,
		Payload:     req.Payload,
		MaxRetries:  req.MaxRetries,
		TimeoutSecs: req.TimeoutSecs,
	}

	// Create job via queue
	queue := services.NewJobQueue(h.DB)
	job, err := queue.Enqueue(ctx, params)
	if err != nil {
		log.Printf("Failed to create job: %v", err)

		// Rollback the submission count on error
		if h.JobRateLimiter != nil {
			h.JobRateLimiter.RecordJobCompletion(userID)
		}

		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	// Return job response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(jobToResponse(job))
}

// GetJobRoute retrieves a job by ID
func (h *Handler) GetJobRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Get job
	queue := services.NewJobQueue(h.DB)
	job, err := queue.Get(r.Context(), jobID)
	if err != nil {
		log.Printf("Failed to get job: %v", err)
		http.Error(w, "Failed to get job", http.StatusInternalServerError)
		return
	}

	if job == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Check ownership
	if job.UserID != userID {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobToResponse(job))
}

// ListJobsRoute lists jobs for the current user
func (h *Handler) ListJobsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	// Set defaults
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Build params
	params := models.JobListParams{
		UserID: userID,
		Status: models.JobStatus(status),
		Limit:  perPage,
		Offset: offset,
	}

	// Get jobs
	queue := services.NewJobQueue(h.DB)
	jobs, err := queue.List(r.Context(), params)
	if err != nil {
		log.Printf("Failed to list jobs: %v", err)
		http.Error(w, "Failed to list jobs", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	var total int
	if status != "" {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM llm_jobs WHERE user_id = $1 AND status = $2", userID, status).Scan(&total)
	} else {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM llm_jobs WHERE user_id = $1", userID).Scan(&total)
	}
	if err != nil {
		log.Printf("Failed to get job count: %v", err)
		total = len(jobs)
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	// Convert to response format
	jobResponses := make([]JobResponse, 0, len(jobs))
	for _, job := range jobs {
		jobResponses = append(jobResponses, jobToResponse(&job))
	}

	response := JobsListResponse{
		Jobs:       jobResponses,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelJobRoute cancels a pending job
func (h *Handler) CancelJobRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Cancel job
	queue := services.NewJobQueue(h.DB)
	err = queue.Cancel(r.Context(), jobID, userID)
	if err != nil {
		if err.Error() == "job not found" || err.Error() == "sql: no rows in result set" {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to cancel job: %v", err)
		http.Error(w, "Failed to cancel job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Job cancelled successfully",
	})
}

// GetJobStatsRoute retrieves statistics for the current user's jobs
func (h *Handler) GetJobStatsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	queue := services.NewJobQueue(h.DB)
	stats, err := queue.Stats(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to get job stats: %v", err)
		http.Error(w, "Failed to get job stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper: isValidJobType checks if the job type is valid
func isValidJobType(jobType string) bool {
	validTypes := map[string]bool{
		"embedding":         true,
		"summarization":     true,
		"entity_extraction": true,
		"chat":              true,
		"memory":            true,
		"email":             true,
	}
	return validTypes[jobType]
}

// Helper: validateJobPayload validates the payload based on job type
func validateJobPayload(jobType string, payload map[string]interface{}) error {
	switch jobType {
	case "embedding":
		if _, ok := payload["card_pk"]; !ok {
			return fmt.Errorf("card_pk is required for embedding jobs")
		}
	case "entity_extraction":
		if _, ok := payload["card_pk"]; !ok {
			return fmt.Errorf("card_pk is required for entity extraction jobs")
		}
	case "memory":
		if _, ok := payload["memory_type"]; !ok {
			return fmt.Errorf("memory_type is required for memory jobs")
		}
		memoryType := payload["memory_type"].(string)
		if memoryType != "card" && memoryType != "chat" {
			return fmt.Errorf("memory_type must be 'card' or 'chat'")
		}
		if memoryType == "card" {
			if _, ok := payload["card_content"]; !ok {
				return fmt.Errorf("card_content is required for memory jobs with type 'card'")
			}
		} else {
			if _, ok := payload["user_message"]; !ok {
				return fmt.Errorf("user_message is required for memory jobs with type 'chat'")
			}
			if _, ok := payload["assistant_message"]; !ok {
				return fmt.Errorf("assistant_message is required for memory jobs with type 'chat'")
			}
		}
	case "summarization":
		if _, ok := payload["summarization_id"]; !ok {
			return fmt.Errorf("summarization_id is required for summarization jobs")
		}
	case "chat":
		if _, ok := payload["conversation_id"]; !ok {
			return fmt.Errorf("conversation_id is required for chat jobs")
		}
		if _, ok := payload["message"]; !ok {
			return fmt.Errorf("message is required for chat jobs")
		}
	case "email":
		if _, ok := payload["to"]; !ok {
			return fmt.Errorf("to is required for email jobs")
		}
		if _, ok := payload["subject"]; !ok {
			return fmt.Errorf("subject is required for email jobs")
		}
		if _, ok := payload["body"]; !ok {
			return fmt.Errorf("body is required for email jobs")
		}
	}
	return nil
}

// Helper: jobToResponse converts a models.LLMJob to JobResponse
func jobToResponse(job *models.LLMJob) JobResponse {
	response := JobResponse{
		ID:           job.ID,
		UserID:       job.UserID,
		JobType:      string(job.JobType),
		Status:       string(job.Status),
		Priority:     job.Priority,
		Payload:      job.Payload,
		Result:       job.Result,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		RetryCount:   job.RetryCount,
		MaxRetries:   job.MaxRetries,
		TimeoutSecs:  job.TimeoutSecs,
	}

	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return response
}
