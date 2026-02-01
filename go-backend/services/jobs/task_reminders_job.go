package jobs

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/bootstrap"
	"go-backend/mail"
	"go-backend/pkg/config"
	"go-backend/services"
)

// TaskRemindersJob sends task reminders to users
type TaskRemindersJob struct {
	db       *sql.DB
	schedule string
}

// NewTaskRemindersJob creates a new task reminders job
func NewTaskRemindersJob(db *sql.DB) *TaskRemindersJob {
	return &TaskRemindersJob{db: db}
}

// Name returns the unique identifier for this job
func (j *TaskRemindersJob) Name() string {
	return "task-reminders"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *TaskRemindersJob) Schedule() string {
	return "* * * * * *" // Run every minute (seconds, minutes, hours, day, month, weekday)
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

	// Initialize server with mail client
	cfg := config.LoadConfig()
	s := bootstrap.InitServer(cfg.Database)

	// Set up mail client from environment
	s.Mail = &mail.MailClient{
		Host:     os.Getenv("MAIL_HOST"),
		Password: os.Getenv("MAIL_PASSWORD"),
		Queue:    mail.NewEmailQueue(),
		DB:       s.DB,
	}

	log.Println("[task-reminders-job] checking for task reminders to send")

	err := services.SendTaskReminders(s.DB, s.Mail)
	if err != nil {
		log.Printf("[task-reminders-job] error sending reminders: %v", err)
		return err
	}

	log.Println("[task-reminders-job] reminder check complete")
	return nil
}

// Verify TaskRemindersJob implements ScheduledJob interface
var _ services.ScheduledJob = (*TaskRemindersJob)(nil)
