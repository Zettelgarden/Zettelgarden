package services

import "context"

// ScheduledJob defines the interface for jobs that can be scheduled
// for periodic execution using cron syntax.
type ScheduledJob interface {
	// Name returns a unique identifier for this job
	Name() string

	// Schedule returns the cron expression for when this job should run.
	// Supports standard cron (5 fields) or extended (6 fields with seconds).
	// Examples:
	//   "0 * * * *"       - every hour
	//   "*/5 * * * *"     - every 5 minutes
	//   "0 0 * * *"       - daily at midnight
	//   "0 9 * * 1-5"     - weekdays at 9am
	Schedule() string

	// Handler executes the job logic. The context will be cancelled
	// if the job exceeds its timeout or the server is shutting down.
	Handler(ctx context.Context) error

	// MaxRetries returns the number of times to retry on failure.
	// Use 0 for no retries, -1 for infinite retries.
	MaxRetries() int
}
