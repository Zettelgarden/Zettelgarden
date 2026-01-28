package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetTaskStatusesRoute retrieves all task statuses for the current user
func (s *Handler) GetTaskStatusesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	statuses, err := services.GetTaskStatuses(s.GetDB(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// GetTaskStatusRoute retrieves a single task status by ID
func (s *Handler) GetTaskStatusRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	statusID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid status ID", http.StatusBadRequest)
		return
	}

	status, err := services.GetTaskStatus(s.GetDB(), userID, statusID)
	if err != nil {
		http.Error(w, "Task status not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// CreateTaskStatusRoute creates a new task status
func (s *Handler) CreateTaskStatusRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateTaskStatusParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if params.Name == "" || params.DisplayName == "" || params.Color == "" {
		http.Error(w, "name, display_name, and color are required", http.StatusBadRequest)
		return
	}

	statusID, err := services.CreateTaskStatus(s.GetDB(), userID, params)
	if err != nil {
		log.Printf("Error creating task status: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": statusID})
}

// UpdateTaskStatusRoute updates an existing task status
func (s *Handler) UpdateTaskStatusRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	statusID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid status ID", http.StatusBadRequest)
		return
	}

	var params models.UpdateTaskStatusParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err = services.UpdateTaskStatus(s.GetDB(), userID, statusID, params)
	if err != nil {
		log.Printf("Error updating task status: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GenericResponse{
		Message: "success",
		Error:   false,
	})
}

// DeleteTaskStatusRoute deletes a task status
func (s *Handler) DeleteTaskStatusRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	statusID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid status ID", http.StatusBadRequest)
		return
	}

	err = services.DeleteTaskStatus(s.GetDB(), userID, statusID)
	if err != nil {
		log.Printf("Error deleting task status: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderTaskStatusesRoute reorders task statuses
func (s *Handler) ReorderTaskStatusesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.ReorderTaskStatusesParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if len(params.StatusIDs) == 0 {
		http.Error(w, "status_ids array is required", http.StatusBadRequest)
		return
	}

	err := services.ReorderTaskStatuses(s.DB, userID, params.StatusIDs)
	if err != nil {
		log.Printf("Error reordering task statuses: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GenericResponse{
		Message: "success",
		Error:   false,
	})
}
