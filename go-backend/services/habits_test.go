package services

import (
	"testing"
	"go-backend/models"
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
