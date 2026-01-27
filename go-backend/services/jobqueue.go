package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"time"
)

// JobQueue defines the interface for queue operations
type JobQueue interface {
	// Enqueue adds a new job to the queue
	Enqueue(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error)

	// Dequeue claims and returns the next pending job for processing
	// Uses row-level locking to allow multiple workers
	Dequeue(ctx context.Context) (*models.LLMJob, error)

	// DequeueByTypes claims and returns the next pending job of specific types
	// Uses row-level locking to allow multiple workers
	DequeueByTypes(ctx context.Context, jobTypes []models.JobType) (*models.LLMJob, error)

	// UpdateStatus updates the status of a job
	UpdateStatus(ctx context.Context, jobID int, status models.JobStatus) error

	// UpdateStatusWithResult updates status and stores the result
	UpdateStatusWithResult(ctx context.Context, jobID int, status models.JobStatus, result map[string]interface{}) error

	// UpdateStatusWithError updates status and stores an error message
	UpdateStatusWithError(ctx context.Context, jobID int, status models.JobStatus, errorMsg string) error

	// MarkRunning marks a job as running and sets started_at
	MarkRunning(ctx context.Context, jobID int) error

	// IncrementRetry increments retry count and resets to pending
	IncrementRetry(ctx context.Context, jobID int) error

	// Get retrieves a job by ID
	Get(ctx context.Context, jobID int) (*models.LLMJob, error)

	// List retrieves jobs for a user with optional filtering
	List(ctx context.Context, params models.JobListParams) ([]models.LLMJob, error)

	// Cancel cancels a pending job
	Cancel(ctx context.Context, jobID, userID int) error

	// Stats retrieves statistics for a user
	Stats(ctx context.Context, userID int) (*models.JobStats, error)

	// GetQueueDepth returns the number of pending jobs in the queue
	GetQueueDepth(ctx context.Context) (int, error)

	// UpdateHeartbeat updates the last_heartbeat timestamp for a running job
	UpdateHeartbeat(ctx context.Context, jobID int) error

	// MarkStuckJobsFailed finds jobs that have exceeded timeout and marks them as failed
	MarkStuckJobsFailed(ctx context.Context) (int, error)

	// CleanupOrphanedJobs marks jobs stuck in "running" state as failed after server restart
	CleanupOrphanedJobs(ctx context.Context) (int, error)
}

// DatabaseJobQueue implements JobQueue using PostgreSQL as the backend
type DatabaseJobQueue struct {
	db *sql.DB
}

// NewJobQueue creates a new JobQueue backed by the database
func NewJobQueue(db *sql.DB) JobQueue {
	return &DatabaseJobQueue{db: db}
}

// Enqueue adds a new job to the queue
func (q *DatabaseJobQueue) Enqueue(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error) {
	// Use the model function for consistency
	job, err := models.CreateJob(q.db, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}
	log.Printf("[JobQueue] Enqueued job %d (type: %s, user: %d)", job.ID, job.JobType, job.UserID)
	return job, nil
}

// Dequeue claims and returns the next pending job for processing
func (q *DatabaseJobQueue) Dequeue(ctx context.Context) (*models.LLMJob, error) {
	// Begin a transaction for atomic dequeue operation
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call if already committed

	// Use FOR UPDATE SKIP LOCKED to allow multiple workers to run concurrently
	// This locks the row for this transaction but skips rows already locked by other workers
	query := `
		SELECT id, user_id, job_type, status, priority, payload, result, error_message,
			created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds
		FROM llm_jobs
		WHERE status = 'pending'
		ORDER BY priority ASC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job models.LLMJob
	var payloadJSON, resultJSON []byte
	var startedAt, completedAt sql.NullTime
	var errorMessage sql.NullString

	err = tx.QueryRowContext(ctx, query).Scan(
		&job.ID,
		&job.UserID,
		&job.JobType,
		&job.Status,
		&job.Priority,
		&payloadJSON,
		&resultJSON,
		&errorMessage,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&job.RetryCount,
		&job.MaxRetries,
		&job.TimeoutSecs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No jobs available
			return nil, nil
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// Handle nullable error_message
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}

	// Unmarshal payload
	if len(payloadJSON) > 0 && string(payloadJSON) != "null" {
		if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	} else {
		job.Payload = make(map[string]interface{})
	}

	// Unmarshal result if present
	if len(resultJSON) > 0 && string(resultJSON) != "null" {
		if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	// Handle nullable timestamps
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Mark job as running within the same transaction
	now := time.Now()
	markQuery := `
		UPDATE llm_jobs
		SET status = 'running', started_at = $1
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, markQuery, now, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark job as running: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	job.Status = models.JobStatusRunning
	job.StartedAt = &now

	log.Printf("[JobQueue] Dequeued job %d (type: %s, user: %d, priority: %d)",
		job.ID, job.JobType, job.UserID, job.Priority)

	return &job, nil
}

// DequeueByTypes claims and returns the next pending job of specific types for processing
func (q *DatabaseJobQueue) DequeueByTypes(ctx context.Context, jobTypes []models.JobType) (*models.LLMJob, error) {
	// Begin a transaction for atomic dequeue operation
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call if already committed

	// Build job type filter
	typeFilter := ""
	if len(jobTypes) > 0 {
		typeFilter = " AND job_type IN ("
		for i, jt := range jobTypes {
			if i > 0 {
				typeFilter += ","
			}
			typeFilter += fmt.Sprintf("'%s'", jt)
		}
		typeFilter += ")"
	}

	// Use FOR UPDATE SKIP LOCKED to allow multiple workers to run concurrently
	query := `
		SELECT id, user_id, job_type, status, priority, payload, result, error_message,
			created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds
		FROM llm_jobs
		WHERE status = 'pending'` + typeFilter + `
		ORDER BY priority ASC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job models.LLMJob
	var payloadJSON, resultJSON []byte
	var startedAt, completedAt sql.NullTime
	var errorMessage sql.NullString

	err = tx.QueryRowContext(ctx, query).Scan(
		&job.ID,
		&job.UserID,
		&job.JobType,
		&job.Status,
		&job.Priority,
		&payloadJSON,
		&resultJSON,
		&errorMessage,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&job.RetryCount,
		&job.MaxRetries,
		&job.TimeoutSecs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No jobs available
			return nil, nil
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// Handle nullable error_message
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}

	// Unmarshal payload
	if len(payloadJSON) > 0 && string(payloadJSON) != "null" {
		if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	} else {
		job.Payload = make(map[string]interface{})
	}

	// Unmarshal result if present
	if len(resultJSON) > 0 && string(resultJSON) != "null" {
		if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	// Handle nullable timestamps
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Mark job as running within the same transaction
	now := time.Now()
	markQuery := `
		UPDATE llm_jobs
		SET status = 'running', started_at = $1
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, markQuery, now, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark job as running: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	job.Status = models.JobStatusRunning
	job.StartedAt = &now

	log.Printf("[JobQueue] Dequeued job %d (type: %s, user: %d, priority: %d)",
		job.ID, job.JobType, job.UserID, job.Priority)

	return &job, nil
}

// UpdateStatus updates the status of a job
func (q *DatabaseJobQueue) UpdateStatus(ctx context.Context, jobID int, status models.JobStatus) error {
	err := models.UpdateJobStatus(q.db, jobID, status)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	log.Printf("[JobQueue] Job %d status updated to %s", jobID, status)
	return nil
}

// UpdateStatusWithResult updates status and stores the result
func (q *DatabaseJobQueue) UpdateStatusWithResult(ctx context.Context, jobID int, status models.JobStatus, result map[string]interface{}) error {
	err := models.UpdateJobStatusWithResult(q.db, jobID, status, result)
	if err != nil {
		return fmt.Errorf("failed to update job status with result: %w", err)
	}
	log.Printf("[JobQueue] Job %d completed with status %s", jobID, status)
	return nil
}

// UpdateStatusWithError updates status and stores an error message
func (q *DatabaseJobQueue) UpdateStatusWithError(ctx context.Context, jobID int, status models.JobStatus, errorMsg string) error {
	err := models.UpdateJobStatusWithError(q.db, jobID, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to update job status with error: %w", err)
	}
	log.Printf("[JobQueue] Job %d failed with status %s: %s", jobID, status, errorMsg)
	return nil
}

// MarkRunning marks a job as running and sets started_at
func (q *DatabaseJobQueue) MarkRunning(ctx context.Context, jobID int) error {
	err := models.MarkJobRunning(q.db, jobID)
	if err != nil {
		return fmt.Errorf("failed to mark job as running: %w", err)
	}
	return nil
}

// IncrementRetry increments retry count and resets to pending
func (q *DatabaseJobQueue) IncrementRetry(ctx context.Context, jobID int) error {
	err := models.IncrementJobRetry(q.db, jobID)
	if err != nil {
		return fmt.Errorf("failed to increment job retry: %w", err)
	}
	log.Printf("[JobQueue] Job %d retry count incremented", jobID)
	return nil
}

// Get retrieves a job by ID
func (q *DatabaseJobQueue) Get(ctx context.Context, jobID int) (*models.LLMJob, error) {
	job, err := models.GetJob(q.db, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

// List retrieves jobs for a user with optional filtering
func (q *DatabaseJobQueue) List(ctx context.Context, params models.JobListParams) ([]models.LLMJob, error) {
	jobs, err := models.ListJobs(q.db, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	return jobs, nil
}

// Cancel cancels a pending job
func (q *DatabaseJobQueue) Cancel(ctx context.Context, jobID, userID int) error {
	err := models.CancelJob(q.db, jobID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("job not found, already processed, or not owned by user")
		}
		return fmt.Errorf("failed to cancel job: %w", err)
	}
	log.Printf("[JobQueue] Job %d cancelled by user %d", jobID, userID)
	return nil
}

// Stats retrieves statistics for a user
func (q *DatabaseJobQueue) Stats(ctx context.Context, userID int) (*models.JobStats, error) {
	stats, err := models.GetJobStats(q.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	return stats, nil
}

// GetQueueDepth returns the number of pending jobs in the queue
func (q *DatabaseJobQueue) GetQueueDepth(ctx context.Context) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM llm_jobs WHERE status = 'pending'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth: %w", err)
	}
	return count, nil
}

// UpdateHeartbeat updates the last_heartbeat timestamp for a running job
func (q *DatabaseJobQueue) UpdateHeartbeat(ctx context.Context, jobID int) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE llm_jobs SET last_heartbeat = NOW() WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat for job %d: %w", jobID, err)
	}
	return nil
}

// MarkStuckJobsFailed finds jobs that have exceeded timeout and marks them as failed
// Returns the number of jobs marked as failed
func (q *DatabaseJobQueue) MarkStuckJobsFailed(ctx context.Context) (int, error) {
	// Mark jobs as failed if:
	// 1. No heartbeat for too long (stuck process) - 120 seconds
	// 2. Exceeded timeout since start
	result, err := q.db.ExecContext(ctx,
		`UPDATE llm_jobs
		 SET status = 'failed',
			 error_message = 'Job exceeded timeout or stopped responding',
			 completed_at = NOW()
		 WHERE status = 'running'
		   AND (
			 -- No heartbeat for too long (stuck process)
			 (last_heartbeat IS NOT NULL AND EXTRACT(EPOCH FROM (NOW() - last_heartbeat)) > 120)
			 OR
			 -- Exceeded timeout since start
			 EXTRACT(EPOCH FROM (NOW() - started_at)) > timeout_seconds
		   )`)
	if err != nil {
		return 0, fmt.Errorf("failed to mark stuck jobs: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// CleanupOrphanedJobs marks jobs stuck in "running" state as failed after server restart
// Returns the number of jobs cleaned up
func (q *DatabaseJobQueue) CleanupOrphanedJobs(ctx context.Context) (int, error) {
	// Mark jobs as failed if they've been "running" since before server start
	// and haven't had a heartbeat recently
	result, err := q.db.ExecContext(ctx,
		`UPDATE llm_jobs
		 SET status = 'failed',
			 error_message = 'Job orphaned by server restart',
			 completed_at = NOW()
		 WHERE status = 'running'
		   AND (
			 -- No heartbeat in last 5 minutes (likely dead)
			 last_heartbeat < NOW() - INTERVAL '5 minutes'
			 OR
			 -- No heartbeat at all but started more than 10 minutes ago
			 (last_heartbeat IS NULL AND started_at < NOW() - INTERVAL '10 minutes')
		   )`)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup orphaned jobs: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// Helper function to check if a job should be retried
func ShouldRetryJob(job *models.LLMJob) bool {
	return job.RetryCount < job.MaxRetries
}

// CalculateBackoff calculates exponential backoff delay for retries
func CalculateBackoff(retryCount int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, etc.
	// Cap at 60 seconds
	backoff := time.Duration(1<<uint(retryCount)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}
	return backoff
}

// GetJobTimeout returns the timeout duration for a job
func GetJobTimeout(job *models.LLMJob) time.Duration {
	return time.Duration(job.TimeoutSecs) * time.Second
}

// TypeFilteredJobQueue wraps a JobQueue and only dequeues jobs of specific types
// This allows different worker pools to share the same underlying queue without
// picking up jobs meant for other workers.
type TypeFilteredJobQueue struct {
	queue     JobQueue
	jobTypes  []models.JobType
}

// NewTypeFilteredJobQueue creates a new filtered job queue that only returns jobs of the specified types
func NewTypeFilteredJobQueue(queue JobQueue, jobTypes ...models.JobType) JobQueue {
	return &TypeFilteredJobQueue{
		queue:    queue,
		jobTypes: jobTypes,
	}
}

func (q *TypeFilteredJobQueue) Enqueue(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error) {
	return q.queue.Enqueue(ctx, params)
}

func (q *TypeFilteredJobQueue) Dequeue(ctx context.Context) (*models.LLMJob, error) {
	return q.queue.DequeueByTypes(ctx, q.jobTypes)
}

func (q *TypeFilteredJobQueue) DequeueByTypes(ctx context.Context, jobTypes []models.JobType) (*models.LLMJob, error) {
	// Intersection of our types and requested types
	// For simplicity, just use our configured types
	return q.queue.DequeueByTypes(ctx, q.jobTypes)
}

func (q *TypeFilteredJobQueue) UpdateStatus(ctx context.Context, jobID int, status models.JobStatus) error {
	return q.queue.UpdateStatus(ctx, jobID, status)
}

func (q *TypeFilteredJobQueue) UpdateStatusWithResult(ctx context.Context, jobID int, status models.JobStatus, result map[string]interface{}) error {
	return q.queue.UpdateStatusWithResult(ctx, jobID, status, result)
}

func (q *TypeFilteredJobQueue) UpdateStatusWithError(ctx context.Context, jobID int, status models.JobStatus, errorMsg string) error {
	return q.queue.UpdateStatusWithError(ctx, jobID, status, errorMsg)
}

func (q *TypeFilteredJobQueue) MarkRunning(ctx context.Context, jobID int) error {
	return q.queue.MarkRunning(ctx, jobID)
}

func (q *TypeFilteredJobQueue) IncrementRetry(ctx context.Context, jobID int) error {
	return q.queue.IncrementRetry(ctx, jobID)
}

func (q *TypeFilteredJobQueue) Get(ctx context.Context, jobID int) (*models.LLMJob, error) {
	return q.queue.Get(ctx, jobID)
}

func (q *TypeFilteredJobQueue) List(ctx context.Context, params models.JobListParams) ([]models.LLMJob, error) {
	return q.queue.List(ctx, params)
}

func (q *TypeFilteredJobQueue) Cancel(ctx context.Context, jobID, userID int) error {
	return q.queue.Cancel(ctx, jobID, userID)
}

func (q *TypeFilteredJobQueue) Stats(ctx context.Context, userID int) (*models.JobStats, error) {
	return q.queue.Stats(ctx, userID)
}

func (q *TypeFilteredJobQueue) GetQueueDepth(ctx context.Context) (int, error) {
	return q.queue.GetQueueDepth(ctx)
}

func (q *TypeFilteredJobQueue) UpdateHeartbeat(ctx context.Context, jobID int) error {
	return q.queue.UpdateHeartbeat(ctx, jobID)
}

func (q *TypeFilteredJobQueue) MarkStuckJobsFailed(ctx context.Context) (int, error) {
	return q.queue.MarkStuckJobsFailed(ctx)
}

func (q *TypeFilteredJobQueue) CleanupOrphanedJobs(ctx context.Context) (int, error) {
	return q.queue.CleanupOrphanedJobs(ctx)
}
