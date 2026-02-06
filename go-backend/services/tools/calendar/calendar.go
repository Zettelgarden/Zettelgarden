// Package calendar provides calendar-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The calendar domain contains tools for managing external calendar subscriptions and events.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// For this implementation, the calendar domain package demonstrates the pattern
// for splitting calendar tools into a separate domain package. The registration
// is handled in services/calendar_tools.go to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions (GetExternalCalendars, GetExternalEvents, LinkEventToCard)
// 2. Domain-specific business logic for calendar operations
// 3. External calendar integration support
//
// Tools provided:
// - list_external_calendars: List all external calendar subscriptions
// - list_external_events: List events within a date range
// - link_event_to_card: Link an event to a card
package calendar

import (
	"database/sql"
	"fmt"
	"time"

	"go-backend/models"
)

// GetExternalCalendars retrieves all external calendar subscriptions for a user.
// This is the domain data access function for calendar listing operations.
func GetExternalCalendars(db *sql.DB, userID int) ([]models.ExternalCalendar, error) {
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
			_calError := lastError.String
			cal.LastError = &_calError
		}
		if username.Valid {
			_username := username.String
			cal.Username = &_username
		}

		calendars = append(calendars, cal)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return calendars, nil
}

// GetExternalEvents retrieves external calendar events within a specified date range.
// This is the domain data access function for event listing operations.
func GetExternalEvents(db *sql.DB, userID int, start, end time.Time, limit, offset int) ([]models.ExternalEvent, int, error) {
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

	rows, err := db.Query(query, userID, start, end, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.ExternalEvent
	for rows.Next() {
		var event models.ExternalEvent
		var desc, loc, uid, url, rule, rid, color sql.NullString
		var calID, cardPK sql.NullInt32
		var recurrenceInstance sql.NullInt32

		err := rows.Scan(
			&event.ID, &event.UserID, &calID, &event.Title, &desc,
			&event.StartTime, &event.EndTime, &event.AllDay, &loc,
			&uid, &url, &rule, &rid, &recurrenceInstance,
			&color, &cardPK,
			&event.CreatedAt, &event.UpdatedAt, &event.LastSyncedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if desc.Valid {
			event.Description = &desc.String
		}
		if loc.Valid {
			event.Location = &loc.String
		}
		if uid.Valid {
			event.ExternalUID = &uid.String
		}
		if url.Valid {
			event.ExternalURL = &url.String
		}
		if rule.Valid {
			event.RecurrenceRule = &rule.String
		}
		if rid.Valid {
			event.RecurrenceID = &rid.String
		}
		if color.Valid {
			event.Color = &color.String
		}
		if calID.Valid {
			_calID := int(calID.Int32)
			event.ExternalCalendarID = &_calID
		}
		if cardPK.Valid {
			_cardPK := int(cardPK.Int32)
			event.CardPK = &_cardPK
		}
		if recurrenceInstance.Valid {
			_instance := int(recurrenceInstance.Int32)
			event.RecurrenceInstance = &_instance
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// LinkEventToCard creates a bidirectional association between an external event and a card.
// This is the domain data access function for event-card linking operations.
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
	var event models.ExternalEvent
	var desc, loc, uid, url, rule, rid, color sql.NullString
	var calID, _cardPK sql.NullInt32
	var recurrenceInstance sql.NullInt32

	err = db.QueryRow(`
		SELECT e.id, e.user_id, e.external_calendar_id, e.title, e.description,
		       e.start_time, e.end_time, e.all_day, e.location,
		       e.external_uid, e.external_url, e.recurrence_rule,
		       e.recurrence_id, e.recurrence_instance,
		       COALESCE(ec.color, e.color) as color,
		       e.card_pk, e.created_at, e.updated_at, e.last_synced_at
		FROM external_events e
		LEFT JOIN external_calendars ec ON e.external_calendar_id = ec.id
		WHERE e.id = $1 AND e.user_id = $2
	`, eventID, userID).Scan(
		&event.ID, &event.UserID, &calID, &event.Title, &desc,
		&event.StartTime, &event.EndTime, &event.AllDay, &loc,
		&uid, &url, &rule, &rid, &recurrenceInstance,
		&color, &_cardPK,
		&event.CreatedAt, &event.UpdatedAt, &event.LastSyncedAt,
	)
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		event.Description = &desc.String
	}
	if loc.Valid {
		event.Location = &loc.String
	}
	if uid.Valid {
		event.ExternalUID = &uid.String
	}
	if url.Valid {
		event.ExternalURL = &url.String
	}
	if rule.Valid {
		event.RecurrenceRule = &rule.String
	}
	if rid.Valid {
		event.RecurrenceID = &rid.String
	}
	if color.Valid {
		event.Color = &color.String
	}
	if calID.Valid {
		_calID := int(calID.Int32)
		event.ExternalCalendarID = &_calID
	}
	if _cardPK.Valid {
		_cardPKVal := int(_cardPK.Int32)
		event.CardPK = &_cardPKVal
	}
	if recurrenceInstance.Valid {
		_instance := int(recurrenceInstance.Int32)
		event.RecurrenceInstance = &_instance
	}

	return &event, nil
}
