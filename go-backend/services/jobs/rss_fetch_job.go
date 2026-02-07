package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
)

// RSSFetchJob fetches articles from enabled RSS feeds
type RSSFetchJob struct {
	db       *sql.DB
	schedule string
}

// NewRSSFetchJob creates a new RSS fetch job
func NewRSSFetchJob(db *sql.DB) *RSSFetchJob {
	return &RSSFetchJob{
		db:       db,
		schedule: "0 */60 * * * *", // Every 60 minutes
	}
}

// Name returns the unique identifier for this job
func (j *RSSFetchJob) Name() string {
	return "rss-fetch"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *RSSFetchJob) Schedule() string {
	return j.schedule
}

// MaxRetries returns the number of times to retry on failure
func (j *RSSFetchJob) MaxRetries() int {
	return 3
}

// NextRun returns the next scheduled run time for this job
func (j *RSSFetchJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the RSS fetch job logic
func (j *RSSFetchJob) Handler(ctx context.Context) error {
	log.Println("[rss-fetch] starting RSS fetch job")

	if j.db == nil {
		log.Println("[rss-fetch] no database configured, skipping")
		return nil
	}

	// Get enabled RSS feeds
	rows, err := j.db.QueryContext(ctx, `
		SELECT id
		FROM rss_feeds
		WHERE enabled = true
	`)
	if err != nil {
		log.Printf("[rss-fetch] failed to fetch feeds: %v", err)
		return err
	}
	defer rows.Close()

	var feedIDs []int
	for rows.Next() {
		var feedID int
		if err := rows.Scan(&feedID); err != nil {
			log.Printf("[rss-fetch] failed to scan feed ID: %v", err)
			continue
		}
		feedIDs = append(feedIDs, feedID)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[rss-fetch] error iterating feeds: %v", err)
		return err
	}

	// Fetch from each feed
	totalFeeds := 0
	for _, feedID := range feedIDs {
		if err := services.FetchRSSFeedArticles(j.db, feedID); err != nil {
			log.Printf("[rss-fetch] failed to fetch feed %d: %v", feedID, err)
		}
		totalFeeds++
	}

	log.Printf("[rss-fetch] completed, processed %d feeds", totalFeeds)
	return nil
}

// Verify RSSFetchJob implements ScheduledJob interface
var _ services.ScheduledJob = (*RSSFetchJob)(nil)
