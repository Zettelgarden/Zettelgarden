package handlers

import (
	"encoding/json"
	"go-backend/llms"
	"io"
	"log"
	"net/http"
)

func (s *Handler) GenerateMemory(userID uint, cardContent string) {

	if s.Server.Testing {
		return
	}

	go func() {
		client := llms.NewDefaultClient(s.DB, int(userID))
		client.RequestType = "memory"
		_, err := llms.GenerateUserMemory(s.DB, client, userID, cardContent)
		if err != nil {
			log.Printf("error generating user memory: %v", err)
			return
		}
		_, err = s.DB.Exec("UPDATE users SET memory_has_changed = true WHERE id = $1", userID)
		if err != nil {
			log.Printf("failed to update memory_has_changed flag for user %d: %v", userID, err)
			return
		}
	}()
}

func (s *Handler) GetUserMemoryRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	memory, err := llms.GetUserMemory(s.DB, uint(userID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"memory": memory})
}

func (s *Handler) UpdateUserMemoryRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON
	var requestData struct {
		Memory string `json:"memory"`
	}

	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Update memory in database
	err = llms.UpdateUserMemory(s.DB, uint(userID), requestData.Memory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Memory updated successfully"})
}
