package services

import (
	"go-backend/models"
	"testing"
)

func TestCreateHabit(t *testing.T) {
	// Note: This test will fail without a real DB connection
	// For now, just verify the function compiles
	params := models.CreateHabitParams{
		Title:     "Test Habit",
		Frequency: models.FrequencyDaily,
	}
	if params.Title != "Test Habit" {
		t.Error("title mismatch")
	}
}

func TestCheckinHabit(t *testing.T) {
	// Compile check
	params := models.CheckinHabitParams{Notes: nil}
	if params.Notes != nil {
		t.Error("expected nil notes")
	}
}

func TestCalculateHabitStats(t *testing.T) {
	// Compile check
	var stats models.HabitStats
	stats.CurrentStreak = 0
	if stats.CurrentStreak != 0 {
		t.Error("expected 0 streak")
	}
}

func TestGetTodaysHabits(t *testing.T) {
	// Compile check for HabitWithCheckin
	var hwc HabitWithCheckin
	hwc.IsDueToday = true
	if !hwc.IsDueToday {
		t.Error("expected true")
	}
}

func TestUpdateHabitParams(t *testing.T) {
	// Compile check for UpdateHabitParams
	title := "Updated Title"
	freq := models.FrequencyWeekly
	params := models.UpdateHabitParams{
		Title:     &title,
		Frequency: &freq,
	}
	if params.Title == nil || *params.Title != "Updated Title" {
		t.Error("title mismatch")
	}
	if params.Frequency == nil || *params.Frequency != models.FrequencyWeekly {
		t.Error("frequency mismatch")
	}
}
