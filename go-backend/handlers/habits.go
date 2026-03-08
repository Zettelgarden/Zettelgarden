package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"go-backend/models"
	"go-backend/services"

	"github.com/gorilla/mux"
)

func (s *Handler) GetHabitsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	habits, err := services.GetHabits(s.GetDB(), userID)
	if err != nil {
		log.Printf("Error getting habits: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(habits)
}

func (s *Handler) GetHabitRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid habit id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	habit, err := services.GetHabit(s.GetDB(), userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(habit)
}

func (s *Handler) CreateHabitRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	var params models.CreateHabitParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding create habit request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	id, err := services.CreateHabit(s.GetDB(), userID, params)
	if err != nil {
		log.Printf("Error creating habit: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (s *Handler) DeleteHabitRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid habit id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	err = services.DeleteHabit(s.GetDB(), userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Handler) CheckinHabitRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid habit id param: %v", err)
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var params models.CheckinHabitParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding checkin request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	timezone, _ := s.GetUserTimezone(userID)
	logID, err := services.CheckinHabit(s.GetDB(), userID, id, params, timezone)
	if err != nil {
		if err.Error() == "already checked in today" {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": logID})
}

func (s *Handler) GetTodaysHabitsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	timezone, _ := s.GetUserTimezone(userID)
	habits, err := services.GetTodaysHabits(s.GetDB(), userID, timezone)
	if err != nil {
		log.Printf("Error getting today's habits: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(habits)
}

// GetHabitLogsRoute retrieves the check-in history for a specific habit
// GET /api/habits/{id}/logs
func (s *Handler) GetHabitLogsRoute(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement this handler in Task 2
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
