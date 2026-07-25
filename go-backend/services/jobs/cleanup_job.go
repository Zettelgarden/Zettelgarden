package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/models"
	"go-backend/services"
)

// CleanupJob performs daily cleanup of old data
type CleanupJob struct {
	db               *sql.DB
	schedule         string
	llmRetentionDays int
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(db *sql.DB) *CleanupJob {
	// Get retention days from environment or use default
	retentionDays := 30
	if val := os.Getenv("JOB_RETENTION_DAYS"); val != "" {
		var days int
		if _, err := fmt.Sscanf(val, "%d", &days); err == nil && days > 0 {
			retentionDays = days
		}
	}

	return &CleanupJob{
		db:               db,
		llmRetentionDays: retentionDays,
	}
}

// Name returns the unique identifier for this job
func (j *CleanupJob) Name() string {
	return "daily-cleanup"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *CleanupJob) Schedule() string {
	return "0 0 2 * * *" // Run at 2 AM daily (seconds, minutes, hours, day, month, weekday)
}

// MaxRetries returns the number of times to retry on failure
func (j *CleanupJob) MaxRetries() int {
	return 3
}

// NextRun returns the next scheduled run time for this job
func (j *CleanupJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the cleanup job logic
func (j *CleanupJob) Handler(ctx context.Context) error {
	log.Println("[cleanup-job] starting daily cleanup")

	if j.db == nil {
		log.Println("[cleanup-job] no database configured, skipping")
		return nil
	}

	// Clean up old scheduled_job_runs (keep last 90 days). App-side cutoff
	// because SQLite has no INTERVAL. See migration design P3.
	scheduledCutoff := time.Now().UTC().AddDate(0, 0, -90)
	scheduledRunsResult, err := j.db.ExecContext(ctx,
		"DELETE FROM scheduled_job_runs WHERE created_at < $1", scheduledCutoff)
	if err != nil {
		log.Printf("[cleanup-job] failed to clean old scheduled job runs: %v", err)
		return err
	}
	scheduledRunsDeleted, _ := scheduledRunsResult.RowsAffected()
	log.Printf("[cleanup-job] cleaned up %d old scheduled_job_runs (retention: 90 days)", scheduledRunsDeleted)

	// Clean up old llm_jobs (keep last N days, default 30)
	llmJobsDeleted, err := models.CleanupOldJobs(j.db, j.llmRetentionDays)
	if err != nil {
		log.Printf("[cleanup-job] failed to clean old llm_jobs: %v", err)
		return err
	}
	log.Printf("[cleanup-job] cleaned up %d old llm_jobs (retention: %d days)", llmJobsDeleted, j.llmRetentionDays)

	log.Println("[cleanup-job] completed successfully")
	return nil
}

// Verify CleanupJob implements ScheduledJob interface
var _ services.ScheduledJob = (*CleanupJob)(nil)
