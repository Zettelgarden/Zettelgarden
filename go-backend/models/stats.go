package models

import "time"

// DailyStats represents activity statistics for a single day
type DailyStats struct {
	Date           time.Time `json:"date"`
	CardsCreated   int       `json:"cards_created"`
	TasksCreated   int       `json:"tasks_created"`
	TasksCompleted int       `json:"tasks_completed"`
}

// DailyStatsResponse is the API response for daily stats
type DailyStatsResponse struct {
	Stats []DailyStats `json:"stats"`
	Total struct {
		CardsCreated   int `json:"cards_created"`
		TasksCreated   int `json:"tasks_created"`
		TasksCompleted int `json:"tasks_completed"`
	} `json:"total"`
}
