package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

const (
	// MaxCalendarNameLength is the maximum allowed length for calendar names
	MaxCalendarNameLength = 255
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// respondWithError sends a standardized error response
func respondWithError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message, Code: code})
}

// ListExternalCalendarsRoute handles GET /api/user/external-calendars
func (s *Handler) ListExternalCalendarsRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	calendars, err := svc.GetCalendars(userID)
	if err != nil {
		log.Printf("Error fetching external calendars: %v", err)
		respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch calendars")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendars)
}

// CreateExternalCalendarRoute handles POST /api/user/external-calendars
func (s *Handler) CreateExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var req models.CreateExternalCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}
	// Validate name length (in runes for proper UTF-8 handling)
	if utf8.RuneCountInString(req.Name) > MaxCalendarNameLength {
		respondWithError(w, http.StatusBadRequest, "NAME_TOO_LONG", fmt.Sprintf("Name must be at most %d characters", MaxCalendarNameLength))
		return
	}
	// Check for whitespace-only name
	if strings.TrimSpace(req.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "INVALID_NAME", "Name cannot be empty or whitespace only")
		return
	}
	if req.URL == "" {
		respondWithError(w, http.StatusBadRequest, "MISSING_URL", "URL is required")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	calendar, err := svc.CreateCalendar(userID, req)
	if err != nil {
		log.Printf("Error creating external calendar: %v", err)
		// Determine error code based on error message
		code := "CREATE_FAILED"
		if strings.Contains(err.Error(), "invalid iCal URL") {
			code = "INVALID_URL"
		} else if strings.Contains(err.Error(), "invalid color format") {
			code = "INVALID_COLOR"
		}
		respondWithError(w, http.StatusBadRequest, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(calendar)
}

// UpdateExternalCalendarRoute handles PUT /api/user/external-calendars/{id}
func (s *Handler) UpdateExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid calendar ID")
		return
	}

	var req models.UpdateExternalCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	// Validate name length if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			respondWithError(w, http.StatusBadRequest, "INVALID_NAME", "Name cannot be empty or whitespace only")
			return
		}
		if utf8.RuneCountInString(name) > MaxCalendarNameLength {
			respondWithError(w, http.StatusBadRequest, "NAME_TOO_LONG", fmt.Sprintf("Name must be at most %d characters", MaxCalendarNameLength))
			return
		}
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	if err := svc.UpdateCalendar(calendarID, userID, req); err != nil {
		log.Printf("Error updating external calendar: %v", err)
		// Determine error code based on error message
		code := "UPDATE_FAILED"
		if strings.Contains(err.Error(), "sync_interval_hours must be between") {
			code = "INVALID_SYNC_INTERVAL"
		} else if strings.Contains(err.Error(), "invalid color format") {
			code = "INVALID_COLOR"
		}
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "must be between") || strings.Contains(err.Error(), "invalid color") {
			status = http.StatusBadRequest
		}
		respondWithError(w, status, code, err.Error())
		return
	}

	// Fetch and return the updated calendar
	calendar, err := svc.GetCalendar(calendarID, userID)
	if err != nil {
		log.Printf("Error fetching updated calendar: %v", err)
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Calendar not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// DeleteExternalCalendarRoute handles DELETE /api/user/external-calendars/{id}
func (s *Handler) DeleteExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid calendar ID")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	if err := svc.DeleteCalendar(calendarID, userID); err != nil {
		log.Printf("Error deleting external calendar: %v", err)
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncExternalCalendarRoute handles POST /api/user/external-calendars/{id}/sync
func (s *Handler) SyncExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	calendarID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid calendar ID")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	if err := svc.SyncExternalCalendar(calendarID, userID); err != nil {
		log.Printf("Error syncing external calendar: %v", err)
		// Determine error code based on error message
		code := "SYNC_FAILED"
		if strings.Contains(err.Error(), "sync cooldown active") {
			code = "SYNC_COOLDOWN"
		} else if strings.Contains(err.Error(), "calendar not found") {
			code = "NOT_FOUND"
		}
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cooldown") || strings.Contains(err.Error(), "not found") {
			status = http.StatusBadRequest
		}
		respondWithError(w, status, code, err.Error())
		return
	}

	// Fetch and return the updated calendar
	calendar, err := svc.GetCalendar(calendarID, userID)
	if err != nil {
		log.Printf("Error fetching synced calendar: %v", err)
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Calendar not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// GetExternalEventsRoute handles GET /api/user/external-events
func (s *Handler) GetExternalEventsRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	// Parse date range from query params
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		respondWithError(w, http.StatusBadRequest, "MISSING_PARAMS", "start and end query parameters are required")
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "Invalid start date format (use RFC3339)")
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "Invalid end date format (use RFC3339)")
		return
	}

	// Parse pagination params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 0  // Default will be set by service
	offset := 0 // Default

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			respondWithError(w, http.StatusBadRequest, "INVALID_LIMIT", "Invalid limit parameter")
			return
		}
	}

	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			respondWithError(w, http.StatusBadRequest, "INVALID_OFFSET", "Invalid offset parameter")
			return
		}
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	events, total, err := svc.GetEventsInRange(userID, start, end, limit, offset)
	if err != nil {
		log.Printf("Error fetching external events: %v", err)
		respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch events")
		return
	}

	// Return response with pagination info
	response := map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LinkEventToCardRoute handles PUT /api/user/external-events/{id}/link
// Links an external calendar event to a card
func (s *Handler) LinkEventToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid event ID")
		return
	}

	var req models.LinkEventToCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.CardPK <= 0 {
		respondWithError(w, http.StatusBadRequest, "INVALID_CARD_ID", "Valid card ID is required")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	event, err := svc.LinkEventToCard(s.GetDB(), userID, eventID, req.CardPK)
	if err != nil {
		log.Printf("Error linking event to card: %v", err)
		respondWithError(w, http.StatusInternalServerError, "LINK_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// UnlinkEventFromCardRoute handles DELETE /api/user/external-events/{id}/link
// Unlinks an external calendar event from a card
func (s *Handler) UnlinkEventFromCardRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid event ID")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	if err := svc.UnlinkEventFromCard(s.GetDB(), userID, eventID); err != nil {
		log.Printf("Error unlinking event from card: %v", err)
		respondWithError(w, http.StatusInternalServerError, "UNLINK_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEventsByCardRoute handles GET /api/user/cards/{id}/external-events
// Returns all external events linked to a specific card
func (s *Handler) GetEventsByCardRoute(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	vars := mux.Vars(r)
	cardPK, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid card ID")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)
	events, err := svc.GetEventsByCard(s.GetDB(), userID, cardPK)
	if err != nil {
		log.Printf("Error fetching events for card: %v", err)
		respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch events")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
