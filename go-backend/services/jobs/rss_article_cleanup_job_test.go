package jobs

import (
	"context"
	"os"
	"testing"

	"go-backend/services"
)

// TestRSSArticleCleanupJobImplementsInterface verifies that RSSArticleCleanupJob
// implements the ScheduledJob interface with correct values.
func TestRSSArticleCleanupJobImplementsInterface(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)

	// Verify Name returns expected value
	if got, want := job.Name(), "rss-article-cleanup"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (6-field format with seconds)
	if got, want := job.Schedule(), "0 0 3 * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 3; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}

// TestRSSArticleCleanupJobHandler verifies that the handler executes correctly.
func TestRSSArticleCleanupJobHandler(t *testing.T) {
	t.Run("with nil DB should succeed", func(t *testing.T) {
		job := NewRSSArticleCleanupJob(nil)
		ctx := context.Background()

		if err := job.Handler(ctx); err != nil {
			t.Errorf("Handler() with nil DB should succeed, got error: %v", err)
		}
	})
}

// TestRSSArticleCleanupJobDefaultRetention verifies default retention is 30 days.
func TestRSSArticleCleanupJobDefaultRetention(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	if got, want := job.retentionDays, 30; got != want {
		t.Errorf("retentionDays = %d, want %d", got, want)
	}
}

// TestRSSArticleCleanupJobCustomRetention verifies custom retention from env var.
func TestRSSArticleCleanupJobCustomRetention(t *testing.T) {
	os.Setenv("RSS_ARTICLE_RETENTION_DAYS", "60")
	defer os.Unsetenv("RSS_ARTICLE_RETENTION_DAYS")

	job := NewRSSArticleCleanupJob(nil)
	if got, want := job.retentionDays, 60; got != want {
		t.Errorf("retentionDays = %d, want %d", got, want)
	}
}
