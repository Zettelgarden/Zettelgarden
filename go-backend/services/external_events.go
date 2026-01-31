package services

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"strings"
	"time"
)

// ExternalEventService handles external calendar event operations
type ExternalEventService struct {
	db models.Database
}

// NewExternalEventService creates a new ExternalEventService instance
func NewExternalEventService(db models.Database) *ExternalEventService {
	return &ExternalEventService{db: db}
}

// SyncExternalCalendar fetches and imports events from an external calendar
func (s *ExternalEventService) SyncExternalCalendar(calendarID int, userID int) error {
	// Get calendar details
	var url string
	var color string
	err := s.db.QueryRow(`
		SELECT url, color
		FROM external_calendars
		WHERE id = $1 AND user_id = $2
	`, calendarID, userID).Scan(&url, &color)

	if err == sql.ErrNoRows {
		return fmt.Errorf("calendar not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get calendar: %w", err)
	}

	// Fetch events from iCal feed
	icalEvents, err := FetchICalURL(url)
	if err != nil {
		s.UpdateLastSyncError(calendarID, err.Error())
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	// Import events
	successCount := 0
	errorCount := 0
	for _, event := range icalEvents {
		err := s.importEvent(calendarID, userID, event, color)
		if err != nil {
			log.Printf("Failed to import event %s: %v", event.UID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	log.Printf("Synced calendar %d: %d events imported, %d errors", calendarID, successCount, errorCount)

	// Update last synced timestamp and clear error
	s.UpdateLastSynced(calendarID)
	return nil
}

// importEvent imports a single event, handling upsert logic
func (s *ExternalEventService) importEvent(calendarID, userID int, event ICalEvent, defaultColor string) error {
	// Check if event already exists
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

// GetEventsInRange returns events for a user within a date range
func (s *ExternalEventService) GetEventsInRange(userID int, start, end time.Time) ([]models.ExternalEvent, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule, color,
		       created_at, updated_at, last_synced_at
		FROM external_events
		WHERE user_id = $1
		  AND start_time >= $2
		  AND end_time <= $3
		ORDER BY start_time
	`, userID, start, end)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.ExternalEvent
	for rows.Next() {
		var event models.ExternalEvent
		var description, location, externalUID, externalURL, recurrenceRule, color sql.NullString
		var externalCalendarID sql.NullInt32
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.UserID, &externalCalendarID,
			&event.Title, &description,
			&event.StartTime, &event.EndTime, &event.AllDay, &location,
			&externalUID, &externalURL, &recurrenceRule, &color,
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
		if color.Valid {
			event.Color = &color.String
		}
		if externalCalendarID.Valid {
			id := int(externalCalendarID.Int32)
			event.ExternalCalendarID = &id
		}
		if lastSyncedAt.Valid {
			event.LastSyncedAt = &lastSyncedAt.Time
		}

		events = append(events, event)
	}

	return events, nil
}

// GetCalendars returns all external calendars for a user
func (s *ExternalEventService) GetCalendars(userID int) ([]models.ExternalCalendar, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, name, url, sync_enabled, sync_interval_hours,
		       color, last_synced_at, last_error, created_at, updated_at
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

		err := rows.Scan(
			&cal.ID, &cal.UserID, &cal.Name, &cal.URL,
			&cal.SyncEnabled, &cal.SyncIntervalHours, &cal.Color,
			&lastSyncedAt, &lastError, &cal.CreatedAt, &cal.UpdatedAt,
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

		calendars = append(calendars, cal)
	}

	return calendars, nil
}

// CreateCalendar creates a new external calendar subscription
func (s *ExternalEventService) CreateCalendar(userID int, req models.CreateExternalCalendarRequest) (*models.ExternalCalendar, error) {
	// Validate URL by attempting to fetch it
	if err := ValidateICalURL(req.URL); err != nil {
		return nil, fmt.Errorf("invalid iCal URL: %w", err)
	}

	// Set default color if not provided
	color := req.Color
	if color == "" {
		color = "#6366f1" // Default indigo
	}

	var id int
	err := s.db.QueryRow(`
		INSERT INTO external_calendars (user_id, name, url, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, req.Name, req.URL, color).Scan(&id)

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

	err := s.db.QueryRow(`
		SELECT id, user_id, name, url, sync_enabled, sync_interval_hours,
		       color, last_synced_at, last_error, created_at, updated_at
		FROM external_calendars
		WHERE id = $1 AND user_id = $2
	`, calendarID, userID).Scan(
		&cal.ID, &cal.UserID, &cal.Name, &cal.URL,
		&cal.SyncEnabled, &cal.SyncIntervalHours, &cal.Color,
		&lastSyncedAt, &lastError, &cal.CreatedAt, &cal.UpdatedAt,
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

	return &cal, nil
}

// UpdateCalendar updates an existing calendar
func (s *ExternalEventService) UpdateCalendar(calendarID, userID int, req models.UpdateExternalCalendarRequest) error {
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
