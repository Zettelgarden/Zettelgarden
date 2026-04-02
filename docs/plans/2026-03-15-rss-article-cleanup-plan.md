# RSS Article Cleanup Job Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a scheduled job that deletes old RSS articles while protecting starred and converted articles.

**Architecture:** Create a new `RSSArticleCleanupJob` following the existing `ScheduledJob` interface pattern. The job runs daily at 3 AM and deletes articles older than the configured retention period that are neither starred nor converted to cards.

**Tech Stack:** Go, PostgreSQL, cron scheduling via robfig/cron

---

### Task 1: Write the Job Implementation

**Files:**
- Create: `go-backend/services/jobs/rss_article_cleanup_job.go`

**Step 1: Create the job file**

```go
package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
)

// RSSArticleCleanupJob cleans up old RSS articles
type RSSArticleCleanupJob struct {
	db            *sql.DB
	retentionDays int
}

// NewRSSArticleCleanupJob creates a new RSS article cleanup job
func NewRSSArticleCleanupJob(db *sql.DB) *RSSArticleCleanupJob {
	// Get retention days from environment or use default (30 days)
	retentionDays := 30
	if val := os.Getenv("RSS_ARTICLE_RETENTION_DAYS"); val != "" {
		var days int
		if _, err := fmt.Sscanf(val, "%d", &days); err == nil && days > 0 {
			retentionDays = days
		}
	}

	return &RSSArticleCleanupJob{
		db:            db,
		retentionDays: retentionDays,
	}
}

// Name returns the unique identifier for this job
func (j *RSSArticleCleanupJob) Name() string {
	return "rss-article-cleanup"
}

// Schedule returns the cron expression for when this job should run
// Runs daily at 3 AM (seconds, minutes, hours, day, month, weekday)
func (j *RSSArticleCleanupJob) Schedule() string {
	return "0 0 3 * * *"
}

// MaxRetries returns the number of times to retry on failure
func (j *RSSArticleCleanupJob) MaxRetries() int {
	return 3
}

// NextRun returns the next scheduled run time for this job
func (j *RSSArticleCleanupJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the RSS article cleanup job logic
func (j *RSSArticleCleanupJob) Handler(ctx context.Context) error {
	log.Println("[rss-article-cleanup] starting RSS article cleanup")

	if j.db == nil {
		log.Println("[rss-article-cleanup] no database configured, skipping")
		return nil
	}

	// Delete old articles that are not starred and not converted to cards
	result, err := j.db.ExecContext(ctx, `
		DELETE FROM rss_articles
		WHERE fetched_at < NOW() - INTERVAL '1 day' * $1
		  AND is_starred = false
		  AND card_id IS NULL
	`, j.retentionDays)
	if err != nil {
		log.Printf("[rss-article-cleanup] failed to delete old articles: %v", err)
		return err
	}

	articlesDeleted, _ := result.RowsAffected()
	log.Printf("[rss-article-cleanup] cleaned up %d old articles (retention: %d days)", articlesDeleted, j.retentionDays)

	return nil
}

// Verify RSSArticleCleanupJob implements ScheduledJob interface
var _ services.ScheduledJob = (*RSSArticleCleanupJob)(nil)
```

**Step 2: Verify the file compiles**

Run: `cd /home/nick/code/Zettelgarden/go-backend && go build ./...`
Expected: No errors

---

### Task 2: Register the Job in main.go

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Add the job registration**

Find the line:
```go
scheduler.Register(jobs.NewRSSFetchJob(s.DB))
```

Add after it:
```go
scheduler.Register(jobs.NewRSSArticleCleanupJob(s.DB))
```

**Step 2: Verify the build**

Run: `cd /home/nick/code/Zettelgarden/go-backend && go build ./...`
Expected: No errors

---

### Task 3: Write Tests

**Files:**
- Create: `go-backend/services/jobs/rss_article_cleanup_job_test.go`

**Step 1: Create the test file**

```go
package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"go-backend/models"
)

func TestRSSArticleCleanupJob_Name(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	if job.Name() != "rss-article-cleanup" {
		t.Errorf("expected name 'rss-article-cleanup', got '%s'", job.Name())
	}
}

func TestRSSArticleCleanupJob_Schedule(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	if job.Schedule() != "0 0 3 * * *" {
		t.Errorf("expected schedule '0 0 3 * * *', got '%s'", job.Schedule())
	}
}

func TestRSSArticleCleanupJob_MaxRetries(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	if job.MaxRetries() != 3 {
		t.Errorf("expected max retries 3, got %d", job.MaxRetries())
	}
}

func TestRSSArticleCleanupJob_NextRun(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	// Test that NextRun returns a valid future time
	from := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	next := job.NextRun(from)
	if next.Before(from) {
		t.Errorf("expected next run to be after 'from' time")
	}
}

func TestRSSArticleCleanupJob_DefaultRetention(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	if job.retentionDays != 30 {
		t.Errorf("expected default retention days 30, got %d", job.retentionDays)
	}
}

func TestRSSArticleCleanupJob_CustomRetention(t *testing.T) {
	os.Setenv("RSS_ARTICLE_RETENTION_DAYS", "60")
	defer os.Unsetenv("RSS_ARTICLE_RETENTION_DAYS")

	job := NewRSSArticleCleanupJob(nil)
	if job.retentionDays != 60 {
		t.Errorf("expected retention days 60, got %d", job.retentionDays)
	}
}

func TestRSSArticleCleanupJob_HandlerNilDB(t *testing.T) {
	job := NewRSSArticleCleanupJob(nil)
	err := job.Handler(context.Background())
	if err != nil {
		t.Errorf("expected no error with nil db, got %v", err)
	}
}

func TestRSSArticleCleanupJob_HandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test database
	db, cleanup := models.TestDB()
	defer cleanup()

	job := NewRSSArticleCleanupJob(db)

	// Create a test user first
	var userID int
	err := db.QueryRow(`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"test-cleanup@example.com", "hash").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create a test feed
	var feedID int
	err = db.QueryRow(`INSERT INTO rss_feeds (user_id, url, name) VALUES ($1, $2, $3) RETURNING id`,
		userID, "https://example.com/feed.xml", "Test Feed").Scan(&feedID)
	if err != nil {
		t.Fatalf("failed to create test feed: %v", err)
	}

	// Insert test articles with different states
	// 1. Old article (should be deleted)
	db.Exec(`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id)
		VALUES ($1, $2, 'Old article', 'https://example.com/old', NOW() - INTERVAL '40 days', false, NULL)`,
		userID, feedID)

	// 2. Old starred article (should be kept)
	db.Exec(`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id)
		VALUES ($1, $2, 'Old starred', 'https://example.com/starred', NOW() - INTERVAL '40 days', true, NULL)`,
		userID, feedID)

	// 3. Old converted article (should be kept)
	db.Exec(`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id)
		VALUES ($1, $2, 'Old converted', 'https://example.com/converted', NOW() - INTERVAL '40 days', false, 1)`,
		userID, feedID)

	// 4. Recent article (should be kept)
	db.Exec(`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id)
		VALUES ($1, $2, 'Recent article', 'https://example.com/recent', NOW() - INTERVAL '10 days', false, NULL)`,
		userID, feedID)

	// Run the cleanup job
	err = job.Handler(context.Background())
	if err != nil {
		t.Fatalf("cleanup job failed: %v", err)
	}

	// Verify: only 3 articles should remain (starred, converted, recent)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM rss_articles WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count articles: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 articles to remain, got %d", count)
	}
}
```

**Step 2: Run the tests**

Run: `cd /home/nick/code/Zettelgarden/go-backend && go test ./services/jobs/... -v`
Expected: All tests pass

---

### Task 4: Final Verification and Commit

**Step 1: Run all tests**

Run: `cd /home/nick/code/Zettelgarden/go-backend && go test ./...`
Expected: All tests pass

**Step 2: Commit**

```bash
git add go-backend/services/jobs/rss_article_cleanup_job.go
git add go-backend/services/jobs/rss_article_cleanup_job_test.go
git add go-backend/main.go
git commit -m "feat: add RSS article cleanup scheduled job

Adds a daily cleanup job that deletes RSS articles older than 30 days
(default, configurable via RSS_ARTICLE_RETENTION_DAYS). Protects starred
articles and articles converted to cards from deletion."
```

---

## Summary

| Task | Files |
|------|-------|
| 1. Job Implementation | Create `services/jobs/rss_article_cleanup_job.go` |
| 2. Registration | Modify `main.go` |
| 3. Tests | Create `services/jobs/rss_article_cleanup_job_test.go` |
| 4. Verify & Commit | Run tests, commit |
