package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetTaskSavedSearchesRoute retrieves all saved task searches for the current user.
func (s *Handler) GetTaskSavedSearchesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	searches, err := services.GetTaskSavedSearches(s.GetDB(), userID)
	if err != nil {
		http.Error(w, "Failed to retrieve saved searches", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searches)
}

// GetTaskSavedSearchRoute retrieves a single saved task search by ID.
func (s *Handler) GetTaskSavedSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	searchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid search ID", http.StatusBadRequest)
		return
	}

	search, err := services.GetTaskSavedSearch(s.GetDB(), userID, searchID)
	if err != nil {
		http.Error(w, "Saved search not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(search)
}

// CreateTaskSavedSearchRoute creates a new saved task search.
func (s *Handler) CreateTaskSavedSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateTaskSavedSearchParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if params.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Default omitted enum fields so clients can save with just a name + filter.
	if params.SortField == "" {
		params.SortField = "priority"
	}
	if params.SortDirection == "" {
		params.SortDirection = "asc"
	}
	if params.ViewMode == "" {
		params.ViewMode = "list"
	}

	searchID, err := services.CreateTaskSavedSearch(s.GetDB(), userID, params)
	if err != nil {
		log.Printf("Error creating task saved search: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": searchID})
}

// UpdateTaskSavedSearchRoute updates an existing saved task search.
func (s *Handler) UpdateTaskSavedSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	searchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid search ID", http.StatusBadRequest)
		return
	}

	var params models.UpdateTaskSavedSearchParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := services.UpdateTaskSavedSearch(s.GetDB(), userID, searchID, params); err != nil {
		log.Printf("Error updating task saved search: %v", err)
		status := http.StatusInternalServerError
		if err == sql.ErrNoRows || err.Error() == "task saved search not found" {
			status = http.StatusNotFound
		} else if isValidationError(err) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
}

// DeleteTaskSavedSearchRoute deletes a saved task search.
func (s *Handler) DeleteTaskSavedSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	searchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid search ID", http.StatusBadRequest)
		return
	}

	if err := services.DeleteTaskSavedSearch(s.GetDB(), userID, searchID); err != nil {
		log.Printf("Error deleting task saved search: %v", err)
		if err == sql.ErrNoRows {
			http.Error(w, "Saved search not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete saved search", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isValidationError returns true for errors produced by the service's enum
// validation (messages starting with "invalid ").
func isValidationError(err error) bool {
	return len(err.Error()) >= 8 && err.Error()[:8] == "invalid "
}
