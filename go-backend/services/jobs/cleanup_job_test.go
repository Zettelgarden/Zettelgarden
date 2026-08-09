package jobs

import (
	"context"
	"testing"

	"go-backend/services"
)

// TestCleanupJobImplementsInterface verifies that CleanupJob implements
// the ScheduledJob interface with correct values.
func TestCleanupJobImplementsInterface(t *testing.T) {
	job := NewCleanupJob(nil, nil)

	// Verify Name returns expected value
	if got, want := job.Name(), "daily-cleanup"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (6-field format with seconds)
	if got, want := job.Schedule(), "0 0 2 * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 3; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}

// TestCleanupJobHandler verifies that the handler executes correctly.
func TestCleanupJobHandler(t *testing.T) {
	t.Run("with nil DB should succeed", func(t *testing.T) {
		job := NewCleanupJob(nil, nil)
		ctx := context.Background()

		if err := job.Handler(ctx); err != nil {
			t.Errorf("Handler() with nil DB should succeed, got error: %v", err)
		}
	})
}
