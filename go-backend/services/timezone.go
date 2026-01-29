package services

import (
	"log"
	"time"

	"go-backend/models"
)

// ConvertToUserTimezone converts a time.Time to the user's timezone
func ConvertToUserTimezone(t *time.Time, timezone string) *time.Time {
	if t == nil {
		return nil
	}

	if timezone == "" {
		timezone = "UTC"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Failed to load timezone %s, falling back to UTC: %v", timezone, err)
		loc = time.UTC
	}

	converted := t.In(loc)
	return &converted
}

// ConvertTaskTimesToUserTimezone converts all time fields in a task to the user's timezone
func ConvertTaskTimesToUserTimezone(task *models.Task, timezone string) {
	if timezone == "" {
		timezone = "UTC"
	}

	// Convert ScheduledDate if it exists
	task.ScheduledDate = ConvertToUserTimezone(task.ScheduledDate, timezone)

	// Convert DueDate if it exists
	task.DueDate = ConvertToUserTimezone(task.DueDate, timezone)

	// Convert CreatedAt
	convertedCreatedAt := ConvertToUserTimezone(&task.CreatedAt, timezone)
	if convertedCreatedAt != nil {
		task.CreatedAt = *convertedCreatedAt
	}

	// Convert UpdatedAt
	convertedUpdatedAt := ConvertToUserTimezone(&task.UpdatedAt, timezone)
	if convertedUpdatedAt != nil {
		task.UpdatedAt = *convertedUpdatedAt
	}

	// Convert CompletedAt if it exists
	task.CompletedAt = ConvertToUserTimezone(task.CompletedAt, timezone)

	// Convert ReminderTime if it exists
	task.ReminderTime = ConvertToUserTimezone(task.ReminderTime, timezone)
}