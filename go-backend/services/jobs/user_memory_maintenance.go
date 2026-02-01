package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/bootstrap"
	"go-backend/handlers"
	"go-backend/pkg/config"
	"go-backend/server"
	"go-backend/services"
)

// UserMemoryMaintenanceJob processes user memory compression for users with memory_has_changed flag
type UserMemoryMaintenanceJob struct {
	db       *sql.DB
	schedule string
}

// NewUserMemoryMaintenanceJob creates a new user memory maintenance job
func NewUserMemoryMaintenanceJob(db *sql.DB) *UserMemoryMaintenanceJob {
	return &UserMemoryMaintenanceJob{db: db}
}

// Name returns the unique identifier for this job
func (j *UserMemoryMaintenanceJob) Name() string {
	return "user-memory-maintenance"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *UserMemoryMaintenanceJob) Schedule() string {
	return "0 0 * * * *" // Run at the top of every hour (seconds, minutes, hours, day, month, weekday)
}

// MaxRetries returns the number of times to retry on failure
func (j *UserMemoryMaintenanceJob) MaxRetries() int {
	return 2
}

// NextRun returns the next scheduled run time for this job
func (j *UserMemoryMaintenanceJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// processUserMemory processes memory compression for a single user
func (j *UserMemoryMaintenanceJob) processUserMemory(ctx context.Context, s *server.Server, userID uint) error {
	client := services.NewDefaultClient(s.DB, int(userID), false)
	client.RequestType = "memory"

	result, err := handlers.CompressUserMemory(s.DB, client, userID)
	if err != nil {
		log.Printf("[memory-maintenance-job] failed to compress memory for user %d: %v", userID, err)
		return err
	}
	log.Printf("[memory-maintenance-job] compression result for user %d: %s", userID, result)

	// Reset the memory_has_changed flag
	_, err = s.DB.ExecContext(ctx, "UPDATE users SET memory_has_changed = false WHERE id = $1", userID)
	if err != nil {
		log.Printf("[memory-maintenance-job] failed to reset memory_has_changed flag for user %d: %v", userID, err)
		return err
	}

	return nil
}

// Handler executes the user memory maintenance job logic
func (j *UserMemoryMaintenanceJob) Handler(ctx context.Context) error {
	log.Println("[memory-maintenance-job] starting user memory maintenance")

	if j.db == nil {
		log.Println("[memory-maintenance-job] no database configured, skipping")
		return nil
	}

	// Query for users with memory_has_changed = true
	rows, err := j.db.QueryContext(ctx, "SELECT id FROM users WHERE memory_has_changed = true")
	if err != nil {
		log.Printf("[memory-maintenance-job] failed to query users: %v", err)
		return err
	}
	defer rows.Close()

	// Get a minimal server instance (we need it for LLM client operations)
	cfg := config.LoadConfig()
	s := bootstrap.InitServer(cfg.Database)

	userCount := 0
	errorCount := 0

	for rows.Next() {
		var userID uint
		if err := rows.Scan(&userID); err != nil {
			log.Printf("[memory-maintenance-job] failed to scan user ID: %v", err)
			errorCount++
			continue
		}

		if err := j.processUserMemory(ctx, s, userID); err != nil {
			errorCount++
		}
		userCount++
	}

	if err := rows.Err(); err != nil {
		log.Printf("[memory-maintenance-job] rows iteration error: %v", err)
		return err
	}

	log.Printf("[memory-maintenance-job] completed: processed %d users, %d errors", userCount, errorCount)
	return nil
}

// Verify UserMemoryMaintenanceJob implements ScheduledJob interface
var _ services.ScheduledJob = (*UserMemoryMaintenanceJob)(nil)
