package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
	"go-backend/settings"
)

// RSSArticleCleanupJob cleans up old RSS articles
type RSSArticleCleanupJob struct {
	db       *sql.DB
	settings *settings.Manager
}

// NewRSSArticleCleanupJob creates a new RSS article cleanup job. The settings
// manager is optional (nil falls back to the registry default, 30 days);
// retention is read per-run so admin UI edits hot-reload without a restart.
func NewRSSArticleCleanupJob(db *sql.DB, sm *settings.Manager) *RSSArticleCleanupJob {
	return &RSSArticleCleanupJob{
		db:       db,
		settings: sm,
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
	// Retention is admin-tunable via rss_article_retention_days in config.yaml.
	retentionDays := 30
	if j.settings != nil {
		retentionDays = j.settings.GetInt("rss_article_retention_days", retentionDays)
	}
	articleCutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)

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
	log.Printf("[rss-article-cleanup] cleaned up %d old articles (retention: %d days)", articlesDeleted, retentionDays)

	return nil
}

// Verify RSSArticleCleanupJob implements ScheduledJob interface
var _ services.ScheduledJob = (*RSSArticleCleanupJob)(nil)
