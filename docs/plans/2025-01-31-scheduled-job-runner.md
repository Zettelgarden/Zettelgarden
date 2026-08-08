> **STATUS: HISTORICAL — pre-SQLite era.** This plan predates the PostgreSQL→SQLite cutover (2026-07-28, epic Zettelgarden-c7j) and the move to local on-disk file storage (epic Zettelgarden-yar). Zettelgarden now runs SQLite-only with local storage; this document is kept for design history.

# Scheduled Job Runner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an in-process scheduled job runner using cron that executes periodic tasks with retry logic and execution history tracking.

**Architecture:** A new scheduler service running as a goroutine alongside existing worker pools, using robfig/cron for time-based triggering and a new database table for execution history. Scheduled jobs implement a common interface and are registered at startup.

**Tech Stack:** Go 1.23+, robfig/cron v3, PostgreSQL (existing), context-based cancellation

---

## Task 1: Create database migration for scheduled_job_runs table

**Files:**
- Create: `go-backend/schema/00XY-scheduled-job-runs-table.sql` (check existing migrations for next number)

**Step 1: Write the migration SQL**

Create table to track scheduled job executions:

```sql
-- Track scheduled job execution history
CREATE TABLE IF NOT EXISTS scheduled_job_runs (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(255) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL, -- 'running', 'completed', 'failed'
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for job name lookups
CREATE INDEX idx_scheduled_job_runs_job_name ON scheduled_job_runs(job_name);

-- Index for status queries
CREATE INDEX idx_scheduled_job_runs_status ON scheduled_job_runs(status);

-- Index for started_at (for history queries)
CREATE INDEX idx_scheduled_job_runs_started_at ON scheduled_job_runs(started_at DESC);

-- Index for cleanup of old records
CREATE INDEX idx_scheduled_job_runs_created_at ON scheduled_job_runs(created_at);
```

**Step 2: Test migration locally**

Run: `cd go-backend && source .env-bash && psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f schema/00XY-scheduled-job-runs-table.sql`

Expected: Table created successfully, indexes created

**Step 3: Verify table structure**

Run: `psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\d scheduled_job_runs"`

Expected: Table schema showing all columns and indexes

**Step 4: Commit**

```bash
git add go-backend/schema/00XY-scheduled-job-runs-table.sql
git commit -m "feat: add scheduled_job_runs table for execution tracking"
```

---

## Task 2: Create ScheduledJob interface and base types

**Files:**
- Create: `go-backend/services/scheduled_job.go`

**Step 1: Write the failing test**

Create `services/scheduled_job_test.go`:

```go
package services

import (
	"context"
	"testing"
	"time"
)

// Test that a job implementing the interface can be queried
func TestScheduledJobInterface(t *testing.T) {
	job := &mockScheduledJob{
		name:       "test-job",
		schedule:   "*/5 * * * *",
		maxRetries: 3,
	}

	if job.Name() != "test-job" {
		t.Errorf("expected name 'test-job', got '%s'", job.Name())
	}

	if job.Schedule() != "*/5 * * * *" {
		t.Errorf("expected schedule '*/5 * * * *', got '%s'", job.Schedule())
	}

	if job.MaxRetries() != 3 {
		t.Errorf("expected max retries 3, got %d", job.MaxRetries())
	}
}

// mockScheduledJob implements ScheduledJob for testing
type mockScheduledJob struct {
	name       string
	schedule   string
	maxRetries int
	handlerErr error
}

func (m *mockScheduledJob) Name() string       { return m.name }
func (m *mockScheduledJob) Schedule() string   { return m.schedule }
func (m *mockScheduledJob) MaxRetries() int    { return m.maxRetries }
func (m *mockScheduledJob) Handler(ctx context.Context) error {
	return m.handlerErr
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -v -run TestScheduledJobInterface`

Expected: FAIL with "undefined: ScheduledJob" (interface doesn't exist yet)

**Step 3: Write minimal implementation**

Create `services/scheduled_job.go`:

```go
package services

import "context"

// ScheduledJob defines the interface for jobs that can be scheduled
// for periodic execution using cron syntax.
type ScheduledJob interface {
	// Name returns a unique identifier for this job
	Name() string

	// Schedule returns the cron expression for when this job should run.
	// Supports standard cron (5 fields) or extended (6 fields with seconds).
	// Examples:
	//   "0 * * * *"       - every hour
	//   "*/5 * * * *"     - every 5 minutes
	//   "0 0 * * *"       - daily at midnight
	//   "0 9 * * 1-5"     - weekdays at 9am
	Schedule() string

	// Handler executes the job logic. The context will be cancelled
	// if the job exceeds its timeout or the server is shutting down.
	Handler(ctx context.Context) error

	// MaxRetries returns the number of times to retry on failure.
	// Use 0 for no retries, -1 for infinite retries.
	MaxRetries() int
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -v -run TestScheduledJobInterface`

Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/scheduled_job.go go-backend/services/scheduled_job_test.go
git commit -m "feat: add ScheduledJob interface"
```

---

## Task 3: Create execution tracker model for database operations

**Files:**
- Create: `go-backend/services/scheduled_execution.go`
- Modify: `go-backend/services/scheduled_execution_test.go` (create test file)

**Step 1: Write the failing test**

Create `services/scheduled_execution_test.go`:

```go
package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "dbname=zettelgarden_test sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestRecordJobStart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	tracker := NewScheduledExecutionTracker(db)

	runID, err := tracker.RecordStart(ctx, "test-job")
	assert.NoError(t, err)
	assert.NotZero(t, runID)

	// Verify record was created
	var status string
	err = db.QueryRowContext(ctx, "SELECT status FROM scheduled_job_runs WHERE id = $1", runID).Scan(&status)
	assert.NoError(t, err)
	assert.Equal(t, "running", status)
}

func TestRecordJobCompletion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	tracker := NewScheduledExecutionTracker(db)

	runID, _ := tracker.RecordStart(ctx, "test-job")

	err := tracker.RecordCompletion(ctx, runID, nil)
	assert.NoError(t, err)

	// Verify record was updated
	var status, completedAt time.Time
	err = db.QueryRowContext(ctx,
		"SELECT status, completed_at FROM scheduled_job_runs WHERE id = $1",
		runID).Scan(&status, &completedAt)
	assert.NoError(t, err)
	assert.Equal(t, "completed", status)
	assert.False(t, completedAt.IsZero())
}

func TestRecordJobFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	tracker := NewScheduledExecutionTracker(db)

	runID, _ := tracker.RecordStart(ctx, "test-job")

	testErr := assert.AnError
	err := tracker.RecordFailure(ctx, runID, testErr, 2)
	assert.NoError(t, err)

	// Verify failure was recorded
	var status string
	var retryCount int
	err = db.QueryRowContext(ctx,
		"SELECT status, retry_count FROM scheduled_job_runs WHERE id = $1",
		runID).Scan(&status, &retryCount)
	assert.NoError(t, err)
	assert.Equal(t, "failed", status)
	assert.Equal(t, 2, retryCount)
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -v -run TestRecordJob`

Expected: FAIL with "undefined: ScheduledExecutionTracker" (type doesn't exist yet)

**Step 3: Write minimal implementation**

Create `services/scheduled_execution.go`:

```go
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ScheduledExecutionTracker manages database records for scheduled job executions
type ScheduledExecutionTracker struct {
	db *sql.DB
}

// NewScheduledExecutionTracker creates a new tracker
func NewScheduledExecutionTracker(db *sql.DB) *ScheduledExecutionTracker {
	return &ScheduledExecutionTracker{db: db}
}

// RecordStart creates a new execution record when a job starts
func (t *ScheduledExecutionTracker) RecordStart(ctx context.Context, jobName string) (int64, error) {
	query := `
		INSERT INTO scheduled_job_runs (job_name, status, started_at)
		VALUES ($1, 'running', NOW())
		RETURNING id
	`

	var runID int64
	err := t.db.QueryRowContext(ctx, query, jobName).Scan(&runID)
	if err != nil {
		return 0, fmt.Errorf("failed to record job start: %w", err)
	}

	return runID, nil
}

// RecordCompletion marks a job as successfully completed
func (t *ScheduledExecutionTracker) RecordCompletion(ctx context.Context, runID int64, err error) error {
	query := `
		UPDATE scheduled_job_runs
		SET status = 'completed', completed_at = NOW()
		WHERE id = $1
	`

	_, execErr := t.db.ExecContext(ctx, query, runID)
	if execErr != nil {
		return fmt.Errorf("failed to record job completion: %w", execErr)
	}

	return nil
}

// RecordFailure marks a job as failed with error details and retry count
func (t *ScheduledExecutionTracker) RecordFailure(ctx context.Context, runID int64, jobErr error, retryCount int) error {
	query := `
		UPDATE scheduled_job_runs
		SET status = 'failed',
		    completed_at = NOW(),
		    error_message = $1,
		    retry_count = $2
		WHERE id = $3
	`

	var errMsg string
	if jobErr != nil {
		errMsg = jobErr.Error()
	}

	_, execErr := t.db.ExecContext(ctx, query, errMsg, retryCount, runID)
	if execErr != nil {
		return fmt.Errorf("failed to record job failure: %w", execErr)
	}

	return nil
}

// GetRecentRuns returns execution history for a specific job
func (t *ScheduledExecutionTracker) GetRecentRuns(ctx context.Context, jobName string, limit int) ([]JobRun, error) {
	query := `
		SELECT id, job_name, started_at, completed_at, status, error_message, retry_count
		FROM scheduled_job_runs
		WHERE job_name = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := t.db.QueryContext(ctx, query, jobName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query job runs: %w", err)
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var run JobRun
		var completedAt sql.NullTime
		var errMsg sql.NullString

		err := rows.Scan(
			&run.ID, &run.JobName, &run.StartedAt, &completedAt,
			&run.Status, &errMsg, &run.RetryCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job run: %w", err)
		}

		run.CompletedAt = completedAt.Time
		run.ErrorMessage = errMsg.String
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// JobRun represents a single scheduled job execution record
type JobRun struct {
	ID          int64
	JobName     string
	StartedAt   time.Time
	CompletedAt time.Time
	Status      string
	ErrorMessage string
	RetryCount  int
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -v -run TestRecordJob`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add go-backend/services/scheduled_execution.go go-backend/services/scheduled_execution_test.go
git commit -m "feat: add scheduled execution tracker with database operations"
```

---

## Task 4: Create the core Scheduler service

**Files:**
- Create: `go-backend/services/scheduler.go`
- Create: `go-backend/services/scheduler_test.go`

**Step 1: Write the failing test**

Create `services/scheduler_test.go`:

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerRegisterAndStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler := NewScheduler(nil) // nil DB for test

	executed := false
	mockJob := &mockScheduledJob{
		name:       "test-job",
		schedule:   "* * * * *", // Every minute (we'll trigger manually)
		maxRetries: 0,
		handler: func(ctx context.Context) error {
			executed = true
			return nil
		},
	}

	err := scheduler.Register(mockJob)
	require.NoError(t, err)

	// Start scheduler in background
	scheduler.Start()
	defer scheduler.Stop()

	// Trigger job manually for test
	scheduler.runJob(mockJob)

	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "job should have been executed")
}

func TestSchedulerDuplicateRegistration(t *testing.T) {
	scheduler := NewScheduler(nil)

	job1 := &mockScheduledJob{name: "dup-job", schedule: "* * * * *", maxRetries: 0}
	job2 := &mockScheduledJob{name: "dup-job", schedule: "*/5 * * * *", maxRetries: 0}

	err := scheduler.Register(job1)
	require.NoError(t, err)

	err = scheduler.Register(job2)
	assert.Error(t, err, "should not allow duplicate job names")
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -v -run TestScheduler`

Expected: FAIL with "undefined: NewScheduler" (scheduler doesn't exist yet)

**Step 3: Write minimal implementation**

Create `services/scheduler.go`:

```go
package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled jobs using cron expressions
type Scheduler struct {
	cron       *cron.Cron
	jobs       map[string]ScheduledJob
	tracker    *ScheduledExecutionTracker
	mu         sync.RWMutex
	logger     *log.Logger
}

// NewScheduler creates a new scheduler instance
// If db is nil, execution tracking is disabled (useful for testing)
func NewScheduler(db *sql.DB) *Scheduler {
	opts := cron.New(
		cron.WithSeconds(), // Support 6-field cron syntax
		cron.WithLogger(cron.VerbosePrintfLogger(log.Default())),
	)

	var tracker *ScheduledExecutionTracker
	if db != nil {
		tracker = NewScheduledExecutionTracker(db)
	}

	return &Scheduler{
		cron:    opts,
		jobs:    make(map[string]ScheduledJob),
		tracker: tracker,
		logger:  log.Default(),
	}
}

// Register adds a scheduled job to the scheduler
func (s *Scheduler) Register(job ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := job.Name()
	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job with name '%s' is already registered", name)
	}

	s.jobs[name] = job
	s.logger.Printf("[scheduler] registered job: %s (schedule: %s)", name, job.Schedule())

	return nil
}

// Start begins the scheduler, should be called after all jobs are registered
func (s *Scheduler) Start() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, job := range s.jobs {
		schedule := job.Schedule()
		_, err := s.cron.AddFunc(schedule, func() {
			s.runJob(job)
		})
		if err != nil {
			s.logger.Printf("[scheduler] ERROR invalid cron schedule for %s: %v", name, err)
			continue
		}
	}

	s.cron.Start()
	s.logger.Println("[scheduler] started")
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop() // Stops cron and returns context that's done when running jobs finish
	<-ctx.Done()
	s.logger.Println("[scheduler] stopped")
}

// runJob executes a single scheduled job with retry logic and tracking
func (s *Scheduler) runJob(job ScheduledJob) {
	name := job.Name()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var runID int64
	if s.tracker != nil {
		var err error
		runID, err = s.tracker.RecordStart(ctx, name)
		if err != nil {
			s.logger.Printf("[scheduler] ERROR failed to record start for %s: %v", name, err)
			return
		}
	}

	var lastErr error
	maxRetries := job.MaxRetries()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Printf("[scheduler] %s retry %d/%d", name, attempt, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second) // Backoff
		}

		err := job.Handler(ctx)
		if err == nil {
			if s.tracker != nil {
				if recErr := s.tracker.RecordCompletion(ctx, runID, nil); recErr != nil {
					s.logger.Printf("[scheduler] ERROR failed to record completion for %s: %v", name, recErr)
				}
			}
			s.logger.Printf("[scheduler] %s completed successfully", name)
			return
		}

		lastErr = err
		s.logger.Printf("[scheduler] %s failed: %v", name, err)
	}

	// All retries exhausted
	if s.tracker != nil {
		if recErr := s.tracker.RecordFailure(ctx, runID, lastErr, maxRetries); recErr != nil {
			s.logger.Printf("[scheduler] ERROR failed to record failure for %s: %v", name, recErr)
		}
	}
	s.logger.Printf("[scheduler] %s failed after %d retries: %v", name, maxRetries, lastErr)
}

// GetJobHistory returns recent execution history for a job
func (s *Scheduler) GetJobHistory(ctx context.Context, jobName string, limit int) ([]JobRun, error) {
	if s.tracker == nil {
		return nil, fmt.Errorf("execution tracking is not enabled")
	}
	return s.tracker.GetRecentRuns(ctx, jobName, limit)
}

// ListJobs returns all registered job names
func (s *Scheduler) ListJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.jobs))
	for name := range s.jobs {
		names = append(names, name)
	}
	return names
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -v -run TestScheduler`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add go-backend/services/scheduler.go go-backend/services/scheduler_test.go
git commit -m "feat: add core Scheduler service with cron"
```

---

## Task 5: Create admin HTTP handlers for scheduler management

**Files:**
- Create: `go-backend/handlers/scheduler.go`
- Create: `go-backend/handlers/scheduler_test.go`

**Step 1: Write the failing test**

Create `handlers/scheduler_test.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListScheduledJobsHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/admin/scheduler/jobs", nil)
	w := httptest.NewRecorder()

	// Mock scheduler with test jobs
	mockScheduler := &mockSchedulerForHandler{
		jobs: []string{"cleanup", "daily-report"},
	}

	handler := ListScheduledJobs(mockScheduler)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	jobs := response["jobs"].([]interface{})
	assert.Len(t, jobs, 2)
}

func TestGetJobHistoryHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/admin/scheduler/jobs/test-job/history?limit=10", nil)
	w := httptest.NewRecorder()

	mockScheduler := &mockSchedulerForHandler{
		history: []JobRunResponse{
			{Status: "completed"},
			{Status: "failed"},
		},
	}

	handler := GetJobHistory(mockScheduler)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	runs := response["runs"].([]interface{})
	assert.Len(t, runs, 2)
}

// Mock types for testing
type mockSchedulerForHandler struct {
	jobs    []string
	history []JobRunResponse
}

func (m *mockSchedulerForHandler) ListJobs() []string                 { return m.jobs }
func (m *mockSchedulerForHandler) GetJobHistory(ctx context.Context, jobName string, limit int) ([]JobRunResponse, error) {
	return m.history, nil
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./handlers -v -run TestListScheduledJobs`

Expected: FAIL with "undefined: ListScheduledJobs" (handler doesn't exist yet)

**Step 3: Write minimal implementation**

Create `handlers/scheduler.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// SchedulerHandler provides HTTP interface for scheduler operations
type SchedulerHandler struct {
	scheduler SchedulerAPI
}

// SchedulerAPI defines the interface needed by handlers (for testability)
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]JobRunResponse, error)
}

// JobRunResponse represents a job run in API responses
type JobRunResponse struct {
	ID           int64  `json:"id"`
	JobName      string `json:"job_name"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	RetryCount   int    `json:"retry_count"`
}

// ListScheduledJobs returns all registered scheduled jobs
func ListScheduledJobs(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobs := scheduler.ListJobs()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jobs": jobs,
		})
	}
}

// GetJobHistory returns execution history for a specific job
func GetJobHistory(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		jobName := vars["jobName"]

		// Parse limit from query params
		limit := 50 // default
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		runs, err := scheduler.GetJobHistory(r.Context(), jobName, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"runs": runs,
		})
	}
}
```

Update `services/scheduler.go` to add the adapter:

```go
// Add these methods to the Scheduler struct

// ToJobRunResponse converts a JobRun to JobRunResponse
func ToJobRunResponse(run JobRun) JobRunResponse {
	resp := JobRunResponse{
		ID:         run.ID,
		JobName:    run.JobName,
		StartedAt:  run.StartedAt.Format(time.RFC3339),
		Status:     run.Status,
		RetryCount: run.RetryCount,
	}

	if !run.CompletedAt.IsZero() {
		resp.CompletedAt = run.CompletedAt.Format(time.RFC3339)
	}

	if run.ErrorMessage != "" {
		resp.ErrorMessage = run.ErrorMessage
	}

	return resp
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./handlers -v -run TestListScheduledJobs`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add go-backend/handlers/scheduler.go go-backend/handlers/scheduler_test.go
git commit -m "feat: add admin HTTP handlers for scheduler"
```

---

## Task 6: Register routes in routes.go

**Files:**
- Modify: `go-backend/routes/routes.go`

**Step 1: Add routes to existing routes file**

In `routes/routes.go`, add the admin scheduler routes:

```go
// Add import
import "github.com/gorilla/mux"

// Add to AdminRouter function (create if it doesn't exist)
func AdminRouter(r *mux.Router, scheduler services.SchedulerAPI) {
	admin := r.PathPrefix("/api/admin").Subrouter()

	// Existing admin routes...

	// Scheduler management routes (admin only)
	admin.HandleFunc("/scheduler/jobs", handlers.ListScheduledJobs(scheduler)).Methods("GET")
	admin.HandleFunc("/scheduler/jobs/{jobName}/history", handlers.GetJobHistory(scheduler)).Methods("GET")
}
```

**Step 2: Verify route registration compiles**

Run: `cd go-backend && go build ./routes`

Expected: Compilation succeeds

**Step 3: Commit**

```bash
git add go-backend/routes/routes.go
git commit -m "feat: register scheduler admin routes"
```

---

## Task 7: Integrate scheduler into main.go

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Add scheduler initialization to main.go**

After worker pool initialization, add scheduler startup:

```go
// In main(), after db initialization

// Initialize and start the scheduled job runner
scheduler := services.NewScheduler(db)

// Register scheduled jobs here (we'll add actual jobs in next tasks)
// Example: scheduler.Register(jobs.NewCleanupJob(db))

scheduler.Start()
defer scheduler.Stop()

log.Println("[main] scheduled job runner started")
```

**Step 2: Update AdminRouter call to pass scheduler**

In main.go where routes are registered:

```go
routes.AdminRouter(router, scheduler)
```

**Step 3: Verify compilation**

Run: `cd go-backend && go build`

Expected: Binary compiles successfully

**Step 4: Test local startup**

Run: `cd go-backend && source .env-bash && timeout 5s go run main.go` or run normally and verify startup

Expected: Server starts, log shows "[scheduler] started"

**Step 5: Commit**

```bash
git add go-backend/main.go
git commit -m "feat: integrate scheduler into main server"
```

---

## Task 8: Create example scheduled job - Daily Cleanup Job

**Files:**
- Create: `go-backend/services/jobs/cleanup_job.go`
- Create: `go-backend/services/jobs/cleanup_job_test.go`

**Step 1: Write the failing test**

Create `services/jobs/cleanup_job_test.go`:

```go
package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupJobImplementsInterface(t *testing.T) {
	job := NewCleanupJob(nil)

	assert.Implements(t, (*ScheduledJob)(nil), job)
	assert.Equal(t, "daily-cleanup", job.Name())
	assert.Equal(t, "0 2 * * *", job.Schedule()) // 2 AM daily
	assert.Equal(t, 3, job.MaxRetries())
}

func TestCleanupJobHandler(t *testing.T) {
	// This test requires a test database setup
	// For now, test the structure
	job := NewCleanupJob(nil)

	ctx := context.Background()
	err := job.Handler(ctx)

	// Should either succeed or fail gracefully (not panic)
	// We'll make it succeed with nil DB
	assert.NoError(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services/jobs -v`

Expected: FAIL with package does not exist

**Step 3: Write minimal implementation**

Create `services/jobs/cleanup_job.go`:

```go
package jobs

import (
	"context"
	"database/sql"
	"log"

	"github.com/yourusername/zettelgarden/services"
)

// CleanupJob performs daily cleanup of old data
type CleanupJob struct {
	db *sql.DB
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(db *sql.DB) *CleanupJob {
	return &CleanupJob{db: db}
}

func (j *CleanupJob) Name() string {
	return "daily-cleanup"
}

func (j *CleanupJob) Schedule() string {
	return "0 2 * * *" // Run at 2 AM daily
}

func (j *CleanupJob) MaxRetries() int {
	return 3
}

func (j *CleanupJob) Handler(ctx context.Context) error {
	log.Println("[cleanup-job] starting daily cleanup")

	if j.db == nil {
		log.Println("[cleanup-job] no database configured, skipping")
		return nil
	}

	// Clean up old scheduled_job_runs (keep last 90 days)
	_, err := j.db.ExecContext(ctx,
		"DELETE FROM scheduled_job_runs WHERE created_at < NOW() - INTERVAL '90 days'")
	if err != nil {
		log.Printf("[cleanup-job] failed to clean old runs: %v", err)
		return err
	}

	log.Println("[cleanup-job] completed successfully")
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services/jobs -v`

Expected: PASS

**Step 5: Register the job in main.go**

In main.go, after creating the scheduler:

```go
// Register scheduled jobs
scheduler.Register(jobs.NewCleanupJob(db))
```

**Step 6: Commit**

```bash
git add go-backend/services/jobs/cleanup_job.go go-backend/services/jobs/cleanup_job_test.go go-backend/main.go
git commit -m "feat: add daily cleanup scheduled job"
```

---

## Task 9: Add graceful shutdown support

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Update shutdown logic to include scheduler**

Ensure the scheduler's Stop() is called in the shutdown sequence:

```go
// In the shutdown handler (where worker pools are stopped)
log.Println("[main] shutting down scheduler...")
scheduler.Stop()
```

**Step 2: Verify shutdown works**

Run server, then send SIGINT (Ctrl+C), verify scheduler stops cleanly

Expected: Log shows "[scheduler] stopped" before exit

**Step 3: Commit**

```bash
git add go-backend/main.go
git commit -m "feat: add graceful shutdown for scheduler"
```

---

## Task 10: Add integration test for end-to-end flow

**Files:**
- Create: `go-backend/integration_test.go` or add to handlers package

**Step 1: Write integration test**

```go
package handlers_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/zettelgarden/handlers"
	"github.com/yourusername/zettelgarden/services"
	"github.com/yourusername/zettelgarden/services/jobs"
	_ "github.com/lib/pq"
)

func TestSchedulerIntegration(t *testing.T) {
	// Setup test database
	db, err := sql.Open("postgres", "dbname=zettelgarden_test sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduled_job_runs (
			id SERIAL PRIMARY KEY,
			job_name VARCHAR(255) NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE,
			status VARCHAR(50) NOT NULL,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	// Create scheduler
	scheduler := services.NewScheduler(db)

	// Register test job
	job := jobs.NewCleanupJob(db)
	err = scheduler.Register(job)
	require.NoError(t, err)

	// Start scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Create handler
	handler := handlers.ListScheduledJobs(scheduler)

	// Test list jobs endpoint
	req := httptest.NewRequest("GET", "/api/admin/scheduler/jobs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for job to potentially run (won't run on 1-second test)
	time.Sleep(2 * time.Second)
}
```

**Step 2: Run integration test**

Run: `cd go-backend && go test -v -run TestSchedulerIntegration`

Expected: PASS

**Step 3: Commit**

```bash
git add go-backend/integration_test.go
git commit -m "test: add scheduler integration test"
```

---

## Task 11: Add health check endpoint for scheduler status

**Files:**
- Modify: `go-backend/handlers/scheduler.go`
- Modify: `go-backend/routes/routes.go`

**Step 1: Add health check handler**

In `handlers/scheduler.go`:

```go
// SchedulerHealth returns the current health status of the scheduler
type SchedulerHealth struct {
	Running bool     `json:"running"`
	Jobs    []string `json:"jobs"`
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
```

**Step 2: Register health check route**

In `routes/routes.go`:

```go
admin.HandleFunc("/scheduler/health", handlers.GetSchedulerHealth(scheduler)).Methods("GET")
```

**Step 3: Test endpoint manually**

Run: `curl http://localhost:8080/api/admin/scheduler/health`

Expected: JSON response with running status and job list

**Step 4: Commit**

```bash
git add go-backend/handlers/scheduler.go go-backend/routes/routes.go
git commit -m "feat: add scheduler health check endpoint"
```

---

## Summary

This plan creates a complete scheduled job runner system with:

1. **Database schema** for execution tracking
2. **ScheduledJob interface** for type-safe job definitions
3. **Execution tracker** for database operations
4. **Core Scheduler** using robfig/cron
5. **Admin HTTP handlers** for management
6. **Route registration** in the API
7. **main.go integration** for startup/shutdown
8. **Example cleanup job** demonstrating usage
9. **Graceful shutdown** support
10. **Integration tests** for end-to-end verification
11. **Health check** endpoint for monitoring

Each task follows TDD principles with failing tests first, minimal implementation, and immediate commits.
