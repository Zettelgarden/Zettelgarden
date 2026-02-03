package jobs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
)

// UptimeKumaPingJob sends heartbeat POST requests to Uptime Kuma
// to verify the job scheduler is operational
type UptimeKumaPingJob struct {
	pushURL string
}

// NewUptimeKumaPingJob creates a new uptime kuma ping job
func NewUptimeKumaPingJob() *UptimeKumaPingJob {
	return &UptimeKumaPingJob{
		pushURL: os.Getenv("UPTIME_KUMA_PUSH_URL"),
	}
}

// Name returns the unique identifier for this job
func (j *UptimeKumaPingJob) Name() string {
	return "uptime-kuma-ping"
}

// Schedule returns the cron expression for when this job should run
// Runs every minute
func (j *UptimeKumaPingJob) Schedule() string {
	return "0 * * * * *"
}

// MaxRetries returns the number of times to retry on failure
func (j *UptimeKumaPingJob) MaxRetries() int {
	return 3
}

// NextRun returns the next scheduled run time for this job
func (j *UptimeKumaPingJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the uptime kuma ping job logic
func (j *UptimeKumaPingJob) Handler(ctx context.Context) error {
	log.Println("[uptime-kuma-ping] starting heartbeat")

	if j.pushURL == "" {
		log.Println("[uptime-kuma-ping] UPTIME_KUMA_PUSH_URL not configured, skipping")
		return nil
	}

	// Create request with context for timeout cancellation
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.pushURL, nil)
	if err != nil {
		log.Printf("[uptime-kuma-ping] failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (Uptime Kuma accepts empty body for push)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[uptime-kuma-ping] request failed: %v", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[uptime-kuma-ping] unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Println("[uptime-kuma-ping] completed successfully")
	return nil
}

// Verify UptimeKumaPingJob implements ScheduledJob interface
var _ services.ScheduledJob = (*UptimeKumaPingJob)(nil)
