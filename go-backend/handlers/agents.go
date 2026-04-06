package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// CreateAgentHandler creates a new agent for the authenticated user
func (s *Handler) CreateAgentHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req models.CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate API key
	apiKey, err := utils.GenerateAPIKey()
	if err != nil {
		log.Printf("Error generating API key: %v", err)
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash the key for storage
	keyHash, err := utils.HashAPIKey(apiKey)
	if err != nil {
		log.Printf("Error hashing API key: %v", err)
		http.Error(w, "Failed to hash API key", http.StatusInternalServerError)
		return
	}

	// Create agent user account
	var agentID int
	err = s.GetDB().QueryRow(`
		INSERT INTO users (username, email, password, is_agent, owner_user_id, api_key_hash, created_at, updated_at)
		VALUES ($1, '', '', TRUE, $2, $3, NOW(), NOW())
		RETURNING id
	`, req.Name, userID, keyHash).Scan(&agentID)

	if err != nil {
		log.Printf("Error creating agent: %v", err)
		http.Error(w, "Failed to create agent", http.StatusInternalServerError)
		return
	}

	// Create default cards and tags for agent workspace
	if err := s.createDefaultCards(agentID); err != nil {
		log.Printf("Warning: failed to create default cards for agent: %v", err)
	}
	if err := s.createDefaultTags(agentID); err != nil {
		log.Printf("Warning: failed to create default tags for agent: %v", err)
	}

	// Return response with API key (only time it's shown!)
	response := models.CreateAgentResponse{
		Agent: models.Agent{
			ID:        agentID,
			Name:      req.Name,
			CreatedAt: time.Now(),
			IsActive:  true,
		},
		APIKey: apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListAgentsHandler lists all agents for the authenticated user
func (s *Handler) ListAgentsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := s.GetDB().Query(`
		SELECT id, username, created_at, 
		       (api_key_hash IS NOT NULL) as is_active,
		       last_seen as last_used
		FROM users
		WHERE owner_user_id = $1 AND is_agent = TRUE
		ORDER BY created_at DESC
	`, userID)

	if err != nil {
		log.Printf("Error listing agents: %v", err)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agent models.Agent
		err := rows.Scan(&agent.ID, &agent.Name, &agent.CreatedAt, &agent.IsActive, &agent.LastUsed)
		if err != nil {
			log.Printf("Error scanning agent: %v", err)
			continue
		}
		agents = append(agents, agent)
	}

	if agents == nil {
		agents = []models.Agent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
	})
}

// RevokeAgentHandler revokes an agent's API key
func (s *Handler) RevokeAgentHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Verify ownership
	var ownerID int
	err = s.GetDB().QueryRow(`SELECT owner_user_id FROM users WHERE id = $1 AND is_agent = TRUE`, agentID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error checking agent ownership: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Revoke by setting API key hash to NULL
	_, err = s.GetDB().Exec(`UPDATE users SET api_key_hash = NULL WHERE id = $1`, agentID)
	if err != nil {
		log.Printf("Error revoking agent: %v", err)
		http.Error(w, "Failed to revoke agent", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAgentActivityHandler returns activity log for an agent
func (s *Handler) GetAgentActivityHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Verify ownership
	var ownerID int
	err = s.GetDB().QueryRow(`SELECT owner_user_id FROM users WHERE id = $1 AND is_agent = TRUE`, agentID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get pagination params
	page := 1
	perPage := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 && parsed <= 100 {
			perPage = parsed
		}
	}

	logs, total, err := services.GetAgentActivity(s.GetDB(), agentID, page, perPage)
	if err != nil {
		log.Printf("Error getting agent activity: %v", err)
		http.Error(w, "Failed to get activity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
		"pagination": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": (total + perPage - 1) / perPage,
		},
	})
}
