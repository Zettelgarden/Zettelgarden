package admin

import (
	"encoding/json"
	"fmt"
	"go-backend/handlers"
	"go-backend/models"
	"go-backend/services"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// JobQueueHealthResponse represents the health status of the job queue system
type JobQueueHealthResponse struct {
	Running     bool   `json:"running"`
	Paused      bool   `json:"paused"`
	WorkerCount int    `json:"worker_count"`
	QueueDepth  int    `json:"queue_depth"`
	Stats       services.WorkerStats `json:"stats"`
}

// WorkerStatsResponse represents per-worker statistics
type WorkerStatsResponse struct {
	WorkerID    string             `json:"worker_id"`
	JobsProcessed int64            `json:"jobs_processed"`
	JobsSucceeded int64            `json:"jobs_succeeded"`
	JobsFailed    int64            `json:"jobs_failed"`
	JobsRetried   int64            `json:"jobs_retried"`
}

// WorkersStatsResponse represents the response for the workers stats endpoint
type WorkersStatsResponse struct {
	Workers []WorkerStatsResponse `json:"workers"`
	Total   services.WorkerStats  `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetJobQueueHealthRoute returns the health status of the job queue system
// GET /api/admin/jobs/health
func GetJobQueueHealthRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	if h.LLMWorkerPool == nil {
		RespondWithError(w, http.StatusServiceUnavailable, "Job queue worker pool not initialized")
		return
	}

	queueDepth, err := h.LLMWorkerPool.GetQueueDepth()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get queue depth")
		return
	}

	stats := h.LLMWorkerPool.Stats()

	response := JobQueueHealthResponse{
		Running:     h.LLMWorkerPool.IsRunning(),
		Paused:      h.LLMWorkerPool.IsPaused(),
		WorkerCount: h.LLMWorkerPool.WorkerCount(),
		QueueDepth:  queueDepth,
		Stats:       stats,
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// GetJobWorkersStatsRoute returns per-worker statistics
// GET /api/admin/jobs/workers
func GetJobWorkersStatsRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	if h.LLMWorkerPool == nil {
		RespondWithError(w, http.StatusServiceUnavailable, "Job queue worker pool not initialized")
		return
	}

	// Get individual worker stats from the pool
	// Note: We need to expose workers from the pool or aggregate stats differently
	// For now, we'll return the aggregated stats
	stats := h.LLMWorkerPool.Stats()

	response := WorkersStatsResponse{
		Workers: []WorkerStatsResponse{}, // TODO: Get per-worker stats if needed
		Total:   stats,
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// GetAllJobsRoute lists all jobs across all users (admin only)
// GET /api/admin/jobs?status={status}&limit={limit}&offset={offset}
func GetAllJobsRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	// Parse query parameters
	status := query.Get("status")
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Build query
	baseQuery := `
		SELECT id, user_id, job_type, status, priority, payload, result, error_message,
		       created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds
		FROM llm_jobs
	`
	countQuery := "SELECT COUNT(*) FROM llm_jobs"
	args := []interface{}{}
	argIdx := 1

	whereClause := ""
	whereConditions := []string{}

	// Filter by status if provided
	if status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if len(whereConditions) > 0 {
		whereClause = " WHERE " + fmt.Sprintf("status = $%d", 1)
	}

	// Add status filter to count query
	finalCountQuery := countQuery
	if status != "" {
		finalCountQuery += " WHERE status = $1"
	}

	// Add ordering and pagination
	baseQuery += whereClause + " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx) + " OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, limit, offset)

	// Execute main query
	rows, err := h.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to query jobs")
		return
	}
	defer rows.Close()

	// Scan jobs
	jobs := []models.LLMJob{}
	for rows.Next() {
		job, err := models.ScanLLMJobs(rows)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Failed to scan job")
			return
		}
		if len(job) > 0 {
			jobs = append(jobs, job[0])
		}
	}

	// Get total count
	var total int
	if status != "" {
		err = h.DB.QueryRowContext(ctx, finalCountQuery, status).Scan(&total)
	} else {
		err = h.DB.QueryRowContext(ctx, countQuery).Scan(&total)
	}
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to count jobs")
		return
	}

	// Get user IDs for batch lookup
	userIDs := make(map[int]bool)
	for _, job := range jobs {
		userIDs[job.UserID] = true
	}

	// Fetch usernames for all users
	userNames := make(map[int]string)
	for userID := range userIDs {
		var username string
		err := h.DB.QueryRowContext(ctx, "SELECT username FROM users WHERE id = $1", userID).Scan(&username)
		if err == nil {
			userNames[userID] = username
		}
	}

	// Enrich jobs with usernames
	type JobWithUser struct {
		models.LLMJob
		Username string `json:"username"`
	}

	jobsWithUsers := make([]JobWithUser, len(jobs))
	for i, job := range jobs {
		jobsWithUsers[i] = JobWithUser{
			LLMJob:   job,
			Username: userNames[job.UserID],
		}
	}

	response := map[string]interface{}{
		"jobs":  jobsWithUsers,
		"total": total,
		"limit": limit,
		"offset": offset,
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// RetryJobRoute retries a failed job
// POST /api/admin/jobs/{id}/retry
func RetryJobRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		RespondWithError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	ctx := r.Context()

	// Parse job ID
	parsedJobID, err := parseJobID(jobID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Get the job to verify it exists and is failed
	queue := services.NewJobQueue(h.DB)
	job, err := queue.Get(ctx, parsedJobID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Job not found")
		return
	}

	// Only retry failed or cancelled jobs
	if job.Status != models.JobStatusFailed && job.Status != models.JobStatusCancelled {
		RespondWithError(w, http.StatusBadRequest, "Only failed or cancelled jobs can be retried")
		return
	}

	// Reset job to pending status
	if err := queue.UpdateStatus(ctx, parsedJobID, models.JobStatusPending); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retry job")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Job retried successfully",
		"job_id":  parsedJobID,
	})
}

// PauseJobQueueRoute pauses job processing
// POST /api/admin/jobs/pause
func PauseJobQueueRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	if h.LLMWorkerPool == nil {
		RespondWithError(w, http.StatusServiceUnavailable, "Job queue worker pool not initialized")
		return
	}

	if err := h.LLMWorkerPool.Pause(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Job queue paused successfully",
	})
}

// ResumeJobQueueRoute resumes job processing
// POST /api/admin/jobs/resume
func ResumeJobQueueRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	if h.LLMWorkerPool == nil {
		RespondWithError(w, http.StatusServiceUnavailable, "Job queue worker pool not initialized")
		return
	}

	if err := h.LLMWorkerPool.Resume(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Job queue resumed successfully",
	})
}

// Helper functions

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, ErrorResponse{Error: message})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func parseJobID(id string) (int, error) {
	var jobID int
	_, err := fmt.Sscanf(id, "%d", &jobID)
	if err != nil {
		return 0, err
	}
	return jobID, nil
}
