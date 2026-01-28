package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"time"
)

// GetDailyStatsRoute returns daily activity statistics for a date range
// Query params: start_date (optional), end_date (optional)
// Defaults to last 365 days if not specified
func (s *Handler) GetDailyStatsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get user's timezone for proper date filtering
	userTimezone, err := s.GetUserTimezone(userID)
	if err != nil {
		userTimezone = "UTC" // Fallback to UTC on error
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time

	// Default to last 365 days if not specified
	if startDateStr == "" || endDateStr == "" {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -365)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			log.Printf("Invalid start_date format: %v", err)
			http.Error(w, "Invalid start_date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			log.Printf("Invalid end_date format: %v", err)
			http.Error(w, "Invalid end_date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}

	// Validate date range (max 365 days)
	daysDiff := endDate.Sub(startDate).Hours() / 24
	if daysDiff > 365 {
		http.Error(w, "Date range cannot exceed 365 days", http.StatusBadRequest)
		return
	}

	if daysDiff < 0 {
		http.Error(w, "start_date must be before end_date", http.StatusBadRequest)
		return
	}

	// Fetch stats from database
	stats, err := services.GetDailyStats(s.TX(), userID, startDate, endDate, userTimezone)
	if err != nil {
		log.Printf("Error getting daily stats: %v", err)
		http.Error(w, "Failed to fetch daily stats", http.StatusInternalServerError)
		return
	}

	// Calculate totals
	var response models.DailyStatsResponse
	response.Stats = stats
	response.Total.CardsCreated = 0
	response.Total.TasksCreated = 0
	response.Total.TasksCompleted = 0

	for _, stat := range stats {
		response.Total.CardsCreated += stat.CardsCreated
		response.Total.TasksCreated += stat.TasksCreated
		response.Total.TasksCompleted += stat.TasksCompleted
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDayTasksRoute returns all tasks completed on a specific date
// Query param: date (required, format: YYYY-MM-DD)
func (s *Handler) GetDayTasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get user's timezone for proper date filtering
	userTimezone, err := s.GetUserTimezone(userID)
	if err != nil {
		userTimezone = "UTC" // Fallback to UTC on error
	}

	// Parse date parameter
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date parameter is required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Printf("Invalid date format: %v", err)
		http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Fetch tasks from database
	tasks, err := services.GetTasksCompletedOnDate(s.TX(), userID, date, userTimezone)
	if err != nil {
		log.Printf("Error getting tasks for date: %v", err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	// Load tags for each task (following pattern from handlers/tasks.go)
	for i := range tasks {
		tags, err := s.QueryTagsForTask(userID, tasks[i].ID)
		if err == nil {
			tasks[i].Tags = tags
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetDayCardsRoute returns all cards created on a specific date
// Query param: date (required, format: YYYY-MM-DD)
func (s *Handler) GetDayCardsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get user's timezone for proper date filtering
	userTimezone, err := s.GetUserTimezone(userID)
	if err != nil {
		userTimezone = "UTC" // Fallback to UTC on error
	}

	// Parse date parameter
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date parameter is required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Printf("Invalid date format: %v", err)
		http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Fetch cards from database
	cards, err := services.GetCardsCreatedOnDate(s.TX(), userID, date, userTimezone)
	if err != nil {
		log.Printf("Error getting cards for date: %v", err)
		http.Error(w, "Failed to fetch cards", http.StatusInternalServerError)
		return
	}

	// Load tags for each card (following pattern from handlers/pins.go)
	for i := range cards {
		tags, err := services.QueryTagsForCard(s.DB, userID, cards[i].ID)
		if err == nil {
			cards[i].Tags = tags
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
