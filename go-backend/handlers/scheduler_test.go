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
}

func (m *mockSchedulerForHandler) ListJobs() []string {
	return m.jobs
}

func (m *mockSchedulerForHandler) GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error) {
	return m.history, nil
}

func TestListScheduledJobsHandler(t *testing.T) {
	tests := []struct {
		name           string
		jobs           []string
		expectedStatus int
		expectedJobs   []string
	}{
		{
			name:           "returns empty list when no jobs",
			jobs:           []string{},
			expectedStatus: http.StatusOK,
			expectedJobs:   []string{},
		},
		{
			name:           "returns list of registered jobs",
			jobs:           []string{"daily-cleanup", "hourly-sync", "weekly-report"},
			expectedStatus: http.StatusOK,
			expectedJobs:   []string{"daily-cleanup", "hourly-sync", "weekly-report"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSchedulerForHandler{jobs: tt.jobs}
			handler := ListScheduledJobs(mock)

			req := httptest.NewRequest("GET", "/admin/scheduler/jobs", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string][]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			jobs, ok := response["jobs"]
			if !ok {
				t.Fatal("response missing 'jobs' field")
			}

			if len(jobs) != len(tt.expectedJobs) {
				t.Errorf("expected %d jobs, got %d", len(tt.expectedJobs), len(jobs))
			}

			for i, job := range jobs {
				if job != tt.expectedJobs[i] {
					t.Errorf("expected job %q at index %d, got %q", tt.expectedJobs[i], i, job)
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
			mock := &mockSchedulerForHandler{history: tt.history}
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
