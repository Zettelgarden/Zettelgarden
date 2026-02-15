package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// MinSyncIntervalHours is the minimum allowed sync interval (1 hour)
	MinSyncIntervalHours = 1
	// MaxSyncIntervalHours is the maximum allowed sync interval (7 days)
	MaxSyncIntervalHours = 168
	// SyncCooldownMinutes is the minimum time between sync operations
	SyncCooldownMinutes = 5
	// MaxICalFeedSizeBytes is the maximum allowed size for an iCal feed response
	MaxICalFeedSizeBytes = 10 * 1024 * 1024 // 10MB
)

// Global testing flag for URL validation bypass in tests
// This should only be set in test environments
var testingMu sync.RWMutex
var testingMode bool

// SetExternalEventsTestingMode sets the global testing mode for external events
func SetExternalEventsTestingMode(testing bool) {
	testingMu.Lock()
	defer testingMu.Unlock()
	testingMode = testing
}

// isTestingMode returns true if we're in testing mode
func isTestingMode() bool {
	testingMu.RLock()
	defer testingMu.RUnlock()
	return testingMode
}

// ExternalEventService handles external calendar event operations
type ExternalEventService struct {
	db                models.Database
	encryptionService *EncryptionService
}

// isValidHexColor checks if a color string is a valid hex color code
func isValidHexColor(color string) bool {
	if color == "" {
		return false
	}
	matched, _ := regexp.MatchString("^#[0-9a-fA-F]{6}$", color)
	return matched
}

// validateSyncIntervalHours checks if the sync interval is within valid bounds
func validateSyncIntervalHours(hours *int) error {
	if hours == nil {
		return nil // nil is valid, means no change
	}
	if *hours < MinSyncIntervalHours || *hours > MaxSyncIntervalHours {
		return fmt.Errorf("sync_interval_hours must be between %d and %d", MinSyncIntervalHours, MaxSyncIntervalHours)
	}
	return nil
}

// NewExternalEventService creates a new ExternalEventService instance
func NewExternalEventService(db models.Database, encryptionService *EncryptionService) *ExternalEventService {
	return &ExternalEventService{db: db, encryptionService: encryptionService}
}

// SyncExternalCalendar fetches and imports events from an external calendar
func (s *ExternalEventService) SyncExternalCalendar(calendarID int, userID int) error {
	log.Printf("[Sync] Starting sync for calendar %d (user %d)", calendarID, userID)

	// Get calendar details with last_synced_at for cooldown check
	var url string
	var color string
	var lastSyncedAt sql.NullTime
	var username sql.NullString
	var encryptedPassword sql.NullString
	err := s.db.QueryRow(`
		SELECT url, color, last_synced_at, username, password
		FROM external_calendars
		WHERE id = $1 AND user_id = $2
	`, calendarID, userID).Scan(&url, &color, &lastSyncedAt, &username, &encryptedPassword)

	if err == sql.ErrNoRows {
		log.Printf("[Sync] Calendar %d not found for user %d", calendarID, userID)
		return fmt.Errorf("calendar not found")
	}
	if err != nil {
		log.Printf("[Sync] Failed to get calendar %d: %v", calendarID, err)
		return fmt.Errorf("failed to get calendar: %w", err)
	}

	// Check sync cooldown
	if lastSyncedAt.Valid {
		timeSinceLastSync := time.Since(lastSyncedAt.Time)
		cooldownDuration := time.Duration(SyncCooldownMinutes) * time.Minute
		if timeSinceLastSync < cooldownDuration {
			remaining := cooldownDuration - timeSinceLastSync
			log.Printf("[Sync] Calendar %d is in cooldown, %.0f minutes remaining", calendarID, remaining.Minutes())
			return fmt.Errorf("sync cooldown active, please wait %.0f minutes", remaining.Minutes())
		}
	}

	log.Printf("[Sync] Fetching events from iCal feed for calendar %d", calendarID)
	log.Printf("[Sync] URL: %s, Username: %s", url, username.String)

	// Decrypt password if provided
	var password string
	if encryptedPassword.Valid && s.encryptionService != nil {
		decrypted, err := s.encryptionService.Decrypt(encryptedPassword.String)
		if err != nil {
			// Log generic message to avoid side-channel information leakage
			log.Printf("[Sync] Failed to decrypt password for calendar %d", calendarID)
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		password = decrypted
		log.Printf("[Sync] Decrypted password length: %d (first 3 chars: %s***)", len(password), string(password[:3]))
	}

	// Fetch events and todos from iCal feed with credentials
	icalEvents, icalTodos, err := FetchICalURLWithTodos(url, username.String, password)
	if err != nil {
		log.Printf("[Sync] Failed to fetch iCal feed for calendar %d: %v", calendarID, err)
		s.UpdateLastSyncError(calendarID, err.Error())
		return fmt.Errorf("failed to fetch iCal feed: %w", err)
	}

	log.Printf("[Sync] Fetched %d events and %d todos from iCal feed for calendar %d", len(icalEvents), len(icalTodos), calendarID)

	// Import events
	successCount := 0
	errorCount := 0
	for _, event := range icalEvents {
		err := s.importEvent(calendarID, userID, event, color)
		if err != nil {
			log.Printf("[Sync] Failed to import event %s: %v", event.UID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	// Import todos
	todoSuccessCount := 0
	todoErrorCount := 0
	for _, todo := range icalTodos {
		err := s.importTodo(calendarID, userID, todo)
		if err != nil {
			log.Printf("[Sync] Failed to import todo %s: %v", todo.UID, err)
			todoErrorCount++
		} else {
			todoSuccessCount++
		}
	}

	log.Printf("[Sync] Synced calendar %d: %d events imported, %d todos imported, %d total errors", calendarID, successCount, todoSuccessCount, successCount+errorCount+todoSuccessCount+todoErrorCount)

	// Update last synced timestamp and clear error
	s.UpdateLastSynced(calendarID)
	return nil
}

// CreateEventOnCalendar creates a new event via CalDAV and returns the UID
func (s *ExternalEventService) CreateEventOnCalendar(calendarID, userID int, req models.CreateEventRequest) (string, error) {
	// Get calendar details
	cal, err := s.GetCalendar(calendarID, userID)
	if err != nil {
		return "", fmt.Errorf("calendar not found: %w", err)
	}

	// Check if calendar has credentials (writable)
	if cal.Username == nil || *cal.Username == "" {
		return "", fmt.Errorf("calendar does not have credentials configured for write access")
	}

	// Get the encrypted password from database
	var encryptedPassword sql.NullString
	err = s.db.QueryRow(`
		SELECT password
		FROM external_calendars
		WHERE id = $1 AND user_id = $2
	`, calendarID, userID).Scan(&encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to get calendar credentials: %w", err)
	}

	// Decrypt password if present
	password := ""
	if encryptedPassword.Valid && encryptedPassword.String != "" {
		password, err = s.encryptionService.Decrypt(encryptedPassword.String)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt calendar credentials: %w", err)
		}
	}

	// Create CalDAV client
	username := *cal.Username
	caldavSvc, err := NewCalDAVEventService(cal.URL, username, password)
	if err != nil {
		return "", fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	// Create the event via CalDAV
	uid, err := caldavSvc.CreateEvent(context.Background(), cal.URL, req)
	if err != nil {
		return "", fmt.Errorf("failed to create event on CalDAV server: %w", err)
	}

	return uid, nil
}

// importEvent imports a single event, handling upsert logic
func (s *ExternalEventService) importEvent(calendarID, userID int, event ICalEvent, defaultColor string) error {
	// If this event has a recurrence rule, we need to expand it
	if event.RecurrenceRule != "" {
		return s.importRecurringEvent(calendarID, userID, event, defaultColor)
	}

	// For non-recurring events, check if event already exists
	var existingID int
	var existingColor sql.NullString
	err := s.db.QueryRow(`
		SELECT id, color
		FROM external_events
		WHERE user_id = $1 AND external_uid = $2
	`, userID, event.UID).Scan(&existingID, &existingColor)

	if err == sql.ErrNoRows {
		// Insert new event
		_, err = s.db.Exec(`
			INSERT INTO external_events (
				user_id, external_calendar_id, title, description,
				start_time, end_time, all_day, location,
				external_uid, external_url, recurrence_rule, color,
				last_synced_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, userID, calendarID, event.Summary, nullString(event.Description),
			event.DTStart, event.DTEnd, event.AllDay, nullString(event.Location),
			event.UID, nullString(event.URL), nullString(event.RecurrenceRule), defaultColor)
		return err
	} else if err == nil {
		// Update existing event (preserve event-specific color if set)
		color := existingColor.String
		if !existingColor.Valid {
			color = defaultColor
		}

		_, err = s.db.Exec(`
			UPDATE external_events SET
				title = $1, description = $2, start_time = $3, end_time = $4,
				all_day = $5, location = $6, external_url = $7,
				recurrence_rule = $8, color = $9,
				updated_at = NOW(), last_synced_at = NOW()
			WHERE id = $10
		`, event.Summary, nullString(event.Description), event.DTStart, event.DTEnd,
			event.AllDay, nullString(event.Location), nullString(event.URL),
			nullString(event.RecurrenceRule), color, existingID)
		return err
	}

	return err
}

// importTodo imports a single todo as a task, handling upsert logic
func (s *ExternalEventService) importTodo(calendarID, userID int, todo ICalTodo) error {
	// Skip todos without a summary (required field)
	if todo.Summary == "" {
		log.Printf("[Sync] Skipping todo %s: no summary", todo.UID)
		return nil
	}

	// Skip todos with recurrence - not supported in this phase
	if todo.Summary == "" {
		log.Printf("[Sync] Skipping todo %s: no summary", todo.UID)
		return nil
	}

	// Check if task already exists by external_uid
	var existingID int
	var existingStatus sql.NullString
	var existingPriority sql.NullString
	var existingIsComplete bool
	err := s.db.QueryRow(`
		SELECT id, status, priority, is_complete
		FROM tasks
		WHERE user_id = $1 AND external_uid = $2 AND is_deleted = FALSE
	`, userID, todo.UID).Scan(&existingID, &existingStatus, &existingPriority, &existingIsComplete)

	if err == sql.ErrNoRows {
		// Insert new task
		return s.insertTaskFromTodo(calendarID, userID, todo)
	} else if err == nil {
		// Update existing task (preserve status and priority per requirements)
		return s.updateTaskFromTodo(existingID, userID, todo, existingStatus.String, existingPriority.String, existingIsComplete)
	}

	return err
}

// insertTaskFromTodo creates a new task from a VTODO
func (s *ExternalEventService) insertTaskFromTodo(calendarID, userID int, todo ICalTodo) error {
	// Map VTODO status to task status
	status, isComplete := mapTodoStatus(todo.Status, todo.PercentComplete)

	// Map due date - use DUE if available, otherwise DTSTART as scheduled_date
	var dueDate *time.Time
	if !todo.Due.IsZero() {
		dueDate = &todo.Due
	}

	var scheduledDate *time.Time
	if !todo.DTStart.IsZero() && todo.Due.IsZero() {
		scheduledDate = &todo.DTStart
	}

	// Map completed timestamp
	var completedAt *time.Time
	if !todo.Completed.IsZero() {
		completedAt = &todo.Completed
	} else if isComplete {
		now := time.Now()
		completedAt = &now
	}

	// Title is required
	title := todo.Summary
	if title == "" {
		title = "(No title)"
	}

	_, err := s.db.Exec(`
		INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, status, is_complete, is_deleted, external_uid, external_calendar_id)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7, $8, $9, FALSE, $10, $11)
	`, nil, userID, scheduledDate, dueDate, completedAt, title, nullString(todo.Description), status, isComplete, todo.UID, calendarID)

	if err != nil {
		log.Printf("[Sync] Failed to insert task from todo %s: %v", todo.UID, err)
		return fmt.Errorf("failed to insert task: %w", err)
	}

	log.Printf("[Sync] Created task from todo %s: %s", todo.UID, title)
	return nil
}

// updateTaskFromTodo updates an existing task from VTODO, preserving status and priority
func (s *ExternalEventService) updateTaskFromTodo(taskID, userID int, todo ICalTodo, existingStatus, existingPriority string, existingIsComplete bool) error {
	// Map VTODO status to task status
	_, sourceIsComplete := mapTodoStatus(todo.Status, todo.PercentComplete)

	// Map due date - use DUE if available, otherwise DTSTART as scheduled_date
	var dueDate *time.Time
	if !todo.Due.IsZero() {
		dueDate = &todo.Due
	}

	var scheduledDate *time.Time
	if !todo.DTStart.IsZero() && todo.Due.IsZero() {
		scheduledDate = &todo.DTStart
	}

	// Preserve user's status and priority choices (per requirements)
	status := existingStatus
	priority := existingPriority
	isComplete := existingIsComplete

	// Only update completion status if the source says completed and task wasn't already complete
	if sourceIsComplete && !existingIsComplete {
		isComplete = true
		// Also update status to user's "done" status if possible
		defaultStatus, err := GetDefaultTaskStatus(s.db, userID)
		if err == nil {
			status = defaultStatus.Name
		}
	}

	// Update completed_at if source says completed but we don't have a completed_at
	var completedAt *time.Time
	if sourceIsComplete && !todo.Completed.IsZero() {
		completedAt = &todo.Completed
	} else if sourceIsComplete {
		// STATUS=COMPLETED but no COMPLETED timestamp - use current time
		now := time.Now()
		completedAt = &now
	}

	// Preserve existing completed_at if task was already complete locally
	if existingIsComplete {
		completedAt = nil // Don't overwrite existing completion
	}

	// Title is required
	title := todo.Summary
	if title == "" {
		title = "(No title)"
	}

	_, err := s.db.Exec(`
		UPDATE tasks SET
			title = $1,
			description = $2,
			scheduled_date = $3,
			due_date = $4,
			status = $5,
			priority = $6,
			is_complete = $7,
			completed_at = COALESCE($8, completed_at),
			updated_at = NOW()
		WHERE id = $9 AND user_id = $10
	`, title, nullString(todo.Description), scheduledDate, dueDate, status, priority, isComplete, completedAt, taskID, userID)

	if err != nil {
		log.Printf("[Sync] Failed to update task from todo %s: %v", todo.UID, err)
		return fmt.Errorf("failed to update task: %w", err)
	}

	log.Printf("[Sync] Updated task from todo %s: %s", todo.UID, title)
	return nil
}

// mapTodoStatus maps VTODO STATUS and PERCENT-COMPLETE to task status
// Returns (status string, isComplete bool)
func mapTodoStatus(vtodoStatus string, percentComplete int) (string, bool) {
	// Check percent-complete first (overrides status)
	if percentComplete >= 100 {
		return "done", true
	}

	// Map VTODO status values
	switch strings.ToUpper(vtodoStatus) {
	case "COMPLETED":
		return "done", true
	case "CANCELLED":
		// Treat cancelled as deleted - skip import
		return "", false
	case "IN-PROCESS", "IN-PROGRESS":
		return "in-progress", false
	case "NEEDS-ACTION":
		return "todo", false
	default:
		// Unknown or empty status - treat as todo
		return "todo", false
	}
}

// importRecurringEvent expands and imports a recurring event
func (s *ExternalEventService) importRecurringEvent(calendarID, userID int, event ICalEvent, defaultColor string) error {
	log.Printf("[Recurrence] Expanding recurring event %s with RRULE: %s", event.UID, event.RecurrenceRule)

	// Expand the recurrence rule into occurrences
	occurrences, err := ExpandRecurrence(event.RecurrenceRule, event.DTStart, event.DTEnd, event.AllDay)
	if err != nil {
		log.Printf("[Recurrence] Failed to expand RRULE for event %s: %v", event.UID, err)
		// Fall back to storing just the base event
		return s.importSingleEvent(calendarID, userID, event, defaultColor)
	}

	log.Printf("[Recurrence] Generated %d occurrences for event %s", len(occurrences), event.UID)

	// Clean up old instances of this recurring series
	s.cleanupOldRecurringInstances(userID, event.UID)

	// Import each occurrence
	for _, occ := range occurrences {
		instanceUID := GetInstanceUID(event.UID, occ.Index)

		// Check if this instance already exists
		var existingID int
		var existingColor sql.NullString
		err := s.db.QueryRow(`
			SELECT id, color
			FROM external_events
			WHERE user_id = $1 AND external_uid = $2
		`, userID, instanceUID).Scan(&existingID, &existingColor)

		if err == sql.ErrNoRows {
			// Insert new instance
			_, err = s.db.Exec(`
				INSERT INTO external_events (
					user_id, external_calendar_id, title, description,
					start_time, end_time, all_day, location,
					external_uid, external_url, recurrence_rule, recurrence_id, recurrence_instance, color,
					last_synced_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
			`, userID, calendarID, event.Summary, nullString(event.Description),
				occ.StartTime, occ.EndTime, event.AllDay, nullString(event.Location),
				instanceUID, nullString(event.URL), nullString(event.RecurrenceRule),
				GetRecurrenceID(event.UID), occ.Index, defaultColor)
			if err != nil {
				log.Printf("[Recurrence] Failed to insert instance %s: %v", instanceUID, err)
			}
		} else if err == nil {
			// Update existing instance (preserve event-specific color if set)
			color := existingColor.String
			if !existingColor.Valid {
				color = defaultColor
			}

			_, err = s.db.Exec(`
				UPDATE external_events SET
					title = $1, description = $2, start_time = $3, end_time = $4,
					all_day = $5, location = $6, external_url = $7,
					recurrence_rule = $8, color = $9,
					updated_at = NOW(), last_synced_at = NOW()
				WHERE id = $10
			`, event.Summary, nullString(event.Description), occ.StartTime, occ.EndTime,
				event.AllDay, nullString(event.Location), nullString(event.URL),
				nullString(event.RecurrenceRule), color, existingID)
			if err != nil {
				log.Printf("[Recurrence] Failed to update instance %s: %v", instanceUID, err)
			}
		} else {
			log.Printf("[Recurrence] Error checking for instance %s: %v", instanceUID, err)
		}
	}

	return nil
}

// importSingleEvent imports a single event (fallback for recurrence expansion failures)
func (s *ExternalEventService) importSingleEvent(calendarID, userID int, event ICalEvent, defaultColor string) error {
	var existingID int
	var existingColor sql.NullString
	err := s.db.QueryRow(`
		SELECT id, color
		FROM external_events
		WHERE user_id = $1 AND external_uid = $2
	`, userID, event.UID).Scan(&existingID, &existingColor)

	if err == sql.ErrNoRows {
		// Insert new event
		_, err = s.db.Exec(`
			INSERT INTO external_events (
				user_id, external_calendar_id, title, description,
				start_time, end_time, all_day, location,
				external_uid, external_url, recurrence_rule, color,
				last_synced_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, userID, calendarID, event.Summary, nullString(event.Description),
			event.DTStart, event.DTEnd, event.AllDay, nullString(event.Location),
			event.UID, nullString(event.URL), nullString(event.RecurrenceRule), defaultColor)
		return err
	} else if err == nil {
		// Update existing event (preserve event-specific color if set)
		color := existingColor.String
		if !existingColor.Valid {
			color = defaultColor
		}

		_, err = s.db.Exec(`
			UPDATE external_events SET
				title = $1, description = $2, start_time = $3, end_time = $4,
				all_day = $5, location = $6, external_url = $7,
				recurrence_rule = $8, color = $9,
				updated_at = NOW(), last_synced_at = NOW()
			WHERE id = $10
		`, event.Summary, nullString(event.Description), event.DTStart, event.DTEnd,
			event.AllDay, nullString(event.Location), nullString(event.URL),
			nullString(event.RecurrenceRule), color, existingID)
		return err
	}

	return err
}

// cleanupOldRecurringInstances removes old instances of a recurring event series
// that are no longer in the current expansion (e.g., past occurrences older than 30 days)
func (s *ExternalEventService) cleanupOldRecurringInstances(userID int, baseUID string) {
	recurrenceID := GetRecurrenceID(baseUID)

	// Delete instances that are more than 60 days old and not manually modified
	// We keep a buffer to avoid deleting instances that might still be relevant
	_, err := s.db.Exec(`
		DELETE FROM external_events
		WHERE user_id = $1
		  AND recurrence_id = $2
		  AND start_time < NOW() - INTERVAL '60 days'
		  AND card_pk IS NULL  -- Don't delete events that are linked to cards
	`, userID, recurrenceID)

	if err != nil {
		log.Printf("[Recurrence] Failed to cleanup old instances for %s: %v", baseUID, err)
	} else {
		log.Printf("[Recurrence] Cleaned up old instances for %s", baseUID)
	}
}

// GetEventsInRange returns events for a user within a date range with pagination
func (s *ExternalEventService) GetEventsInRange(userID int, start, end time.Time, limit, offset int) ([]models.ExternalEvent, int, error) {
	// First, get the total count
	var total int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM external_events
		WHERE user_id = $1
		  AND start_time >= $2
		  AND end_time <= $3
	`, userID, start, end).Scan(&total)

	if err != nil {
		return nil, 0, err
	}

	// Set default pagination values
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit to prevent excessive results
	}

	query := `
		SELECT e.id, e.user_id, e.external_calendar_id, e.title, e.description,
		       e.start_time, e.end_time, e.all_day, e.location,
		       e.external_uid, e.external_url, e.recurrence_rule,
		       e.recurrence_id, e.recurrence_instance,
		       COALESCE(ec.color, e.color) as color,
		       e.card_pk, e.created_at, e.updated_at, e.last_synced_at
		FROM external_events e
		LEFT JOIN external_calendars ec ON e.external_calendar_id = ec.id
		WHERE e.user_id = $1
		  AND e.start_time >= $2
		  AND e.end_time <= $3
		ORDER BY e.start_time
		LIMIT $4 OFFSET $5
	`

	rows, err := s.db.Query(query, userID, start, end, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.ExternalEvent
	for rows.Next() {
		var event models.ExternalEvent
		var description, location, externalUID, externalURL, recurrenceRule, recurrenceID, color sql.NullString
		var externalCalendarID sql.NullInt32
		var cardPK sql.NullInt32
		var recurrenceInstance sql.NullInt32
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.UserID, &externalCalendarID,
			&event.Title, &description,
			&event.StartTime, &event.EndTime, &event.AllDay, &location,
			&externalUID, &externalURL, &recurrenceRule, &recurrenceID, &recurrenceInstance,
			&color,
			&cardPK,
			&event.CreatedAt, &event.UpdatedAt, &lastSyncedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		// Convert null fields
		if description.Valid {
			event.Description = &description.String
		}
		if location.Valid {
			event.Location = &location.String
		}
		if externalUID.Valid {
			event.ExternalUID = &externalUID.String
		}
		if externalURL.Valid {
			event.ExternalURL = &externalURL.String
		}
		if recurrenceRule.Valid {
			event.RecurrenceRule = &recurrenceRule.String
		}
		if recurrenceID.Valid {
			event.RecurrenceID = &recurrenceID.String
		}
		if recurrenceInstance.Valid {
			instance := int(recurrenceInstance.Int32)
			event.RecurrenceInstance = &instance
		}
		if color.Valid {
			event.Color = &color.String
		}
		if externalCalendarID.Valid {
			id := int(externalCalendarID.Int32)
			event.ExternalCalendarID = &id
		}
		if cardPK.Valid {
			pk := int(cardPK.Int32)
			event.CardPK = &pk
		}
		if lastSyncedAt.Valid {
			event.LastSyncedAt = &lastSyncedAt.Time
		}

		// Load linked card info if present
		if event.CardPK != nil {
			card, err := GetPartialCard(s.db, userID, *event.CardPK)
			if err == nil {
				event.Card = card
			}
		}

		events = append(events, event)
	}

	return events, total, nil
}

// GetCalendars returns all external calendars for a user
func (s *ExternalEventService) GetCalendars(userID int) ([]models.ExternalCalendar, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, name, url, sync_enabled, sync_interval_hours,
		       color, last_synced_at, last_error, created_at, updated_at,
		       username
		FROM external_calendars
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []models.ExternalCalendar
	for rows.Next() {
		var cal models.ExternalCalendar
		var lastSyncedAt sql.NullTime
		var lastError sql.NullString
		var username sql.NullString

		err := rows.Scan(
			&cal.ID, &cal.UserID, &cal.Name, &cal.URL,
			&cal.SyncEnabled, &cal.SyncIntervalHours, &cal.Color,
			&lastSyncedAt, &lastError, &cal.CreatedAt, &cal.UpdatedAt,
			&username,
		)
		if err != nil {
			return nil, err
		}

		if lastSyncedAt.Valid {
			cal.LastSyncedAt = &lastSyncedAt.Time
		}
		if lastError.Valid {
			cal.LastError = &lastError.String
		}
		if username.Valid {
			cal.Username = &username.String
		}

		calendars = append(calendars, cal)
	}

	return calendars, nil
}

// CreateCalendar creates a new external calendar subscription
func (s *ExternalEventService) CreateCalendar(userID int, req models.CreateExternalCalendarRequest) (*models.ExternalCalendar, error) {
	// Validate URL by attempting to fetch it with credentials
	username := ""
	if req.Username != nil {
		username = *req.Username
	}
	password := ""
	if req.Password != nil {
		password = *req.Password
	}

	// Skip URL validation in testing mode
	if !isTestingMode() {
		if err := ValidateICalURL(req.URL, username, password); err != nil {
			return nil, fmt.Errorf("invalid iCal URL: %w", err)
		}
	}

	// Validate and set color
	color := req.Color
	if color == "" {
		color = "#6366f1" // Default indigo
	} else if !isValidHexColor(color) {
		return nil, fmt.Errorf("invalid color format: must be a hex color code (e.g., #6366f1)")
	}

	// Encrypt password if provided
	var encryptedPassword *string
	if req.Password != nil && *req.Password != "" {
		if s.encryptionService == nil {
			return nil, fmt.Errorf("encryption service not available")
		}
		encrypted, err := s.encryptionService.Encrypt(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		encryptedPassword = &encrypted
	}

	var id int
	var usernameVal interface{} = nil
	if req.Username != nil {
		usernameVal = *req.Username
	}
	var passwordVal interface{} = nil
	if encryptedPassword != nil {
		passwordVal = *encryptedPassword
	}

	err := s.db.QueryRow(`
		INSERT INTO external_calendars (user_id, name, url, color, username, password)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, req.Name, req.URL, color, usernameVal, passwordVal).Scan(&id)

	if err != nil {
		return nil, err
	}

	// Fetch and return the created calendar
	return s.GetCalendar(id, userID)
}

// GetCalendar returns a single calendar by ID
func (s *ExternalEventService) GetCalendar(calendarID, userID int) (*models.ExternalCalendar, error) {
	var cal models.ExternalCalendar
	var lastSyncedAt sql.NullTime
	var lastError sql.NullString
	var username sql.NullString

	err := s.db.QueryRow(`
		SELECT id, user_id, name, url, sync_enabled, sync_interval_hours,
		       color, last_synced_at, last_error, created_at, updated_at,
		       username
		FROM external_calendars
		WHERE id = $1 AND user_id = $2
	`, calendarID, userID).Scan(
		&cal.ID, &cal.UserID, &cal.Name, &cal.URL,
		&cal.SyncEnabled, &cal.SyncIntervalHours, &cal.Color,
		&lastSyncedAt, &lastError, &cal.CreatedAt, &cal.UpdatedAt,
		&username,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("calendar not found")
	}
	if err != nil {
		return nil, err
	}

	if lastSyncedAt.Valid {
		cal.LastSyncedAt = &lastSyncedAt.Time
	}
	if lastError.Valid {
		cal.LastError = &lastError.String
	}
	if username.Valid {
		cal.Username = &username.String
	}

	return &cal, nil
}

// UpdateCalendar updates an existing calendar
func (s *ExternalEventService) UpdateCalendar(calendarID, userID int, req models.UpdateExternalCalendarRequest) error {
	// Validate sync_interval_hours if provided
	if err := validateSyncIntervalHours(req.SyncIntervalHours); err != nil {
		return err
	}

	// Validate color if provided
	if req.Color != nil && *req.Color != "" && !isValidHexColor(*req.Color) {
		return fmt.Errorf("invalid color format: must be a hex color code (e.g., #6366f1)")
	}

	// Validate URL if provided (re-validate to prevent SSRF bypass)
	// Skip URL validation in testing mode
	if req.URL != nil && !isTestingMode() {
		username := ""
		if req.Username != nil {
			username = *req.Username
		}
		password := ""
		if req.Password != nil {
			password = *req.Password
		}
		if err := ValidateICalURL(*req.URL, username, password); err != nil {
			return fmt.Errorf("invalid iCal URL: %w", err)
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}
	if req.URL != nil {
		updates = append(updates, fmt.Sprintf("url = $%d", argPos))
		args = append(args, *req.URL)
		argPos++
	}
	if req.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argPos))
		args = append(args, *req.Color)
		argPos++
	}
	if req.SyncEnabled != nil {
		updates = append(updates, fmt.Sprintf("sync_enabled = $%d", argPos))
		args = append(args, *req.SyncEnabled)
		argPos++
	}
	if req.SyncIntervalHours != nil {
		updates = append(updates, fmt.Sprintf("sync_interval_hours = $%d", argPos))
		args = append(args, *req.SyncIntervalHours)
		argPos++
	}
	if req.Username != nil {
		updates = append(updates, fmt.Sprintf("username = $%d", argPos))
		args = append(args, *req.Username)
		argPos++
	}
	if req.ClearPassword != nil && *req.ClearPassword {
		// Explicitly clear the password
		updates = append(updates, fmt.Sprintf("password = $%d", argPos))
		args = append(args, nil)
		argPos++
	} else if req.Password != nil && *req.Password != "" {
		// Encrypt and set new password
		if s.encryptionService == nil {
			return fmt.Errorf("encryption service not available")
		}
		encrypted, err := s.encryptionService.Encrypt(*req.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		updates = append(updates, fmt.Sprintf("password = $%d", argPos))
		args = append(args, encrypted)
		argPos++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	updates = append(updates, fmt.Sprintf("updated_at = NOW()"))
	args = append(args, calendarID, userID)

	query := fmt.Sprintf("UPDATE external_calendars SET %s WHERE id = $%d AND user_id = $%d",
		strings.Join(updates, ", "), argPos, argPos+1)

	_, err := s.db.Exec(query, args...)
	return err
}

// DeleteCalendar deletes a calendar and all its events
func (s *ExternalEventService) DeleteCalendar(calendarID, userID int) error {
	result, err := s.db.Exec("DELETE FROM external_calendars WHERE id = $1 AND user_id = $2", calendarID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("calendar not found")
	}

	return nil
}

// UpdateLastSynced updates the last_synced_at timestamp and clears error
func (s *ExternalEventService) UpdateLastSynced(calendarID int) {
	_, err := s.db.Exec(`
		UPDATE external_calendars
		SET last_synced_at = NOW(), last_error = NULL
		WHERE id = $1
	`, calendarID)
	if err != nil {
		log.Printf("Failed to update last_synced_at for calendar %d: %v", calendarID, err)
	}
}

// UpdateLastSyncError records an error message for a calendar
func (s *ExternalEventService) UpdateLastSyncError(calendarID int, errMsg string) {
	// Truncate error message to prevent db bloat
	if len(errMsg) > 500 {
		errMsg = errMsg[:500] + "..."
	}

	_, err := s.db.Exec(`
		UPDATE external_calendars
		SET last_error = $1
		WHERE id = $2
	`, errMsg, calendarID)
	if err != nil {
		log.Printf("Failed to update last_error for calendar %d: %v", calendarID, err)
	}
}

// nullString converts a string to *string, returning nil for empty strings
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// LinkEventToCard links an external event to a card
func (s *ExternalEventService) LinkEventToCard(db models.Database, userID int, eventID int, cardPK int) (*models.ExternalEvent, error) {
	// Verify the event belongs to the user
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM external_events WHERE id = $1 AND user_id = $2)
	`, eventID, userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("event not found")
	}

	// Verify the card belongs to the user
	err = db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM cards WHERE id = $1 AND user_id = $2)
	`, cardPK, userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("card not found")
	}

	// Update the event with the card link
	_, err = db.Exec(`
		UPDATE external_events
		SET card_pk = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`, cardPK, eventID, userID)
	if err != nil {
		return nil, err
	}

	// Fetch and return the updated event
	return s.getEventByID(db, eventID, userID)
}

// UnlinkEventFromCard unlinks an external event from its card
func (s *ExternalEventService) UnlinkEventFromCard(db models.Database, userID int, eventID int) error {
	// Verify the event belongs to the user
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM external_events WHERE id = $1 AND user_id = $2)
	`, eventID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("event not found")
	}

	// Clear the card link
	_, err = db.Exec(`
		UPDATE external_events
		SET card_pk = NULL, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, eventID, userID)
	return err
}

// GetEventsByCard returns all external events linked to a specific card
func (s *ExternalEventService) GetEventsByCard(db models.Database, userID int, cardPK int) ([]models.ExternalEvent, error) {
	query := `
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule, recurrence_id, recurrence_instance,
		       color, card_pk,
		       created_at, updated_at, last_synced_at
		FROM external_events
		WHERE user_id = $1 AND card_pk = $2
		ORDER BY start_time DESC
	`

	rows, err := db.Query(query, userID, cardPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.ExternalEvent
	for rows.Next() {
		var event models.ExternalEvent
		var description, location, externalUID, externalURL, recurrenceRule, recurrenceID, color sql.NullString
		var externalCalendarID sql.NullInt32
		var scannedCardPK sql.NullInt32
		var recurrenceInstance sql.NullInt32
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.UserID, &externalCalendarID,
			&event.Title, &description,
			&event.StartTime, &event.EndTime, &event.AllDay, &location,
			&externalUID, &externalURL, &recurrenceRule, &recurrenceID, &recurrenceInstance,
			&color,
			&scannedCardPK,
			&event.CreatedAt, &event.UpdatedAt, &lastSyncedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert null fields
		if description.Valid {
			event.Description = &description.String
		}
		if location.Valid {
			event.Location = &location.String
		}
		if externalUID.Valid {
			event.ExternalUID = &externalUID.String
		}
		if externalURL.Valid {
			event.ExternalURL = &externalURL.String
		}
		if recurrenceRule.Valid {
			event.RecurrenceRule = &recurrenceRule.String
		}
		if recurrenceID.Valid {
			event.RecurrenceID = &recurrenceID.String
		}
		if recurrenceInstance.Valid {
			instance := int(recurrenceInstance.Int32)
			event.RecurrenceInstance = &instance
		}
		if color.Valid {
			event.Color = &color.String
		}
		if externalCalendarID.Valid {
			id := int(externalCalendarID.Int32)
			event.ExternalCalendarID = &id
		}
		if scannedCardPK.Valid {
			pk := int(scannedCardPK.Int32)
			event.CardPK = &pk
		}
		if lastSyncedAt.Valid {
			event.LastSyncedAt = &lastSyncedAt.Time
		}

		// Load linked card info if present
		if event.CardPK != nil {
			card, err := GetPartialCard(db, userID, *event.CardPK)
			if err == nil {
				event.Card = card
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// getEventByID fetches a single event by ID with card info
func (s *ExternalEventService) getEventByID(db models.Database, eventID int, userID int) (*models.ExternalEvent, error) {
	var event models.ExternalEvent
	var description, location, externalUID, externalURL, recurrenceRule, recurrenceID, color sql.NullString
	var externalCalendarID sql.NullInt32
	var cardPK sql.NullInt32
	var recurrenceInstance sql.NullInt32
	var lastSyncedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule, recurrence_id, recurrence_instance,
		       color, card_pk,
		       created_at, updated_at, last_synced_at
		FROM external_events
		WHERE id = $1 AND user_id = $2
	`, eventID, userID).Scan(
		&event.ID, &event.UserID, &externalCalendarID,
		&event.Title, &description,
		&event.StartTime, &event.EndTime, &event.AllDay, &location,
		&externalUID, &externalURL, &recurrenceRule, &recurrenceID, &recurrenceInstance,
		&color,
		&cardPK,
		&event.CreatedAt, &event.UpdatedAt, &lastSyncedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, err
	}

	// Convert null fields
	if description.Valid {
		event.Description = &description.String
	}
	if location.Valid {
		event.Location = &location.String
	}
	if externalUID.Valid {
		event.ExternalUID = &externalUID.String
	}
	if externalURL.Valid {
		event.ExternalURL = &externalURL.String
	}
	if recurrenceRule.Valid {
		event.RecurrenceRule = &recurrenceRule.String
	}
	if recurrenceID.Valid {
		event.RecurrenceID = &recurrenceID.String
	}
	if recurrenceInstance.Valid {
		instance := int(recurrenceInstance.Int32)
		event.RecurrenceInstance = &instance
	}
	if color.Valid {
		event.Color = &color.String
	}
	if externalCalendarID.Valid {
		id := int(externalCalendarID.Int32)
		event.ExternalCalendarID = &id
	}
	if cardPK.Valid {
		pk := int(cardPK.Int32)
		event.CardPK = &pk
	}
	if lastSyncedAt.Valid {
		event.LastSyncedAt = &lastSyncedAt.Time
	}

	// Load linked card info if present
	if event.CardPK != nil {
		card, err := GetPartialCard(db, userID, *event.CardPK)
		if err == nil {
			event.Card = card
		}
	}

	return &event, nil
}

