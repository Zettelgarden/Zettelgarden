package jobs

import (
	"testing"

	"go-backend/services"
)

// TestUptimeKumaPingJobImplementsInterface verifies that UptimeKumaPingJob implements
// the ScheduledJob interface with correct values.
func TestUptimeKumaPingJobImplementsInterface(t *testing.T) {
	job := NewUptimeKumaPingJob()

	// Verify Name returns expected value
	if got, want := job.Name(), "uptime-kuma-ping"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (every minute)
	if got, want := job.Schedule(), "0 * * * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 3; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}
