package admin

import (
	"encoding/json"
	"fmt"
	"go-backend/handlers"
	"go-backend/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetAllJobsRoute lists all jobs across all users (admin only).
//
// Jobs are now an audit record of inline-processed LLM work (see
// services.JobRunner), so this endpoint simply surfaces that log. There is no
// longer a queue to pause/resume or a worker pool to inspect.
//
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
		       created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds, correlation_id
		FROM llm_jobs
	`
	countQuery := "SELECT COUNT(*) FROM llm_jobs"
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		baseQuery += fmt.Sprintf(" WHERE status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx) + " OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, limit, offset)

	// Execute main query
	rows, err := h.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to query jobs")
		return
	}
	defer rows.Close()

	jobs, err := models.ScanLLMJobs(rows)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to scan jobs")
		return
	}

	// Get total count
	var total int
	if status != "" {
		err = h.DB.QueryRowContext(ctx, countQuery+" WHERE status = $1", status).Scan(&total)
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
		"jobs":   jobsWithUsers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// RetryJobRoute re-runs a failed or cancelled job inline via the JobRunner.
// POST /api/admin/jobs/{id}/retry
func RetryJobRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	if jobIDStr == "" {
		RespondWithError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	parsedJobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if h.JobRunner == nil {
		RespondWithError(w, http.StatusServiceUnavailable, "Job runner not initialized")
		return
	}

	job, err := h.JobRunner.Retry(r.Context(), parsedJobID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Job retried successfully",
		"job_id":  job.ID,
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
