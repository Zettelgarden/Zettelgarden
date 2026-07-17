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
	GetJobHistory(ctx context.Context, jobName string, limit int, offset int) ([]services.JobRun, int, error)
	GetJobInfo(name string) (schedule string, nextRun time.Time, err error)
	GetJobSummary(ctx context.Context, jobName string) (services.ServiceJobSummary, error)
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

// JobSummary represents summary statistics for a scheduled job
type JobSummary struct {
	JobName       string   `json:"job_name"`
	LastRunStatus string   `json:"last_run_status"`
	LastRunAt     *string  `json:"last_run_at,omitempty"`
	RecentStats   JobStats `json:"recent_stats"`
}

// JobStats represents statistics for job runs
type JobStats struct {
	TotalRuns    int     `json:"total_runs"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"`
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

		// Parse offset from query params, default to 0
		offset := 0
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		runs, total, err := scheduler.GetJobHistory(r.Context(), jobName, limit, offset)
		if err != nil {
			http.Error(w, "Failed to get job history", http.StatusInternalServerError)
			return
		}

		// Calculate has_more: true if we got a full page, which means there might be more results
		// This is the standard pattern for cursor/pagination APIs
		hasMore := len(runs) == limit

		// Convert services.JobRun to JobRunResponse
		responses := convertJobRunsToResponses(runs)

		response := map[string]interface{}{
			"runs":     responses,
			"total":    total,
			"offset":   offset,
			"limit":    limit,
			"has_more": hasMore,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
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

// GetJobSummaryHandler returns a handler that gets summary statistics for a job
func GetJobSummaryHandler(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		jobName := vars["jobName"]

		summary, err := scheduler.GetJobSummary(r.Context(), jobName)
		if err != nil {
			http.Error(w, "Failed to get job summary", http.StatusInternalServerError)
			return
		}

		// Convert services.ServiceJobSummary to handler JobSummary DTO
		response := JobSummary{
			JobName:       summary.JobName,
			LastRunStatus: summary.LastRunStatus,
			RecentStats: JobStats{
				TotalRuns:    summary.RecentStats.TotalRuns,
				SuccessCount: summary.RecentStats.SuccessCount,
				FailureCount: summary.RecentStats.FailureCount,
				SuccessRate:  summary.RecentStats.SuccessRate,
			},
		}

		if summary.LastRunAt != nil {
			lastRunAt := summary.LastRunAt.Format("2006-01-02T15:04:05Z07:00")
			response.LastRunAt = &lastRunAt
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
