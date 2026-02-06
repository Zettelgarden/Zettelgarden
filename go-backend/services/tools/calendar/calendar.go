// Package calendar provides calendar-related data access and business logic
// for the Zettelgarden tool registry.
//
// This package contains functions for managing external calendar integrations,
// including listing calendars, events, and linking events to cards.
package calendar

import (
	"database/sql"
	"fmt"
	"time"

	"go-backend/models"
)

// GetCalendars retrieves all external calendar subscriptions for a user
func GetCalendars(db *sql.DB, userID int) ([]models.ExternalCalendar, error) {
	rows, err := db.Query(`
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
		var username sql.NullString
		if err := rows.Scan(
			&cal.ID,
			&cal.UserID,
			&cal.Name,
			&cal.URL,
			&cal.SyncEnabled,
			&cal.SyncIntervalHours,
			&cal.Color,
			&cal.LastSyncedAt,
			&cal.LastError,
			&cal.CreatedAt,
			&cal.UpdatedAt,
			&username,
		); err != nil {
			return nil, err
		}
		if username.Valid {
			cal.Username = &username.String
		}
		calendars = append(calendars, cal)
	}

	return calendars, nil
}

// GetEventsInRange retrieves external calendar events within a date range
func GetEventsInRange(db *sql.DB, userID int, start, end time.Time, limit, offset int) ([]models.ExternalEvent, int, error) {
	// First, get the total count
	var total int
	err := db.QueryRow(`
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
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Get events - simplified query without JOIN for now
	query := `
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule,
		       recurrence_id, recurrence_instance,
		       color, card_pk, created_at, updated_at, last_synced_at
		FROM external_events
		WHERE user_id = $1
		  AND start_time >= $2
		  AND end_time <= $3
		ORDER BY start_time
		LIMIT $4 OFFSET $5
	`

	rows, err := db.Query(query, userID, start, end, limit, offset)
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

		if err := rows.Scan(
			&event.ID,
			&event.UserID,
			&externalCalendarID,
			&event.Title,
			&description,
			&event.StartTime,
			&event.EndTime,
			&event.AllDay,
			&location,
			&externalUID,
			&externalURL,
			&recurrenceRule,
			&recurrenceID,
			&recurrenceInstance,
			&color,
			&cardPK,
			&event.CreatedAt,
			&event.UpdatedAt,
			&lastSyncedAt,
		); err != nil {
			return nil, 0, err
		}

		// Convert nullable fields to pointers
		if externalCalendarID.Valid {
			id := int(externalCalendarID.Int32)
			event.ExternalCalendarID = &id
		}
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
		if cardPK.Valid {
			pk := int(cardPK.Int32)
			event.CardPK = &pk
		}
		if lastSyncedAt.Valid {
			event.LastSyncedAt = &lastSyncedAt.Time
		}

		events = append(events, event)
	}

	return events, total, nil
}

// LinkEventToCard links an external calendar event to a card
func LinkEventToCard(db *sql.DB, userID int, eventID int, cardPK int) (*models.ExternalEvent, error) {
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

	// Update the link
	_, err = db.Exec(`
		UPDATE external_events
		SET card_pk = $1, updated_at = NOW()
		WHERE id = $2
	`, cardPK, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to link event to card: %w", err)
	}

	// Return the updated event
	query := `
		SELECT id, user_id, external_calendar_id, title, description,
		       start_time, end_time, all_day, location,
		       external_uid, external_url, recurrence_rule,
		       recurrence_id, recurrence_instance,
		       color, card_pk, created_at, updated_at, last_synced_at
		FROM external_events
		WHERE id = $1
	`

	var event models.ExternalEvent
	var description, location, externalUID, externalURL, recurrenceRule, recurrenceID, color sql.NullString
	var externalCalendarID sql.NullInt32
	var linkedCardPK sql.NullInt32
	var recurrenceInstance sql.NullInt32
	var lastSyncedAt sql.NullTime

	err = db.QueryRow(query, eventID).Scan(
		&event.ID,
		&event.UserID,
		&externalCalendarID,
		&event.Title,
		&description,
		&event.StartTime,
		&event.EndTime,
		&event.AllDay,
		&location,
		&externalUID,
		&externalURL,
		&recurrenceRule,
		&recurrenceID,
		&recurrenceInstance,
		&color,
		&linkedCardPK,
		&event.CreatedAt,
		&event.UpdatedAt,
		&lastSyncedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields to pointers
	if externalCalendarID.Valid {
		id := int(externalCalendarID.Int32)
		event.ExternalCalendarID = &id
	}
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
	if linkedCardPK.Valid {
		pk := int(linkedCardPK.Int32)
		event.CardPK = &pk
	}
	if lastSyncedAt.Valid {
		event.LastSyncedAt = &lastSyncedAt.Time
	}

	return &event, nil
}

// ParseISODate parses an ISO 8601 date string
func ParseISODate(dateStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, dateStr)
}

// ValidateDateRange checks if start is before end
func ValidateDateRange(start, end time.Time) error {
	if start.After(end) {
		return fmt.Errorf("start date must be before end date")
	}
	return nil
}
