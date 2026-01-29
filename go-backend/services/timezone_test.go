package services

import (
	"testing"
	"time"
	"go-backend/models"
)

func TestConvertTaskTimesToUserTimezone(t *testing.T) {
	// Create a test task with some times
	utc := time.Now().UTC()
	createdAt := utc
	updatedAt := utc.Add(time.Hour)
	scheduledDate := &utc
	dueDate := &updatedAt
	reminderTime := &updatedAt
	completedAt := &updatedAt

	task := &models.Task{
		ID:            1,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		ScheduledDate: scheduledDate,
		DueDate:       dueDate,
		ReminderTime:  reminderTime,
		CompletedAt:   completedAt,
		Title:         "Test Task",
		Status:        "todo",
		IsComplete:    false,
	}

	// Test conversion to US/Eastern timezone
	easternTimezone := "America/New_York"
	ConvertTaskTimesToUserTimezone(task, easternTimezone)

	// Load Eastern location
	loc, _ := time.LoadLocation(easternTimezone)

	// Check that times were converted
	if !task.CreatedAt.Equal(createdAt.In(loc)) {
		t.Errorf("CreatedAt not converted correctly: expected %v, got %v", createdAt.In(loc), task.CreatedAt)
	}
	if !task.UpdatedAt.Equal(updatedAt.In(loc)) {
		t.Errorf("UpdatedAt not converted correctly: expected %v, got %v", updatedAt.In(loc), task.UpdatedAt)
	}
	if task.ScheduledDate == nil || !task.ScheduledDate.Equal(utc.In(loc)) {
		t.Errorf("ScheduledDate not converted correctly")
	}
	if task.DueDate == nil || !task.DueDate.Equal(updatedAt.In(loc)) {
		t.Errorf("DueDate not converted correctly")
	}
	if task.ReminderTime == nil || !task.ReminderTime.Equal(updatedAt.In(loc)) {
		t.Errorf("ReminderTime not converted correctly")
	}
	if task.CompletedAt == nil || !task.CompletedAt.Equal(updatedAt.In(loc)) {
		t.Errorf("CompletedAt not converted correctly")
	}
}