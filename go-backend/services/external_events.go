package services

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"regexp"
	"strings"
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

	// Decrypt password if provided
	var password string
	if encryptedPassword.Valid && s.encryptionService != nil {
		decrypted, err := s.encryptionService.Decrypt(encryptedPassword.String)
		if err != nil {
			log.Printf("[Sync] Failed to decrypt password for calendar %d: %v", calendarID, err)
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		password = decrypted
	}

	// Fetch events from iCal feed with credentials
	icalEvents, err := FetchICalURL(url, username.String, password)
	if err != nil {
		log.Printf("[Sync] Failed to fetch iCal feed for calendar %d: %v", calendarID, err)
		s.UpdateLastSyncError(calendarID, err.Error())
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	log.Printf("[Sync] Fetched %d events from iCal feed for calendar %d", len(icalEvents), calendarID)

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

	log.Printf("[Sync] Synced calendar %d: %d events imported, %d errors", calendarID, successCount, errorCount)

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
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule, color, card_pk,
		       created_at, updated_at, last_synced_at
		FROM external_events
		WHERE user_id = $1
		  AND start_time >= $2
		  AND end_time <= $3
		ORDER BY start_time
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
		var description, location, externalUID, externalURL, recurrenceRule, color sql.NullString
		var externalCalendarID sql.NullInt32
		var cardPK sql.NullInt32
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.UserID, &externalCalendarID,
			&event.Title, &description,
			&event.StartTime, &event.EndTime, &event.AllDay, &location,
			&externalUID, &externalURL, &recurrenceRule, &color,
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
		       username, password
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
		var password sql.NullString

		err := rows.Scan(
			&cal.ID, &cal.UserID, &cal.Name, &cal.URL,
			&cal.SyncEnabled, &cal.SyncIntervalHours, &cal.Color,
			&lastSyncedAt, &lastError, &cal.CreatedAt, &cal.UpdatedAt,
			&username, &password,
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
		// Never return password, it stays encrypted

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

	if err := ValidateICalURL(req.URL, username, password); err != nil {
		return nil, fmt.Errorf("invalid iCal URL: %w", err)
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
	if req.URL != nil {
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
	if req.Password != nil {
		// Encrypt new password if provided
		if s.encryptionService == nil {
			return fmt.Errorf("encryption service not available")
		}
		if *req.Password != "" {
			encrypted, err := s.encryptionService.Encrypt(*req.Password)
			if err != nil {
				return fmt.Errorf("failed to encrypt password: %w", err)
			}
			updates = append(updates, fmt.Sprintf("password = $%d", argPos))
			args = append(args, encrypted)
			argPos++
		} else {
			// Empty string means clear the password
			updates = append(updates, fmt.Sprintf("password = $%d", argPos))
			args = append(args, nil)
			argPos++
		}
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
		       external_uid, external_url, recurrence_rule, color, card_pk,
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
		var description, location, externalUID, externalURL, recurrenceRule, color sql.NullString
		var externalCalendarID sql.NullInt32
		var scannedCardPK sql.NullInt32
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.UserID, &externalCalendarID,
			&event.Title, &description,
			&event.StartTime, &event.EndTime, &event.AllDay, &location,
			&externalUID, &externalURL, &recurrenceRule, &color,
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
	var description, location, externalUID, externalURL, recurrenceRule, color sql.NullString
	var externalCalendarID sql.NullInt32
	var cardPK sql.NullInt32
	var lastSyncedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule, color, card_pk,
		       created_at, updated_at, last_synced_at
		FROM external_events
		WHERE id = $1 AND user_id = $2
	`, eventID, userID).Scan(
		&event.ID, &event.UserID, &externalCalendarID,
		&event.Title, &description,
		&event.StartTime, &event.EndTime, &event.AllDay, &location,
		&externalUID, &externalURL, &recurrenceRule, &color,
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

