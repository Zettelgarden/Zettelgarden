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

func TestPrepareSubtask_InheritsCardPK(t *testing.T) {
	// Setup: Parent has a card association
	parent := &models.Task{
		ID:     1,
		Title:  "Parent Task",
		CardPK: 42,
	}

	input := models.Task{
		Title: "Child Task",
	}

	subtask := PrepareSubtask(parent, input)

	// Assert: CardPK should be inherited
	if subtask.CardPK != 42 {
		t.Errorf("Expected CardPK 42, got %d", subtask.CardPK)
	}
}

// ===== Nesting Validation Tests =====

func TestValidateParentAssignment_SingleLevelOnly(t *testing.T) {
	// Setup: grandparent -> parent -> child (should fail)
	grandparentID := 1
	parent := &models.Task{
		ID:           2,
		Title:        "Parent",
		ParentTaskID: &grandparentID,
	}
	child := &models.Task{ID: 3, Title: "Child"}

	// Execute: Try to make child a subtask of parent (which is already a subtask)
	err := ValidateParentAssignment(child, parent)

	// Assert: Should fail
	if err == nil {
		t.Error("Expected error for nested subtask, got nil")
	}
	if err != nil && err.Error() != "cannot nest more than one level deep" {
		t.Errorf("Expected nesting error, got: %v", err)
	}
}

func TestValidateParentAssignment_NoSelfReference(t *testing.T) {
	task := &models.Task{ID: 1, Title: "Task"}

	err := ValidateParentAssignment(task, task)

	if err == nil {
		t.Error("Expected error for self-reference, got nil")
	}
	if err != nil && err.Error() != "task cannot be its own parent" {
		t.Errorf("Expected self-reference error, got: %v", err)
	}
}

func TestValidateParentAssignment_CannotNestParentWithChildren(t *testing.T) {
	// Setup: Parent has children
	parent := &models.Task{
		ID:    1,
		Title: "Parent",
		Subtasks: []models.Task{
			{ID: 2, Title: "Child"},
		},
	}
	newParent := &models.Task{ID: 3, Title: "New Parent"}

	err := ValidateParentAssignment(parent, newParent)

	if err == nil {
		t.Error("Expected error for nesting a parent, got nil")
	}
	if err != nil && err.Error() != "cannot make a parent task into a subtask" {
		t.Errorf("Expected parent-has-children error, got: %v", err)
	}
}

func TestValidateParentAssignment_ValidAssignment(t *testing.T) {
	// Valid case: root task becoming child of another root task
	parent := &models.Task{ID: 1, Title: "Parent"}
	child := &models.Task{ID: 2, Title: "Child"}

	err := ValidateParentAssignment(child, parent)

	if err != nil {
		t.Errorf("Expected no error for valid assignment, got: %v", err)
	}
}

// ===== Completion Validation Tests =====

func TestValidateTaskCompletion_BlockedByIncompleteSubtasks(t *testing.T) {
	parent := &models.Task{
		ID: 1,
		Subtasks: []models.Task{
			{ID: 2, IsComplete: false},
			{ID: 3, IsComplete: false},
		},
	}

	err := ValidateTaskCompletion(parent, false)

	if err == nil {
		t.Error("Expected error for incomplete subtasks, got nil")
	}
	if incompleteErr, ok := err.(*IncompleteSubtaskError); !ok {
		t.Errorf("Expected IncompleteSubtaskError, got %T", err)
	} else {
		if incompleteErr.IncompleteCount != 2 {
			t.Errorf("Expected 2 incomplete, got %d", incompleteErr.IncompleteCount)
		}
	}
}

func TestValidateTaskCompletion_AllowedWhenSubtasksComplete(t *testing.T) {
	parent := &models.Task{
		ID: 1,
		Subtasks: []models.Task{
			{ID: 2, IsComplete: true},
			{ID: 3, IsComplete: true},
		},
	}

	err := ValidateTaskCompletion(parent, false)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateTaskCompletion_ForceBypassesValidation(t *testing.T) {
	parent := &models.Task{
		ID: 1,
		Subtasks: []models.Task{
			{ID: 2, IsComplete: false},
		},
	}

	err := ValidateTaskCompletion(parent, true) // force=true

	if err != nil {
		t.Errorf("Expected no error with force, got: %v", err)
	}
}

func TestValidateTaskCompletion_NoSubtasks(t *testing.T) {
	parent := &models.Task{
		ID:       1,
		Subtasks: []models.Task{},
	}

	err := ValidateTaskCompletion(parent, false)

	if err != nil {
		t.Errorf("Expected no error for task with no subtasks, got: %v", err)
	}
}
