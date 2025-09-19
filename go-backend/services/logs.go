package services

import (
	"database/sql"
	"go-backend/models"
	"log"
	"reflect"
)

func CreateAuditEvent(db *sql.DB, userID int, entityID int, entityType string, action string, oldState interface{}, newState interface{}) error {
	changes := make(map[string]models.FieldChange)

	// If we have both states, compute the differences
	if oldState != nil && newState != nil {
		oldVal := reflect.ValueOf(oldState)
		newVal := reflect.ValueOf(newState)

		// Handle pointer types
		if oldVal.Kind() == reflect.Ptr {
			oldVal = oldVal.Elem()
		}
		if newVal.Kind() == reflect.Ptr {
			newVal = newVal.Elem()
		}

		// Only process if both are structs
		if oldVal.Kind() == reflect.Struct && newVal.Kind() == reflect.Struct {
			for i := 0; i < oldVal.NumField(); i++ {
				field := oldVal.Type().Field(i)
				oldField := oldVal.Field(i)
				newField := newVal.Field(i)

				// Skip certain fields
				if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
					continue
				}

				// Convert interface values to comparable types
				oldValue := oldField.Interface()
				newValue := newField.Interface()

				// Only record if values are different
				if !reflect.DeepEqual(oldValue, newValue) {
					changes[field.Name] = models.FieldChange{
						From: oldValue,
						To:   newValue,
					}
				}
			}
		}
	}

	details := models.Details{
		ChangeType: action,
		Changes:    changes,
	}

	// For create/delete actions, store the full state
	if action == "create" && newState != nil {
		details.CustomData = map[string]interface{}{
			"initial_state": newState,
		}
	} else if action == "delete" && oldState != nil {
		details.CustomData = map[string]interface{}{
			"final_state": oldState,
		}
	}

	_, err := db.Exec(`
		INSERT INTO audit_events (user_id, entity_id, entity_type, action, details)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, entityID, entityType, action, details)

	if err != nil {
		log.Printf("Error creating audit event: %v", err)
		return err
	}

	return nil
}

func GetAuditEvents(db *sql.DB, entityType string, entityID int) ([]models.AuditEvent, error) {
	rows, err := db.Query(`
		SELECT id, user_id, entity_id, entity_type, action, details, created_at
		FROM audit_events
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`, entityType, entityID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.AuditEvent
	for rows.Next() {
		var event models.AuditEvent
		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.EntityID,
			&event.EntityType,
			&event.Action,
			&event.Details,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}
