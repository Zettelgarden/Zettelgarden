package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ListExternalCalendarsRoute handles GET /api/user/external-calendars
func (s *Handler) ListExternalCalendarsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	svc := services.NewExternalEventService(s.GetDB())
	calendars, err := svc.GetCalendars(userID)
	if err != nil {
		log.Printf("Error fetching external calendars: %v", err)
		http.Error(w, "Failed to fetch calendars", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendars)
}

// CreateExternalCalendarRoute handles POST /api/user/external-calendars
func (s *Handler) CreateExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req models.CreateExternalCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	svc := services.NewExternalEventService(s.GetDB())
	calendar, err := svc.CreateCalendar(userID, req)
	if err != nil {
		log.Printf("Error creating external calendar: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(calendar)
}

// UpdateExternalCalendarRoute handles PUT /api/user/external-calendars/{id}
func (s *Handler) UpdateExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid calendar ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateExternalCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	svc := services.NewExternalEventService(s.GetDB())
	if err := svc.UpdateCalendar(calendarID, userID, req); err != nil {
		log.Printf("Error updating external calendar: %v", err)
		http.Error(w, "Failed to update calendar", http.StatusInternalServerError)
		return
	}

	// Fetch and return the updated calendar
	calendar, err := svc.GetCalendar(calendarID, userID)
	if err != nil {
		log.Printf("Error fetching updated calendar: %v", err)
		http.Error(w, "Calendar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// DeleteExternalCalendarRoute handles DELETE /api/user/external-calendars/{id}
func (s *Handler) DeleteExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid calendar ID", http.StatusBadRequest)
		return
	}

	svc := services.NewExternalEventService(s.GetDB())
	if err := svc.DeleteCalendar(calendarID, userID); err != nil {
		log.Printf("Error deleting external calendar: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncExternalCalendarRoute handles POST /api/user/external-calendars/{id}/sync
func (s *Handler) SyncExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid calendar ID", http.StatusBadRequest)
		return
	}

	svc := services.NewExternalEventService(s.GetDB())
	if err := svc.SyncExternalCalendar(calendarID, userID); err != nil {
		log.Printf("Error syncing external calendar: %v", err)
		http.Error(w, fmt.Sprintf("Sync failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Fetch and return the updated calendar
	calendar, err := svc.GetCalendar(calendarID, userID)
	if err != nil {
		log.Printf("Error fetching synced calendar: %v", err)
		http.Error(w, "Calendar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// GetExternalEventsRoute handles GET /api/user/external-events
func (s *Handler) GetExternalEventsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse date range from query params
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		http.Error(w, "start and end query parameters are required", http.StatusBadRequest)
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	svc := services.NewExternalEventService(s.GetDB())
	events, err := svc.GetEventsInRange(userID, start, end)
	if err != nil {
		log.Printf("Error fetching external events: %v", err)
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
