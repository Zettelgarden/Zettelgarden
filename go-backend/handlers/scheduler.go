package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-backend/services"

	"github.com/gorilla/mux"
)

// SchedulerAPI interface for testability
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error)
	GetJobInfo(name string) (schedule string, nextRun time.Time, err error)
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

// ScheduledJobInfo represents a scheduled job with its configuration
type ScheduledJobInfo struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	NextRun  string `json:"next_run"`
}

// ScheduledJobsResponse is the response for listing scheduled jobs
type ScheduledJobsResponse struct {
	Jobs []ScheduledJobInfo `json:"jobs"`
}

// ListScheduledJobs returns a handler that lists all registered scheduled jobs with their schedules
func ListScheduledJobs(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobNames := scheduler.ListJobs()

		jobs := make([]ScheduledJobInfo, 0, len(jobNames))

		for _, name := range jobNames {
			schedule, nextRun, err := scheduler.GetJobInfo(name)
			if err != nil {
				// Log but continue - skip jobs with errors
				log.Printf("Error getting info for job '%s': %v", name, err)
				continue
			}

			var nextRunStr string
			if !nextRun.IsZero() {
				nextRunStr = nextRun.Format(time.RFC3339)
			}

			jobs = append(jobs, ScheduledJobInfo{
				Name:     name,
				Schedule: schedule,
				NextRun:  nextRunStr,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := ScheduledJobsResponse{Jobs: jobs}

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
