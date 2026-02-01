package jobs

import (
	"context"
	"database/sql"
	"log"

	"go-backend/services"
)

// CleanupJob performs daily cleanup of old data
type CleanupJob struct {
	db *sql.DB
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(db *sql.DB) *CleanupJob {
	return &CleanupJob{db: db}
}

// Name returns the unique identifier for this job
func (j *CleanupJob) Name() string {
	return "daily-cleanup"
}

// Schedule returns the cron expression for when this job should run
func (j *CleanupJob) Schedule() string {
	return "0 2 * * *" // Run at 2 AM daily
}

// MaxRetries returns the number of times to retry on failure
func (j *CleanupJob) MaxRetries() int {
	return 3
}

// Handler executes the cleanup job logic
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

// Verify CleanupJob implements ScheduledJob interface
var _ services.ScheduledJob = (*CleanupJob)(nil)
