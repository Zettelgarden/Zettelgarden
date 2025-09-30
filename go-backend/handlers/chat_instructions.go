package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
)

// UpdateInstructionsRequest represents the request to update chat instructions
type UpdateInstructionsRequest struct {
	Instructions string `json:"instructions"`
}

// GetInstructionsRoute gets user's chat instructions
func (s *Handler) GetInstructionsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	instructions, err := s.GetChatInstructions(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty instructions if none exist
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models.ChatInstructions{
				UserID:       userID,
				Instructions: "",
			})
			return
		}
		log.Printf("Error getting chat instructions: %v", err)
		http.Error(w, "Failed to get chat instructions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instructions)
}

// UpdateInstructionsRoute updates user's chat instructions
func (s *Handler) UpdateInstructionsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req UpdateInstructionsRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate instructions length
	if len(req.Instructions) > 10000 {
		http.Error(w, "Instructions too long (max 10000 characters)", http.StatusBadRequest)
		return
	}

	instructions, err := s.UpsertChatInstructions(userID, req.Instructions)
	if err != nil {
		log.Printf("Error updating chat instructions: %v", err)
		http.Error(w, "Failed to update chat instructions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instructions)
}

// GetChatInstructions gets user's chat instructions
func (s *Handler) GetChatInstructions(userID int) (*models.ChatInstructions, error) {
	query := `
		SELECT id, user_id, instructions, created_at, updated_at
		FROM chat_instructions
		WHERE user_id = $1
	`

	var instructions models.ChatInstructions
	err := s.DB.QueryRow(query, userID).Scan(
		&instructions.ID,
		&instructions.UserID,
		&instructions.Instructions,
		&instructions.CreatedAt,
		&instructions.UpdatedAt,
	)

	return &instructions, err
}

// UpsertChatInstructions creates or updates user's chat instructions
func (s *Handler) UpsertChatInstructions(userID int, instructionsText string) (*models.ChatInstructions, error) {
	query := `
		INSERT INTO chat_instructions (user_id, instructions, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			instructions = EXCLUDED.instructions,
			updated_at = NOW()
		RETURNING id, user_id, instructions, created_at, updated_at
	`

	var instructions models.ChatInstructions
	err := s.DB.QueryRow(query, userID, instructionsText).Scan(
		&instructions.ID,
		&instructions.UserID,
		&instructions.Instructions,
		&instructions.CreatedAt,
		&instructions.UpdatedAt,
	)

	return &instructions, err
}