package admin

import (
	"encoding/json"
	"fmt"
	"go-backend/handlers"
	"go-backend/models"
	"go-backend/services"
	"net/http"

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
