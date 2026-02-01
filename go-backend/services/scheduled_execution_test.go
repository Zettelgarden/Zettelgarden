package services

import (
	"context"
	"database/sql"
	"go-backend/tests"
	"testing"
)

// TestRecordJobStart tests that RecordStart creates a new execution record
func TestRecordJobStart(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	tracker := NewScheduledExecutionTracker(s.DB)
	jobName := "test-daily-job"

	// Record the start of a job
	runID, err := tracker.RecordStart(ctx, jobName)
	if err != nil {
		t.Fatalf("RecordStart failed: %v", err)
	}

	if runID <= 0 {
		t.Errorf("Expected run ID > 0, got %d", runID)
	}

	// Verify the record was created in the database
	var run JobRun
	var completedAt sql.NullTime
	var errMsg sql.NullString

	err = s.DB.QueryRowContext(ctx, `
		SELECT id, job_name, started_at, completed_at, status, error_message, retry_count
		FROM scheduled_job_runs
		WHERE id = $1
	`, runID).Scan(&run.ID, &run.JobName, &run.StartedAt, &completedAt, &run.Status, &errMsg, &run.RetryCount)

	if err != nil {
		t.Fatalf("Failed to query job run: %v", err)
	}

	if run.JobName != jobName {
		t.Errorf("Expected job_name %s, got %s", jobName, run.JobName)
	}

	if run.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", run.Status)
	}

	// completed_at should be NULL when job is still running
	if completedAt.Valid {
		t.Errorf("Expected completed_at to be NULL for running job, got %v", completedAt.Time)
	}
}

// TestRecordJobCompletion tests that RecordCompletion marks a job as completed
func TestRecordJobCompletion(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	tracker := NewScheduledExecutionTracker(s.DB)
	jobName := "test-completion-job"

	// First, record the start
	runID, err := tracker.RecordStart(ctx, jobName)
	if err != nil {
		t.Fatalf("RecordStart failed: %v", err)
	}

	// Then record completion
	err = tracker.RecordCompletion(ctx, runID, nil)
	if err != nil {
		t.Fatalf("RecordCompletion failed: %v", err)
	}

	// Verify the record was updated in the database
	var status string
	var completedAt sql.NullTime

	err = s.DB.QueryRowContext(ctx, `
		SELECT status, completed_at
		FROM scheduled_job_runs
		WHERE id = $1
	`, runID).Scan(&status, &completedAt)

	if err != nil {
		t.Fatalf("Failed to query job run: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status)
	}

	if !completedAt.Valid {
		t.Error("Expected completed_at to be set, but it was NULL")
	}
}

// TestRecordJobFailure tests that RecordFailure marks a job as failed with error details
func TestRecordJobFailure(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	tracker := NewScheduledExecutionTracker(s.DB)
	jobName := "test-failure-job"
	testErr := "something went wrong"
	retryCount := 3

	// First, record the start
	runID, err := tracker.RecordStart(ctx, jobName)
	if err != nil {
		t.Fatalf("RecordStart failed: %v", err)
	}

	// Then record failure
	jobErr := &testJobError{msg: testErr}
	err = tracker.RecordFailure(ctx, runID, jobErr, retryCount)
	if err != nil {
		t.Fatalf("RecordFailure failed: %v", err)
	}

	// Verify the record was updated in the database
	var status string
	var completedAt sql.NullTime
	var errMsg sql.NullString
	var retryCountResult int

	err = s.DB.QueryRowContext(ctx, `
		SELECT status, completed_at, error_message, retry_count
		FROM scheduled_job_runs
		WHERE id = $1
	`, runID).Scan(&status, &completedAt, &errMsg, &retryCountResult)

	if err != nil {
		t.Fatalf("Failed to query job run: %v", err)
	}

	if status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", status)
	}

	if errMsg.String != testErr {
		t.Errorf("Expected error_message '%s', got '%s'", testErr, errMsg.String)
	}

	if retryCountResult != retryCount {
		t.Errorf("Expected retry_count %d, got %d", retryCount, retryCountResult)
	}

	if !completedAt.Valid {
		t.Error("Expected completed_at to be set, but it was NULL")
	}
}

// TestGetRecentRuns tests retrieving execution history for a job
func TestGetRecentRuns(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	tracker := NewScheduledExecutionTracker(s.DB)
	jobName := "test-history-job"

	// Create multiple job runs
	for i := 0; i < 3; i++ {
		runID, err := tracker.RecordStart(ctx, jobName)
		if err != nil {
			t.Fatalf("RecordStart failed: %v", err)
		}

		// Alternate between success and failure
		if i%2 == 0 {
			_ = tracker.RecordCompletion(ctx, runID, nil)
		} else {
			_ = tracker.RecordFailure(ctx, runID, &testJobError{msg: "test error"}, i)
		}
	}

	// Get recent runs
	runs, err := tracker.GetRecentRuns(ctx, jobName, 10)
	if err != nil {
		t.Fatalf("GetRecentRuns failed: %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("Expected 3 runs, got %d", len(runs))
	}

	// Verify runs are ordered by started_at DESC (most recent first)
	// The most recent should have the highest ID
	for i := 0; i < len(runs)-1; i++ {
		if runs[i].ID <= runs[i+1].ID {
			t.Errorf("Expected runs to be ordered by most recent first (run %d ID=%d, run %d ID=%d)",
				i, runs[i].ID, i+1, runs[i+1].ID)
		}
	}

	// Verify all runs belong to the same job
	for _, run := range runs {
		if run.JobName != jobName {
			t.Errorf("Expected job_name %s, got %s", jobName, run.JobName)
		}
	}
}

// testJobError is a simple error type for testing
type testJobError struct {
	msg string
}

func (e *testJobError) Error() string {
	return e.msg
}
