package jobs

import (
	"context"
	"testing"

	"go-backend/services"
)

// TestCalendarSyncJobImplementsInterface verifies that CalendarSyncJob implements
// the ScheduledJob interface with correct values.
func TestCalendarSyncJobImplementsInterface(t *testing.T) {
	job := NewCalendarSyncJob(nil, nil)

	// Verify Name returns expected value
	if got, want := job.Name(), "calendar-sync"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (6-field format with seconds)
	// Should run at the top of every hour
	if got, want := job.Schedule(), "0 0 * * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 2; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}

// TestCalendarSyncJobHandler verifies that the handler executes correctly.
func TestCalendarSyncJobHandler(t *testing.T) {
	t.Run("with nil DB should succeed", func(t *testing.T) {
		job := NewCalendarSyncJob(nil, nil)
		ctx := context.Background()

		if err := job.Handler(ctx); err != nil {
			t.Errorf("Handler() with nil DB should succeed, got error: %v", err)
		}
	})
}
