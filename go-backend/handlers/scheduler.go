package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// SchedulerAPI interface for testability
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]JobRunResponse, error)
}

// JobRunResponse is the DTO for job run history
type JobRunResponse struct {
	ID           int64  `json:"id"`
	JobName      string `json:"job_name"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	RetryCount   int    `json:"retry_count"`
}

// ListScheduledJobs returns a handler that lists all registered scheduled jobs
func ListScheduledJobs(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobs := scheduler.ListJobs()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string][]string{
			"jobs": jobs,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// GetJobHistory returns a handler that gets execution history for a specific job
func GetJobHistory(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		jobName := vars["jobName"]

		// Parse limit from query params, default to 50
		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		history, err := scheduler.GetJobHistory(r.Context(), jobName, limit)
		if err != nil {
			http.Error(w, "Failed to get job history", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(history); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
