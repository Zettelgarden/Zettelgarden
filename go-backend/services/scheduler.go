package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled jobs using cron
type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]ScheduledJob
	tracker *ScheduledExecutionTracker
	mu      sync.RWMutex
	logger  *log.Logger
}

// NewScheduler creates a new scheduler. If db is nil, tracking is disabled.
func NewScheduler(db *sql.DB) *Scheduler {
	var tracker *ScheduledExecutionTracker
	if db != nil {
		tracker = NewScheduledExecutionTracker(db)
	}

	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		jobs:    make(map[string]ScheduledJob),
		tracker: tracker,
		logger:  log.Default(),
	}
}

// Register adds a job to the scheduler. Returns an error if a job
// with the same name is already registered.
func (s *Scheduler) Register(job ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := job.Name()

	// Check for duplicate job name
	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job with name '%s' is already registered", name)
	}

	// Add the job to cron
	_, err := s.cron.AddFunc(job.Schedule(), func() {
		s.runJob(job)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job '%s': %w", name, err)
	}

	s.jobs[name] = job
	s.logger.Printf("Registered scheduled job: %s (schedule: %s)", name, job.Schedule())

	return nil
}

// Start begins the cron scheduler. Jobs will begin executing
// according to their schedules.
func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Println("Scheduler started")
}

// Stop gracefully shuts down the scheduler with a timeout.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		// All jobs completed
	case <-time.After(30 * time.Second):
		s.logger.Println("[scheduler] WARNING: timeout waiting for jobs to complete")
	}
	s.logger.Println("[scheduler] stopped")
}

// runJob executes a job with retry logic and tracking
func (s *Scheduler) runJob(job ScheduledJob) {
	name := job.Name()
	var runID int64
	var err error

	// Create a context with timeout for the job
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Record job start if tracker is available
	if s.tracker != nil {
		runID, err = s.tracker.RecordStart(ctx, name)
		if err != nil {
			s.logger.Printf("Failed to record job start for '%s': %v", name, err)
		}
	}

	// Execute the job with retry logic
	maxRetries := job.MaxRetries()
	lastErr := error(nil)
	attempt := 0

	for ; maxRetries == -1 || attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Linear backoff: attempt * time.Second
			backoff := time.Duration(attempt) * time.Second
			s.logger.Printf("Job '%s' retry %d, waiting %v before retry", name, attempt, backoff)
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
				// continue with retry
			case <-ctx.Done():
				s.logger.Printf("[scheduler] %s cancelled during retry backoff", name)
				return
			}
		}

		// Execute the handler
		lastErr = job.Handler(ctx)
		if lastErr == nil {
			// Success - record completion
			if s.tracker != nil && runID > 0 {
				if recordErr := s.tracker.RecordCompletion(ctx, runID, nil); recordErr != nil {
					s.logger.Printf("Failed to record job completion for '%s': %v", name, recordErr)
				}
			}
			s.logger.Printf("Job '%s' completed successfully", name)
			return
		}

		s.logger.Printf("Job '%s' attempt %d failed: %v", name, attempt+1, lastErr)
	}

	// All retries exhausted - record failure
	if s.tracker != nil && runID > 0 {
		// Use actual attempt count (number of retries performed)
		actualRetryCount := attempt
		if recordErr := s.tracker.RecordFailure(ctx, runID, lastErr, actualRetryCount); recordErr != nil {
			s.logger.Printf("Failed to record job failure for '%s': %v", name, recordErr)
		}
	}
	s.logger.Printf("Job '%s' failed after %d attempts: %v", name, attempt+1, lastErr)
}

// GetJobHistory returns execution history for a specific job.
// Requires database tracking to be enabled.
func (s *Scheduler) GetJobHistory(ctx context.Context, jobName string, limit int) ([]JobRun, error) {
	if s.tracker == nil {
		return nil, fmt.Errorf("job tracking is not enabled (database not configured)")
	}

	return s.tracker.GetRecentRuns(ctx, jobName, limit)
}

// ListJobs returns the names of all registered jobs
func (s *Scheduler) ListJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.jobs))
	for name := range s.jobs {
		names = append(names, name)
	}
	return names
}
