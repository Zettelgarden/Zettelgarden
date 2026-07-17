package handlers

import (
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

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
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// JobsListResponse represents a paginated list of jobs
type JobsListResponse struct {
	Jobs       []JobResponse `json:"jobs"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

// GetJobRoute retrieves a job by ID.
//
// Jobs are no longer created or driven through an external queue: they are an
// audit record of LLM work executed inline by services.JobRunner. This
// endpoint (and List/Stats below) exists purely to surface that audit log.
func (h *Handler) GetJobRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := models.GetJob(h.DB, jobID)
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
	jobs, err := models.ListJobs(h.DB, params)
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

// GetJobStatsRoute retrieves statistics for the current user's jobs
func (h *Handler) GetJobStatsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	stats, err := models.GetJobStats(h.DB, userID)
	if err != nil {
		log.Printf("Failed to get job stats: %v", err)
		http.Error(w, "Failed to get job stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// jobToResponse converts a models.LLMJob to JobResponse
func jobToResponse(job *models.LLMJob) JobResponse {
	response := JobResponse{
		ID:            job.ID,
		UserID:        job.UserID,
		JobType:       string(job.JobType),
		Status:        string(job.Status),
		Priority:      job.Priority,
		Payload:       job.Payload,
		Result:        job.Result,
		ErrorMessage:  job.ErrorMessage,
		CreatedAt:     job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		RetryCount:    job.RetryCount,
		MaxRetries:    job.MaxRetries,
		TimeoutSecs:   job.TimeoutSecs,
		CorrelationID: job.CorrelationID,
	}

	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return response
}
