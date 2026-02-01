package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-backend/services"

	"github.com/gorilla/mux"
)

// mockSchedulerForHandler is a mock implementation for testing
type mockSchedulerForHandler struct {
	jobs    []string
	history []services.JobRun
	// jobSchedules maps job name to cron schedule
	jobSchedules map[string]string
	// jobNextRuns maps job name to next run time
	jobNextRuns map[string]time.Time
}

func (m *mockSchedulerForHandler) ListJobs() []string {
	return m.jobs
}

func (m *mockSchedulerForHandler) GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error) {
	return m.history, nil
}

func (m *mockSchedulerForHandler) GetJobInfo(name string) (schedule string, nextRun time.Time, err error) {
	if m.jobSchedules == nil {
		return "", time.Time{}, nil
	}
	schedule = m.jobSchedules[name]
	nextRun = m.jobNextRuns[name]
	return schedule, nextRun, nil
}

func TestListScheduledJobsHandler(t *testing.T) {
	// Helper time for testing
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		jobs           []string
		jobSchedules   map[string]string
		jobNextRuns    map[string]time.Time
		expectedStatus int
		expectedJobs   []ScheduledJobInfo
	}{
		{
			name:           "returns empty list when no jobs",
			jobs:           []string{},
			jobSchedules:   map[string]string{},
			jobNextRuns:    map[string]time.Time{},
			expectedStatus: http.StatusOK,
			expectedJobs:   []ScheduledJobInfo{},
		},
		{
			name: "returns list of registered jobs with schedules",
			jobs: []string{"daily-cleanup", "hourly-sync", "weekly-report"},
			jobSchedules: map[string]string{
				"daily-cleanup":  "0 0 * * *",
				"hourly-sync":    "0 * * * *",
				"weekly-report":  "0 9 * * 1",
			},
			jobNextRuns: map[string]time.Time{
				"daily-cleanup": baseTime.Add(14 * time.Hour),
				"hourly-sync":   baseTime.Add(30 * time.Minute),
				"weekly-report": baseTime.Add(24 * time.Hour),
			},
			expectedStatus: http.StatusOK,
			expectedJobs: []ScheduledJobInfo{
				{Name: "daily-cleanup", Schedule: "0 0 * * *", NextRun: baseTime.Add(14 * time.Hour).Format(time.RFC3339)},
				{Name: "hourly-sync", Schedule: "0 * * * *", NextRun: baseTime.Add(30 * time.Minute).Format(time.RFC3339)},
				{Name: "weekly-report", Schedule: "0 9 * * 1", NextRun: baseTime.Add(24 * time.Hour).Format(time.RFC3339)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSchedulerForHandler{
				jobs:         tt.jobs,
				jobSchedules: tt.jobSchedules,
				jobNextRuns:  tt.jobNextRuns,
			}
			handler := ListScheduledJobs(mock)

			req := httptest.NewRequest("GET", "/admin/scheduler/jobs", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response ScheduledJobsResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(response.Jobs) != len(tt.expectedJobs) {
				t.Errorf("expected %d jobs, got %d", len(tt.expectedJobs), len(response.Jobs))
			}

			for i, job := range response.Jobs {
				if job.Name != tt.expectedJobs[i].Name {
					t.Errorf("expected job name %q at index %d, got %q", tt.expectedJobs[i].Name, i, job.Name)
				}
				if job.Schedule != tt.expectedJobs[i].Schedule {
					t.Errorf("expected schedule %q at index %d, got %q", tt.expectedJobs[i].Schedule, i, job.Schedule)
				}
				if job.NextRun != tt.expectedJobs[i].NextRun {
					t.Errorf("expected next_run %q at index %d, got %q", tt.expectedJobs[i].NextRun, i, job.NextRun)
				}
			}
		})
	}
}

func TestGetJobHistoryHandler(t *testing.T) {
	// Helper time for testing
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		jobName        string
		history        []services.JobRun
		queryLimit     string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:    "returns empty history for job with no runs",
			jobName: "daily-cleanup",
			history: []services.JobRun{},
			queryLimit: "",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:    "returns job history",
			jobName: "daily-cleanup",
			history: []services.JobRun{
				{
					ID:          1,
					JobName:     "daily-cleanup",
					StartedAt:   baseTime,
					CompletedAt: baseTime.Add(time.Minute),
					Status:      "completed",
					RetryCount:  0,
				},
				{
					ID:          2,
					JobName:     "daily-cleanup",
					StartedAt:   baseTime.Add(24 * time.Hour),
					CompletedAt: baseTime.Add(24*time.Hour + 90*time.Second),
					Status:      "completed",
					RetryCount:  0,
				},
			},
			queryLimit: "",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:    "returns job history with error",
			jobName: "daily-cleanup",
			history: []services.JobRun{
				{
					ID:           3,
					JobName:      "daily-cleanup",
					StartedAt:    baseTime.Add(2 * 24 * time.Hour),
					Status:       "failed",
					ErrorMessage: "connection timeout",
					RetryCount:   2,
				},
			},
			queryLimit: "",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:    "parses limit from query params",
			jobName: "daily-cleanup",
			history: []services.JobRun{
				{ID: 1, JobName: "daily-cleanup", StartedAt: baseTime, Status: "completed"},
				{ID: 2, JobName: "daily-cleanup", StartedAt: baseTime.Add(24 * time.Hour), Status: "completed"},
			},
			queryLimit: "10",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSchedulerForHandler{
				history:       tt.history,
				jobSchedules:  map[string]string{},
				jobNextRuns:   map[string]time.Time{},
			}
			handler := GetJobHistory(mock)

			url := "/admin/scheduler/jobs/" + tt.jobName + "/history"
			if tt.queryLimit != "" {
				url += "?limit=" + tt.queryLimit
			}

			req := httptest.NewRequest("GET", url, nil)
			// Set up gorilla/mux to handle path variables
			req = mux.SetURLVars(req, map[string]string{"jobName": tt.jobName})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response []JobRunResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(response) != tt.expectedCount {
				t.Errorf("expected %d history entries, got %d", tt.expectedCount, len(response))
			}

			for i, entry := range response {
				if entry.JobName != tt.jobName {
					t.Errorf("expected job_name %q, got %q", tt.jobName, entry.JobName)
				}
				if entry.ID != tt.history[i].ID {
					t.Errorf("expected ID %d, got %d", tt.history[i].ID, entry.ID)
				}
			}
		})
	}
}
