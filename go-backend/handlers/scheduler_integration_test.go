package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-backend/services"
	"go-backend/services/jobs"
	"go-backend/tests"

	"github.com/gorilla/mux"
)

// TestSchedulerIntegration is an end-to-end integration test for the scheduler system.
//
// This test verifies:
// 1. Setup test database with schema
// 2. Create scheduler with test DB
// 3. Register cleanup job
// 4. Start/stop scheduler
// 5. Verify list jobs HTTP endpoint
// 6. Verify execution tracking works
//
// To run this test:
//   cd go-backend && go test -v -run TestSchedulerIntegration ./handlers
func TestSchedulerIntegration(t *testing.T) {
	// 1. Setup test database with schema
	h := NewHandler()
	defer tests.Teardown()

	// 2. Create scheduler with test DB
	// Note: Use h.DB (not h.Server.Tx) because scheduler runs in background goroutines
	// and transactions don't work well across goroutines
	scheduler := services.NewScheduler(h.DB)

	// 3. Register cleanup job
	cleanupJob := jobs.NewCleanupJob(h.DB)
	err := scheduler.Register(cleanupJob)
	if err != nil {
		t.Fatalf("Failed to register cleanup job: %v", err)
	}

	// Verify job is registered
	jobs := scheduler.ListJobs()
	if len(jobs) != 1 {
		t.Errorf("Expected 1 registered job, got %d", len(jobs))
	}
	if jobs[0] != "daily-cleanup" {
		t.Errorf("Expected job name 'daily-cleanup', got '%s'", jobs[0])
	}

	// 4. Start/stop scheduler
	scheduler.Start()
	// Give scheduler a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the scheduler gracefully
	scheduler.Stop()

	// 5. Verify list jobs HTTP endpoint
	// Create router and register admin routes directly to avoid import cycle
	router := mux.NewRouter()
	adminAPI := router.PathPrefix("/api/admin").Subrouter()

	// Register scheduler routes directly (without admin middleware for test simplicity)
	adminAPI.HandleFunc("/scheduler/jobs", LogRoute(ListScheduledJobs(scheduler))).Methods("GET")
	adminAPI.HandleFunc("/scheduler/jobs/{jobName}/history",
		LogRoute(GetJobHistory(scheduler))).Methods("GET")

	// Generate admin JWT token
	token, err := tests.GenerateTestJWT(1)
	if err != nil {
		t.Fatalf("Failed to generate test JWT: %v", err)
	}

	// Test list jobs endpoint
	req := httptest.NewRequest("GET", "/api/admin/scheduler/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for list jobs, got %d: %s", w.Code, w.Body.String())
	}

	var listResponse ScheduledJobsResponse
	if err := json.NewDecoder(w.Body).Decode(&listResponse); err != nil {
		t.Fatalf("Failed to decode list jobs response: %v", err)
	}

	if len(listResponse.Jobs) != 1 {
		t.Errorf("Expected 1 job in response, got %d", len(listResponse.Jobs))
	}
	if listResponse.Jobs[0].Name != "daily-cleanup" {
		t.Errorf("Expected 'daily-cleanup' job, got '%s'", listResponse.Jobs[0].Name)
	}

	// 6. Verify execution tracking works
	// Create a manual test job for immediate execution
	testJob := &ManualTestJob{
		name:    "test-job",
		execute: func(ctx context.Context) error { return nil },
	}

	err = scheduler.Register(testJob)
	if err != nil {
		t.Fatalf("Failed to register test job: %v", err)
	}

	// Manually trigger the test job to verify tracking
	// We'll simulate this by creating a record directly in the database
	ctx := context.Background()
	tracker := services.NewScheduledExecutionTracker(h.DB)

	runID, err := tracker.RecordStart(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to record job start: %v", err)
	}

	if runID == 0 {
		t.Error("Expected non-zero run ID")
	}

	// Record successful completion
	err = tracker.RecordCompletion(ctx, runID, nil)
	if err != nil {
		t.Fatalf("Failed to record job completion: %v", err)
	}

	// Verify job history via HTTP endpoint
	req = httptest.NewRequest("GET", "/api/admin/scheduler/jobs/test-job/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for job history, got %d: %s", w.Code, w.Body.String())
	}

	var historyResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&historyResponse); err != nil {
		t.Fatalf("Failed to decode job history response: %v", err)
	}

	runs, ok := historyResponse["runs"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'runs' to be an array in response")
	}

	if len(runs) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(runs))
	}

	run, ok := runs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected run to be an object")
	}

	if run["job_name"] != "test-job" {
		t.Errorf("Expected job_name 'test-job', got '%v'", run["job_name"])
	}

	if run["status"] != "completed" {
		t.Errorf("Expected status 'completed', got '%v'", run["status"])
	}

	// Test failure tracking
	runID2, err := tracker.RecordStart(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to record second job start: %v", err)
	}

	testErr := fmt.Errorf("simulated failure")
	err = tracker.RecordFailure(ctx, runID2, testErr, 2)
	if err != nil {
		t.Fatalf("Failed to record job failure: %v", err)
	}

	// Verify failure is in history
	req = httptest.NewRequest("GET", "/api/admin/scheduler/jobs/test-job/history?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for job history with failures, got %d: %s", w.Code, w.Body.String())
	}

	var historyResponse2 map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&historyResponse2); err != nil {
		t.Fatalf("Failed to decode job history response: %v", err)
	}

	runs2, ok := historyResponse2["runs"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'runs' to be an array in response")
	}

	// Should have 2 entries now (1 success, 1 failure)
	if len(runs2) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(runs2))
	}

	// Verify we have both completed and failed statuses
	hasCompleted := false
	hasFailed := false
	for _, entry := range runs2 {
		run, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if run["status"] == "completed" {
			hasCompleted = true
		}
		if run["status"] == "failed" {
			hasFailed = true
			if run["error_message"] != "simulated failure" {
				t.Errorf("Expected error message 'simulated failure', got '%v'", run["error_message"])
			}
			if retryCount, ok := run["retry_count"].(float64); !ok || int(retryCount) != 2 {
				t.Errorf("Expected retry_count 2, got %v", run["retry_count"])
			}
		}
	}

	if !hasCompleted {
		t.Error("Expected to find completed status in history")
	}
	if !hasFailed {
		t.Error("Expected to find failed status in history")
	}
}

// ManualTestJob is a test job for immediate execution
type ManualTestJob struct {
	name    string
	execute func(ctx context.Context) error
}

func (j *ManualTestJob) Name() string {
	return j.name
}

func (j *ManualTestJob) Schedule() string {
	return "* * * * * *" // Every second for testing
}

func (j *ManualTestJob) Handler(ctx context.Context) error {
	return j.execute(ctx)
}

func (j *ManualTestJob) MaxRetries() int {
	return 0
}

func (j *ManualTestJob) NextRun(from time.Time) time.Time {
	// For testing, return 1 second in the future
	return from.Add(1 * time.Second)
}

// Verify ManualTestJob implements ScheduledJob interface
var _ services.ScheduledJob = (*ManualTestJob)(nil)
