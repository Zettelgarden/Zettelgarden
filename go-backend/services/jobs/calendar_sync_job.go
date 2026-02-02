package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
)

// CalendarSyncJob periodically syncs external calendars
type CalendarSyncJob struct {
	db                *sql.DB
	externalEventService *services.ExternalEventService
	schedule          string
}

// NewCalendarSyncJob creates a new calendar sync job
func NewCalendarSyncJob(db *sql.DB, externalEventService *services.ExternalEventService) *CalendarSyncJob {
	return &CalendarSyncJob{
		db:                db,
		externalEventService: externalEventService,
	}
}

// Name returns the unique identifier for this job
func (j *CalendarSyncJob) Name() string {
	return "calendar-sync"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
// Runs every hour to check if any calendars need syncing
func (j *CalendarSyncJob) Schedule() string {
	return "0 0 * * * *" // Run at the top of every hour (seconds, minutes, hours, day, month, weekday)
}

// MaxRetries returns the number of times to retry on failure
func (j *CalendarSyncJob) MaxRetries() int {
	return 2
}

// NextRun returns the next scheduled run time for this job
func (j *CalendarSyncJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// Handler executes the calendar sync job logic
func (j *CalendarSyncJob) Handler(ctx context.Context) error {
	log.Println("[calendar-sync-job] starting calendar sync check")

	if j.db == nil {
		log.Println("[calendar-sync-job] no database configured, skipping")
		return nil
	}

	// Query all enabled calendars that need syncing
	// A calendar needs syncing if:
	// 1. sync_enabled is true
	// 2. last_synced_at is NULL (never synced) OR last_synced_at < NOW() - sync_interval_hours
	query := `
		SELECT id, user_id, name, sync_interval_hours
		FROM external_calendars
		WHERE sync_enabled = true
		  AND (last_synced_at IS NULL
		       OR last_synced_at < NOW() - (sync_interval_hours || ' hours')::INTERVAL)
		ORDER BY user_id, id
	`

	rows, err := j.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("[calendar-sync-job] failed to query calendars: %v", err)
		return fmt.Errorf("failed to query calendars: %w", err)
	}
	defer rows.Close()

	type calendarToSync struct {
		ID               int
		UserID           int
		Name             string
		SyncIntervalHours int
	}

	var calendarsToSync []calendarToSync
	for rows.Next() {
		var cal calendarToSync
		if err := rows.Scan(&cal.ID, &cal.UserID, &cal.Name, &cal.SyncIntervalHours); err != nil {
			log.Printf("[calendar-sync-job] failed to scan calendar row: %v", err)
			continue
		}
		calendarsToSync = append(calendarsToSync, cal)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[calendar-sync-job] error iterating calendar rows: %v", err)
		return fmt.Errorf("error iterating calendars: %w", err)
	}

	if len(calendarsToSync) == 0 {
		log.Println("[calendar-sync-job] no calendars need syncing")
		return nil
	}

	log.Printf("[calendar-sync-job] found %d calendars to sync", len(calendarsToSync))

	// Sync each calendar
	successCount := 0
	errorCount := 0
	for _, cal := range calendarsToSync {
		log.Printf("[calendar-sync-job] syncing calendar %d (%s) for user %d (interval: %dh)",
			cal.ID, cal.Name, cal.UserID, cal.SyncIntervalHours)

		err := j.externalEventService.SyncExternalCalendar(cal.ID, cal.UserID)
		if err != nil {
			log.Printf("[calendar-sync-job] failed to sync calendar %d: %v", cal.ID, err)
			errorCount++
		} else {
			log.Printf("[calendar-sync-job] successfully synced calendar %d", cal.ID)
			successCount++
		}
	}

	log.Printf("[calendar-sync-job] completed: %d succeeded, %d failed", successCount, errorCount)

	// Only return error if all syncs failed (partial success is acceptable)
	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("all %d calendar syncs failed", errorCount)
	}

	return nil
}

// Verify CalendarSyncJob implements ScheduledJob interface
var _ services.ScheduledJob = (*CalendarSyncJob)(nil)
