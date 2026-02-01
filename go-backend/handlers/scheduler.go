package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go-backend/services"

	"github.com/gorilla/mux"
)

// SchedulerAPI interface for testability
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error)
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

// SchedulerHealth returns the current health status of the scheduler
type SchedulerHealth struct {
	Running bool     `json:"running"`
	Jobs    []string `json:"jobs"`
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

// GetSchedulerHealth returns scheduler health information
func GetSchedulerHealth(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobs := scheduler.ListJobs()

		health := SchedulerHealth{
			Running: len(jobs) > 0, // Simple check
			Jobs:    jobs,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
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

		runs, err := scheduler.GetJobHistory(r.Context(), jobName, limit)
		if err != nil {
			http.Error(w, "Failed to get job history", http.StatusInternalServerError)
			return
		}

		// Convert services.JobRun to JobRunResponse
		responses := convertJobRunsToResponses(runs)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// convertJobRunsToResponses converts services.JobRun to JobRunResponse DTOs
func convertJobRunsToResponses(runs []services.JobRun) []JobRunResponse {
	responses := make([]JobRunResponse, len(runs))
	for i, run := range runs {
		responses[i] = JobRunResponse{
			ID:           run.ID,
			JobName:      run.JobName,
			StartedAt:    run.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			Status:       run.Status,
			ErrorMessage: run.ErrorMessage,
			RetryCount:   run.RetryCount,
		}
		// Only include CompletedAt for completed jobs
		if !run.CompletedAt.IsZero() {
			completedAt := run.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			responses[i].CompletedAt = completedAt
		}
	}
	return responses
}
