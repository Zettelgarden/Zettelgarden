package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/mail"
	"go-backend/services"
)

// TaskRemindersJob sends task reminders to users
type TaskRemindersJob struct {
	db    *sql.DB
	mail  *mail.MailClient
}

// NewTaskRemindersJob creates a new task reminders job
func NewTaskRemindersJob(db *sql.DB, mailClient *mail.MailClient) *TaskRemindersJob {
	return &TaskRemindersJob{
		db:   db,
		mail: mailClient,
	}
}

// Name returns the unique identifier for this job
func (j *TaskRemindersJob) Name() string {
	return "task-reminders"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *TaskRemindersJob) Schedule() string {
	return "0 * * * * *" // Run every minute (seconds, minutes, hours, day, month, weekday)
}

// MaxRetries returns the number of times to retry on failure
func (j *TaskRemindersJob) MaxRetries() int {
	return 1
}

// NextRun returns the next scheduled run time for this job
func (j *TaskRemindersJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the task reminders job logic
func (j *TaskRemindersJob) Handler(ctx context.Context) error {
	if j.db == nil {
		log.Println("[task-reminders-job] no database configured, skipping")
		return nil
	}

	if j.mail == nil {
		log.Println("[task-reminders-job] no mail client configured, skipping")
		return nil
	}

	log.Println("[task-reminders-job] checking for task reminders to send")

	err := services.SendTaskReminders(j.db, j.mail)
	if err != nil {
		log.Printf("[task-reminders-job] error sending reminders: %v", err)
		return err
	}

	log.Println("[task-reminders-job] reminder check complete")
	return nil
}

// Verify TaskRemindersJob implements ScheduledJob interface
var _ services.ScheduledJob = (*TaskRemindersJob)(nil)
