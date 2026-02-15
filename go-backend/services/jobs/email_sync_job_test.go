package jobs

import (
	"context"
	"testing"

	"go-backend/services"
)

// TestEmailSyncJobImplementsInterface verifies that EmailSyncJob implements
// the ScheduledJob interface with correct values.
func TestEmailSyncJobImplementsInterface(t *testing.T) {
	job := NewEmailSyncJob(nil)

	// Verify Name returns expected value
	if got, want := job.Name(), "email-sync"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (6-field format with seconds)
	// Should run every 60 minutes
	if got, want := job.Schedule(), "0 */60 * * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 3; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}

// TestEmailSyncJobHandler verifies that the handler executes correctly.
func TestEmailSyncJobHandler(t *testing.T) {
	t.Run("with nil DB should succeed", func(t *testing.T) {
		job := NewEmailSyncJob(nil)
		ctx := context.Background()

		if err := job.Handler(ctx); err != nil {
			t.Errorf("Handler() with nil DB should succeed, got error: %v", err)
		}
	})
}
