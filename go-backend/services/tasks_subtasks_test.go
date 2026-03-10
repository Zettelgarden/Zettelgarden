package services

import (
	"testing"
	"time"

	"go-backend/models"
)

func TestPrepareSubtask_InheritsPriority(t *testing.T) {
	// Setup: Create a parent task with priority A
	priorityA := "A"
	parent := &models.Task{
		ID:       1,
		Title:    "Parent Task",
		Priority: &priorityA,
	}

	// Create subtask input without priority
	input := models.Task{
		Title: "Child Task",
	}

	// Execute: Create subtask logic should inherit priority
	subtask := PrepareSubtask(parent, input)

	// Assert: Priority should be inherited
	if subtask.Priority == nil || *subtask.Priority != "A" {
		t.Errorf("Expected priority A, got %v", subtask.Priority)
	}
}

func TestPrepareSubtask_DoesNotInheritPriorityWhenProvided(t *testing.T) {
	// Setup: Create a parent task with priority A
	priorityA := "A"
	priorityC := "C"
	parent := &models.Task{
		ID:       1,
		Title:    "Parent Task",
		Priority: &priorityA,
	}

	// Create subtask input WITH priority C
	input := models.Task{
		Title:    "Child Task",
		Priority: &priorityC,
	}

	// Execute
	subtask := PrepareSubtask(parent, input)

	// Assert: Priority should be C, not inherited A
	if subtask.Priority == nil || *subtask.Priority != "C" {
		t.Errorf("Expected priority C, got %v", subtask.Priority)
	}
}

func TestPrepareSubtask_InheritsTags(t *testing.T) {
	// Setup: Create a parent task with tags
	parent := &models.Task{
		ID:    1,
		Title: "Parent Task",
		Tags: []models.Tag{
			{ID: 1, Name: "backend"},
			{ID: 2, Name: "feature"},
		},
	}

	input := models.Task{
		Title: "Child Task",
	}

	// Execute
	subtask := PrepareSubtask(parent, input)

	// Assert: Tags should be inherited
	if len(subtask.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(subtask.Tags))
	}
}

func TestPrepareSubtask_SetsParentTaskID(t *testing.T) {
	parentID := 1
	parent := &models.Task{
		ID:    parentID,
		Title: "Parent Task",
	}

	input := models.Task{
		Title: "Child Task",
	}

	subtask := PrepareSubtask(parent, input)

	if subtask.ParentTaskID == nil || *subtask.ParentTaskID != parentID {
		t.Errorf("Expected parent_task_id %d, got %v", parentID, subtask.ParentTaskID)
	}
}

func TestPrepareSubtask_DoesNotInheritDates(t *testing.T) {
	// Parent has scheduled and due dates
	now := time.Now()
	parent := &models.Task{
		ID:            1,
		Title:         "Parent Task",
		ScheduledDate: &now,
		DueDate:       &now,
	}

	input := models.Task{
		Title: "Child Task",
	}

	subtask := PrepareSubtask(parent, input)

	// Dates should NOT be inherited
	if subtask.ScheduledDate != nil {
		t.Errorf("Expected scheduled_date to be nil, got %v", subtask.ScheduledDate)
	}
	if subtask.DueDate != nil {
		t.Errorf("Expected due_date to be nil, got %v", subtask.DueDate)
	}
}
