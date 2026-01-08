package services

import (
	"go-backend/models"
	"go-backend/tests"
	"reflect"
	"testing"
	"time"
)

func TestParseRecurringTasksService(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected models.RecurringTask
		found    bool
	}{
		{
			name:     "every day",
			input:    "hello world every day",
			expected: models.RecurringTask{Frequency: "daily", Interval: 1},
			found:    true,
		},
		{
			name:     "daily",
			input:    "hello world daily",
			expected: models.RecurringTask{Frequency: "daily", Interval: 1},
			found:    true,
		},
		{
			name:     "every 3 days",
			input:    "hello world every 3 days",
			expected: models.RecurringTask{Frequency: "daily", Interval: 3},
			found:    true,
		},
		{
			name:     "weekly",
			input:    "hello world every week",
			expected: models.RecurringTask{Frequency: "weekly", Interval: 7},
			found:    true,
		},
		{
			name:     "every 5 weeks",
			input:    "hello world every 5 weeks",
			expected: models.RecurringTask{Frequency: "weekly", Interval: 5},
			found:    true,
		},
		{
			name:     "monthly",
			input:    "hello world every month",
			expected: models.RecurringTask{Frequency: "monthly", Interval: 30},
			found:    true,
		},
		{
			name:     "every 6 months",
			input:    "hello world every 6 months",
			expected: models.RecurringTask{Frequency: "monthly", Interval: 6},
			found:    true,
		},
		{
			name:     "no recurring task",
			input:    "hello world",
			expected: models.RecurringTask{},
			found:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, found := ParseRecurringTasks(tc.input)
			if found != tc.found {
				t.Errorf("expected found to be %v, but got %v", tc.found, found)
			}
			if !reflect.DeepEqual(output, tc.expected) {
				t.Errorf("expected %+v, but got %+v", tc.expected, output)
			}
		})
	}
}

func TestCreateTask(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	expectedTitle := "Test Task"
	expectedScheduledDate := time.Date(2024, 7, 5, 10, 0, 0, 0, time.UTC)
	expectedDueDate := time.Date(2024, 7, 10, 10, 0, 0, 0, time.UTC)

	task := models.Task{
		CardPK:        1,
		UserID:        userID,
		ScheduledDate: &expectedScheduledDate,
		DueDate:       &expectedDueDate,
		Title:         expectedTitle,
		IsComplete:    false,
	}

	taskID, err := CreateTask(s.DB, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if taskID <= 0 {
		t.Errorf("Expected task ID > 0, got %v", taskID)
	}

	// Verify the task was created correctly
	createdTask, err := GetTask(s.DB, userID, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if createdTask.Title != expectedTitle {
		t.Errorf("Expected title %v, got %v", expectedTitle, createdTask.Title)
	}
	if createdTask.ScheduledDate == nil || !createdTask.ScheduledDate.Equal(expectedScheduledDate) {
		t.Errorf("Expected scheduled date %v, got %v", expectedScheduledDate, createdTask.ScheduledDate)
	}
	if createdTask.DueDate == nil || !createdTask.DueDate.Equal(expectedDueDate) {
		t.Errorf("Expected due date %v, got %v", expectedDueDate, createdTask.DueDate)
	}
}

func TestGetTask(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	task := models.Task{
		UserID:     userID,
		Title:      "Get Task Test",
		IsComplete: false,
	}

	taskID, err := CreateTask(s.DB, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	retrievedTask, err := GetTask(s.DB, userID, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrievedTask.ID != taskID {
		t.Errorf("Expected ID %v, got %v", taskID, retrievedTask.ID)
	}
	if retrievedTask.Title != task.Title {
		t.Errorf("Expected title %v, got %v", task.Title, retrievedTask.Title)
	}
	if retrievedTask.UserID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, retrievedTask.UserID)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	nonExistentID := 99999

	_, err := GetTask(s.DB, userID, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent task, but got none")
	}
}

func TestUpdateTask(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	originalTitle := "Original Task"
	updatedTitle := "Updated Task"

	// Create initial task
	task := models.Task{
		UserID:     userID,
		Title:      originalTitle,
		IsComplete: false,
	}

	taskID, err := CreateTask(s.DB, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Update the task
	updatedTask := models.Task{
		Title:      updatedTitle,
		IsComplete: true,
	}

	_, err = UpdateTask(s.DB, userID, taskID, updatedTask)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify the update
	retrievedTask, err := GetTask(s.DB, userID, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrievedTask.Title != updatedTitle {
		t.Errorf("Expected title %v, got %v", updatedTitle, retrievedTask.Title)
	}
	if !retrievedTask.IsComplete {
		t.Error("Expected task to be complete")
	}
	if retrievedTask.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set when task is completed")
	}
}

func TestDeleteTask(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	task := models.Task{
		UserID:     userID,
		Title:      "Task to Delete",
		IsComplete: false,
	}

	taskID, err := CreateTask(s.DB, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Delete the task
	err = DeleteTask(s.DB, userID, taskID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// Verify task is deleted (should not be found)
	_, err = GetTask(s.DB, userID, taskID)
	if err == nil {
		t.Error("Expected error when getting deleted task, but got none")
	}
}

func TestGetTasks(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create completed task
	completedTask := models.Task{
		UserID:     userID,
		Title:      "Completed Task",
		IsComplete: true,
	}
	_, err := CreateTask(s.DB, completedTask)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Create incomplete task
	incompleteTask := models.Task{
		UserID:     userID,
		Title:      "Incomplete Task",
		IsComplete: false,
	}
	_, err = CreateTask(s.DB, incompleteTask)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get all tasks (including completed)
	allTasks, err := GetTasks(s.DB, userID, true)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	if len(allTasks) < 2 {
		t.Errorf("Expected at least 2 tasks, got %v", len(allTasks))
	}

	// Get only incomplete tasks
	incompleteTasks, err := GetTasks(s.DB, userID, false)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	// Should have fewer tasks when excluding completed ones
	if len(incompleteTasks) >= len(allTasks) {
		t.Errorf("Expected fewer incomplete tasks than all tasks, got %v incomplete and %v total", len(incompleteTasks), len(allTasks))
	}
}

func TestGetTasksByCard(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1

	// Create task associated with a card
	task := models.Task{
		CardPK:     cardPK,
		UserID:     userID,
		Title:      "Card Task",
		IsComplete: false,
	}

	taskID, err := CreateTask(s.DB, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get tasks for the card
	cardTasks, err := GetTasksByCard(s.DB, userID, cardPK)
	if err != nil {
		t.Fatalf("GetTasksByCard failed: %v", err)
	}

	// Should find at least the task we created
	found := false
	for _, cardTask := range cardTasks {
		if cardTask.ID == taskID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Created task not found in card tasks")
	}
}