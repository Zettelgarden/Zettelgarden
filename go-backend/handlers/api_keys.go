package handlers

import (
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"
)

// CreateAPIKey handles POST /api/api-keys - creates a new API key for the authenticated user
func (s *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "API key name is required", http.StatusBadRequest)
		return
	}

	// Check if name is already taken for this user
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND name = $2 AND is_active = true", userID, req.Name).Scan(&count)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "API key with this name already exists", http.StatusConflict)
		return
	}

	// Generate a new API key
	apiKey, err := generateAPIKey()
	if err != nil {
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash the API key for storage
	hashedKey, err := hashAPIKey(apiKey)
	if err != nil {
		http.Error(w, "Failed to hash API key", http.StatusInternalServerError)
		return
	}

	// Store in database
	var apiKeyID int
	err = s.DB.QueryRow(`
		INSERT INTO api_keys (user_id, name, key_hash, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, req.Name, hashedKey, req.Description).Scan(&apiKeyID)

	if err != nil {
		log.Printf("error creating API key: %v", err)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Return the response (key is shown only once)
	response := models.CreateAPIKeyResponse{
		APIKeyResponse: models.APIKeyResponse{
			ID:          apiKeyID,
			Name:        req.Name,
			IsActive:    true,
			Description: req.Description,
		},
		Key: apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListAPIKeys handles GET /api/api-keys - lists all API keys for the authenticated user
func (s *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := s.DB.Query(`
		SELECT id, name, created_at, last_used_at, revoked_at, is_active, description
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apiKeys []models.APIKeyResponse
	for rows.Next() {
		var key models.APIKeyResponse
		err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.CreatedAt,
			&key.LastUsedAt,
			&key.RevokedAt,
			&key.IsActive,
			&key.Description,
		)
		if err != nil {
			http.Error(w, "Error scanning API keys", http.StatusInternalServerError)
			return
		}
		apiKeys = append(apiKeys, key)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error reading API keys", http.StatusInternalServerError)
		return
	}

	response := models.ListAPIKeysResponse{APIKeys: apiKeys}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevokeAPIKey handles DELETE /api/api-keys/{id} - revokes an API key
func (s *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Extract ID from URL path
	path := r.URL.Path
	idStr := path[len("/api/api-keys/"):]
	apiKeyID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid API key ID", http.StatusBadRequest)
		return
	}

	// Verify the API key belongs to the user and revoke it
	result, err := s.DB.Exec(`
		UPDATE api_keys
		SET is_active = false, revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`, apiKeyID, userID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking update result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "API key not found or already revoked", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent) // Success with no content
}
