package models

import (
	"testing"
	"time"
)

func TestHabit_Frequency(t *testing.T) {
	habit := Habit{
		Frequency: FrequencyDaily,
	}
	if habit.Frequency != FrequencyDaily {
		t.Errorf("expected %s, got %s", FrequencyDaily, habit.Frequency)
	}
}

func TestHabitLog_CompletedAt(t *testing.T) {
	now := time.Now().UTC()
	log := HabitLog{
		CompletedAt: now,
	}
	if log.CompletedAt.IsZero() {
		t.Error("completed_at should not be zero")
	}
}

func TestFrequencyConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Daily", FrequencyDaily, "daily"},
		{"Weekly", FrequencyWeekly, "weekly"},
		{"Custom", FrequencyCustom, "custom_days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestHabit_ComputedFields(t *testing.T) {
	habit := Habit{
		ID:           1,
		UserID:       1,
		Title:        "Test Habit",
		Frequency:    FrequencyDaily,
		Position:     1,
		TodayCheckedIn: true,
		CurrentStreak: 5,
		IsDueToday:    true,
		CheckedInToday: true,
	}

	if habit.TodayCheckedIn != true {
		t.Error("expected TodayCheckedIn to be true")
	}
	if habit.CurrentStreak != 5 {
		t.Errorf("expected CurrentStreak to be 5, got %d", habit.CurrentStreak)
	}
	if habit.IsDueToday != true {
		t.Error("expected IsDueToday to be true")
	}
	if habit.CheckedInToday != true {
		t.Error("expected CheckedInToday to be true")
	}
}

func TestCreateHabitParams(t *testing.T) {
	params := CreateHabitParams{
		Title:       "New Habit",
		Description: stringPtr("Description"),
		Frequency:   FrequencyDaily,
		Icon:        stringPtr("icon"),
		Color:       stringPtr("#FF0000"),
		Position:    intPtr(1),
	}

	if params.Title != "New Habit" {
		t.Errorf("expected title 'New Habit', got %s", params.Title)
	}
	if *params.Description != "Description" {
		t.Errorf("expected description 'Description', got %s", *params.Description)
	}
}

func TestUpdateHabitParams(t *testing.T) {
	newTitle := "Updated Habit"
	params := UpdateHabitParams{
		Title:    &newTitle,
		Position: intPtr(5),
	}

	if *params.Title != "Updated Habit" {
		t.Errorf("expected title 'Updated Habit', got %s", *params.Title)
	}
	if *params.Position != 5 {
		t.Errorf("expected position 5, got %d", *params.Position)
	}
}

func TestCheckinHabitParams(t *testing.T) {
	notes := "Great progress!"
	params := CheckinHabitParams{
		Notes: &notes,
	}

	if *params.Notes != "Great progress!" {
		t.Errorf("expected notes 'Great progress!', got %s", *params.Notes)
	}
}

func TestHabitStats(t *testing.T) {
	now := time.Now().UTC()
	stats := HabitStats{
		CurrentStreak:     10,
		LongestStreak:     30,
		TotalCompletions:  100,
		CompletionRate7d:  0.857,
		CompletionRate30d: 0.723,
		LastCompletedAt:   &now,
	}

	if stats.CurrentStreak != 10 {
		t.Errorf("expected CurrentStreak 10, got %d", stats.CurrentStreak)
	}
	if stats.LongestStreak != 30 {
		t.Errorf("expected LongestStreak 30, got %d", stats.LongestStreak)
	}
	if stats.TotalCompletions != 100 {
		t.Errorf("expected TotalCompletions 100, got %d", stats.TotalCompletions)
	}
}

func TestHabitLog(t *testing.T) {
	now := time.Now().UTC()
	notes := "Completed successfully"
	log := HabitLog{
		ID:          1,
		HabitID:     1,
		UserID:      1,
		CompletedAt: now,
		Notes:       &notes,
	}

	if log.ID != 1 {
		t.Errorf("expected ID 1, got %d", log.ID)
	}
	if log.HabitID != 1 {
		t.Errorf("expected HabitID 1, got %d", log.HabitID)
	}
	if *log.Notes != "Completed successfully" {
		t.Errorf("expected notes 'Completed successfully', got %s", *log.Notes)
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
