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
	return t.GetRecentRunWithOffset(ctx, jobName, limit, 0)
}

// GetRecentRunWithOffset returns execution history for a specific job with offset support
func (t *ScheduledExecutionTracker) GetRecentRunWithOffset(ctx context.Context, jobName string, limit int, offset int) ([]JobRun, error) {
	query := `
		SELECT id, job_name, started_at, completed_at, status, error_message, retry_count
		FROM scheduled_job_runs
		WHERE job_name = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := t.db.QueryContext(ctx, query, jobName, limit, offset)
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
