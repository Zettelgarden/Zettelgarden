package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/models"
	"go-backend/services"
	"go-backend/settings"
)

// CleanupJob performs daily cleanup of old data
type CleanupJob struct {
	db       *sql.DB
	schedule string
	settings *settings.Manager
}

// NewCleanupJob creates a new cleanup job. The settings manager is optional:
// when nil, retention falls back to the registry default (30 days). Retention
// is read per-run from the settings manager so admin UI edits hot-reload
// without a restart.
func NewCleanupJob(db *sql.DB, sm *settings.Manager) *CleanupJob {
	return &CleanupJob{
		db:       db,
		settings: sm,
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

	// Clean up old llm_jobs (keep last N days, default 30; admin-tunable via
	// job_retention_days in config.yaml).
	retentionDays := 30
	if j.settings != nil {
		retentionDays = j.settings.GetInt("job_retention_days", retentionDays)
	}
	llmJobsDeleted, err := models.CleanupOldJobs(j.db, retentionDays)
	if err != nil {
		log.Printf("[cleanup-job] failed to clean old llm_jobs: %v", err)
		return err
	}
	log.Printf("[cleanup-job] cleaned up %d old llm_jobs (retention: %d days)", llmJobsDeleted, retentionDays)

	log.Println("[cleanup-job] completed successfully")
	return nil
}

// Verify CleanupJob implements ScheduledJob interface
var _ services.ScheduledJob = (*CleanupJob)(nil)
