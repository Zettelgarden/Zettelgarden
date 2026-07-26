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

	// Delete old articles that are not starred and not converted to cards.
	// App-side cutoff because SQLite has no INTERVAL. See migration design P3.
	articleCutoff := time.Now().UTC().AddDate(0, 0, -j.retentionDays)

	// Notifications were trigger-maintained (0122/0124); now maintained in Go
	// (Phase 5). Remove notifications for the articles about to be deleted —
	// this MUST run before the article DELETE because the selection is over
	// rss_articles. Cross-driver: IN (subquery) works on Postgres and SQLite.
	if _, err := j.db.ExecContext(ctx, `
		DELETE FROM notifications
		WHERE source_type = 'rss'
		  AND source_id IN (
		    SELECT id FROM rss_articles
		    WHERE fetched_at < $1 AND is_starred = false AND card_id IS NULL
		  )
	`, articleCutoff); err != nil {
		log.Printf("[rss-article-cleanup] failed to delete notifications: %v", err)
		// Non-fatal: still delete the articles.
	}

	result, err := j.db.ExecContext(ctx, `
		DELETE FROM rss_articles
		WHERE fetched_at < $1
		  AND is_starred = false
		  AND card_id IS NULL
	`, articleCutoff)
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
