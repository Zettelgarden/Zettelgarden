package models

import "time"

// Frequency constants
const (
	FrequencyDaily  = "daily"
	FrequencyWeekly = "weekly"
	FrequencyCustom = "custom_days"
)

type Habit struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	Title        string     `json:"title"`
	Description  *string    `json:"description"`
	Frequency    string     `json:"frequency"`
	CustomDays   *string    `json:"custom_days"` // JSON array as string
	Icon         *string    `json:"icon"`
	Color        *string    `json:"color"`
	Position     int        `json:"position"`
	LinkedTaskID *int       `json:"linked_task_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Computed fields (not in DB)
	TodayCheckedIn bool   `json:"today_checked_in,omitempty"`
	CurrentStreak   int    `json:"current_streak,omitempty"`
	IsDueToday      bool   `json:"is_due_today,omitempty"`
	CheckedInToday  bool   `json:"checked_in_today,omitempty"`
	TodayLogID      *int   `json:"today_log_id,omitempty"`
}

type HabitLog struct {
	ID          int        `json:"id"`
	HabitID     int        `json:"habit_id"`
	UserID      int        `json:"user_id"`
	CompletedAt time.Time  `json:"completed_at"`
	Notes       *string    `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

type HabitStats struct {
	CurrentStreak     int        `json:"current_streak"`
	LongestStreak     int        `json:"longest_streak"`
	TotalCompletions  int        `json:"total_completions"`
	CompletionRate7d  float64    `json:"completion_rate_7d"`
	CompletionRate30d float64    `json:"completion_rate_30d"`
	LastCompletedAt   *time.Time `json:"last_completed_at"`
}

type CreateHabitParams struct {
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	Frequency    string  `json:"frequency"`
	CustomDays   *string `json:"custom_days"`
	Icon         *string `json:"icon"`
	Color        *string `json:"color"`
	Position     *int    `json:"position"`
	LinkedTaskID *int    `json:"linked_task_id"`
}

type UpdateHabitParams struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Frequency    *string `json:"frequency"`
	CustomDays   *string `json:"custom_days"`
	Icon         *string `json:"icon"`
	Color        *string `json:"color"`
	Position     *int    `json:"position"`
	LinkedTaskID *int    `json:"linked_task_id"`
}

type CheckinHabitParams struct {
	Notes *string `json:"notes"`
}
