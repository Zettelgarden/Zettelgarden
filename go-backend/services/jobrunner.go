package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"time"
)

// JobRunner executes LLM jobs inline (in a background goroutine) while
// recording each run as a row in the llm_jobs audit table.
//
// Unlike the previous multi-worker queue design, there is no dequeue step,
// no worker pool, no heartbeats, and no retry/backoff scheduler. A caller
// invokes Run, an audit row is inserted, and the work happens immediately in
// a recovered goroutine. This is appropriate for a single-process,
// self-hosted deployment.
//
// If the process dies mid-job, CleanupStale (called on startup) marks any
// rows still in "running" as "failed".
type JobRunner struct {
	db        *sql.DB
	processor *LLMJobProcessor
}

// NewJobRunner creates a JobRunner backed by the given database and processor.
func NewJobRunner(db *sql.DB, processor *LLMJobProcessor) *JobRunner {
	return &JobRunner{db: db, processor: processor}
}

// Run records a job in the audit table and executes it inline in a background
// goroutine. It returns the created job row (already in "running" status) so
// callers can link it from domain tables (e.g. summarizations.llm_job_id).
// An error is returned only if the audit row could not be created; processing
// failures are recorded on the row itself rather than returned.
func (r *JobRunner) Run(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error) {
	job, err := models.CreateJob(r.db, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create job record: %w", err)
	}

	// Mark as running immediately so the audit row reflects reality. The
	// DEFAULT is 'pending', but we never actually wait in that state.
	if err := models.UpdateJobStatus(r.db, job.ID, models.JobStatusRunning); err != nil {
		log.Printf("[JobRunner] failed to mark job %d as running: %v", job.ID, err)
	}
	job.Status = models.JobStatusRunning

	log.Printf("[JobRunner] started job %d (type: %s, user: %d)", job.ID, job.JobType, job.UserID)

	go r.execute(job)

	return job, nil
}

// Retry re-runs an existing audit row. The row must be in a terminal state
// (failed or cancelled); it is reset to running and executed again.
func (r *JobRunner) Retry(ctx context.Context, jobID int) (*models.LLMJob, error) {
	job, err := models.GetJob(r.db, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to load job %d: %w", jobID, err)
	}
	if job == nil {
		return nil, fmt.Errorf("job %d not found", jobID)
	}
	if job.Status != models.JobStatusFailed && job.Status != models.JobStatusCancelled {
		return nil, fmt.Errorf("only failed or cancelled jobs can be retried (current: %s)", job.Status)
	}

	if err := models.UpdateJobStatusWithError(r.db, jobID, models.JobStatusRunning, ""); err != nil {
		return nil, fmt.Errorf("failed to reset job %d: %w", jobID, err)
	}
	job.Status = models.JobStatusRunning
	job.ErrorMessage = ""

	log.Printf("[JobRunner] retrying job %d (type: %s)", job.ID, job.JobType)
	go r.execute(job)
	return job, nil
}

// execute runs a single job to completion, updating the audit row with the
// result or error. It always recovers from panics so a single bad job cannot
// take down the process.
func (r *JobRunner) execute(job *models.LLMJob) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[JobRunner] panic in job %d: %v", job.ID, rec)
			_ = models.UpdateJobStatusWithError(r.db, job.ID, models.JobStatusFailed, fmt.Sprintf("panic: %v", rec))
		}
	}()

	timeout := time.Duration(job.TimeoutSecs) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := r.processor.ProcessJob(ctx, job)
	if err != nil {
		log.Printf("[JobRunner] job %d failed: %v", job.ID, err)
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("timed out after %ds: %w", job.TimeoutSecs, err)
		}
		if updateErr := models.UpdateJobStatusWithError(r.db, job.ID, models.JobStatusFailed, err.Error()); updateErr != nil {
			log.Printf("[JobRunner] failed to record failure for job %d: %v", job.ID, updateErr)
		}
		return
	}

	if err := models.UpdateJobStatusWithResult(r.db, job.ID, models.JobStatusCompleted, result); err != nil {
		log.Printf("[JobRunner] failed to record completion for job %d: %v", job.ID, err)
	}
	log.Printf("[JobRunner] job %d completed (type: %s)", job.ID, job.JobType)
}

// CleanupStale marks any jobs left in "running" (from a crashed or restarted
// process) as failed. Intended to be called once at startup.
func (r *JobRunner) CleanupStale(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE llm_jobs
		 SET status = 'failed',
		     error_message = COALESCE(error_message, '') || 'Job orphaned by process restart',
		     completed_at = NOW()
		 WHERE status = 'running'`)
	if err != nil {
		return 0, fmt.Errorf("failed to clean up stale jobs: %w", err)
	}
	count, _ := result.RowsAffected()
	if count > 0 {
		log.Printf("[JobRunner] marked %d stale running job(s) as failed", count)
	}
	return int(count), nil
}
