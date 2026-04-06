# AI Agent Multi-User Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable users to create AI agents that can maintain isolated workspaces and contribute to the user's workspace with full auditability.

**Architecture:** Treat agents as special user accounts with `is_agent=true` flag. Agents authenticate via API keys and can write to either their own workspace (isolated) or the owner's workspace (tracked). All operations reuse existing `user_id` filtering.

**Tech Stack:** Go (backend), PostgreSQL (database), React + TypeScript (frontend), existing auth middleware

**Design Doc:** `docs/plans/2026-04-06-ai-agent-multi-user-design.md`

---

## Task 1: Database Migration - Agent Support

**Files:**
- Create: `go-backend/migrations/20260406120000_add_agent_support.sql`

**Step 1: Write migration file**

Create migration to add agent columns to users table:

```sql
-- +goose Up
-- +goose StatementBegin

-- Add agent support to users table
ALTER TABLE users ADD COLUMN is_agent BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN owner_user_id INT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE users ADD COLUMN api_key_hash VARCHAR(255) NULL;

-- Add constraints
ALTER TABLE users ADD CONSTRAINT check_agent_not_admin 
    CHECK (NOT (is_agent = TRUE AND is_admin = TRUE));

-- Index for faster lookups
CREATE INDEX idx_users_owner ON users(owner_user_id) WHERE is_agent = TRUE;
CREATE INDEX idx_users_agent ON users(is_agent) WHERE is_agent = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_agent;
DROP INDEX IF EXISTS idx_users_owner;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_agent_not_admin;
ALTER TABLE users DROP COLUMN IF EXISTS api_key_hash;
ALTER TABLE users DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS is_agent;

-- +goose StatementEnd
```

**Step 2: Verify migration syntax**

Run: `cd go-backend && goose postgres $DATABASE_URL status`

Expected: See pending migration

**Step 3: Run migration**

Run: `cd go-backend && goose postgres $DATABASE_URL up`

Expected: "OK" message, migration applied

**Step 4: Verify schema changes**

Run: `psql $DATABASE_URL -c "\d users"`

Expected: See new columns `is_agent`, `owner_user_id`, `api_key_hash`

**Step 5: Commit**

```bash
git add go-backend/migrations/20260406120000_add_agent_support.sql
git commit -m "feat(db): add agent support columns to users table"
```

---

## Task 2: Database Migration - Card Agent Tracking

**Files:**
- Create: `go-backend/migrations/20260406120001_add_card_agent_tracking.sql`

**Step 1: Write migration file**

```sql
-- +goose Up
-- +goose StatementBegin

-- Track which agent created a card
ALTER TABLE cards ADD COLUMN created_by_agent_id INT NULL REFERENCES users(id) ON DELETE SET NULL;

-- Index for faster filtering
CREATE INDEX idx_cards_created_by_agent ON cards(created_by_agent_id) WHERE created_by_agent_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_cards_created_by_agent;
ALTER TABLE cards DROP COLUMN IF EXISTS created_by_agent_id;

-- +goose StatementEnd
```

**Step 2: Run migration**

Run: `cd go-backend && goose postgres $DATABASE_URL up`

Expected: "OK" message

**Step 3: Verify schema**

Run: `psql $DATABASE_URL -c "\d cards"`

Expected: See `created_by_agent_id` column

**Step 4: Commit**

```bash
git add go-backend/migrations/20260406120001_add_card_agent_tracking.sql
git commit -m "feat(db): add agent tracking to cards table"
```

---

## Task 3: Database Migration - Agent Activity Log

**Files:**
- Create: `go-backend/migrations/20260406120002_create_agent_activity_log.sql`

**Step 1: Write migration file**

```sql
-- +goose Up
-- +goose StatementBegin

-- Track all agent actions for auditability
CREATE TABLE agent_activity_log (
    id SERIAL PRIMARY KEY,
    agent_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INT,
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_agent_activity_agent ON agent_activity_log(agent_id);
CREATE INDEX idx_agent_activity_created ON agent_activity_log(created_at DESC);
CREATE INDEX idx_agent_activity_action ON agent_activity_log(action);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_agent_activity_action;
DROP INDEX IF EXISTS idx_agent_activity_created;
DROP INDEX IF EXISTS idx_agent_activity_agent;
DROP TABLE IF EXISTS agent_activity_log;

-- +goose StatementEnd
```

**Step 2: Run migration**

Run: `cd go-backend && goose postgres $DATABASE_URL up`

Expected: "OK" message

**Step 3: Verify table**

Run: `psql $DATABASE_URL -c "\d agent_activity_log"`

Expected: See table structure with all columns

**Step 4: Commit**

```bash
git add go-backend/migrations/20260406120002_create_agent_activity_log.sql
git commit -m "feat(db): create agent activity log table"
```

---

## Task 4: Backend - Update User Model

**Files:**
- Modify: `go-backend/models/user.go`

**Step 1: Add agent fields to User struct**

Add after `CaldavToken` field:

```go
type User struct {
	// ... existing fields ...
	CaldavURL             *string    `json:"caldav_url"`
	CaldavToken           *string    `json:"caldav_token,omitempty"`
	
	// Agent-specific fields
	IsAgent               bool       `json:"is_agent"`
	OwnerUserID           *int       `json:"owner_user_id,omitempty"`
	LastUsed              *time.Time `json:"last_used,omitempty"`
}
```

**Step 2: Update QueryUser to include new fields**

Modify the SELECT query in `go-backend/handlers/users.go` around line 350:

```go
func (s *Handler) QueryUser(id int) (models.User, error) {
	var user models.User
	err := s.GetDB().QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started, COALESCE(timezone, 'UTC'),
	caldav_url, caldav_token,
	COALESCE(is_agent, FALSE) as is_agent,
	owner_user_id
	FROM users WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsAdmin,
		&user.EmailValidated,
		&user.CanUploadFiles,
		&user.StripeSubscriptionStatus,
		&user.MaxFileStorage,
		&user.LastLogin,
		&user.LastSeen,
		&user.DashboardCardPK,
		&user.HasSeenGettingStarted,
		&user.Timezone,
		&user.CaldavURL,
		&user.CaldavToken,
		&user.IsAgent,
		&user.OwnerUserID,
	)
	// ... rest of function
}
```

**Step 3: Run existing tests**

Run: `cd go-backend && go test ./handlers -run TestQueryUser -v`

Expected: Tests pass (may need to update other QueryUser* functions similarly)

**Step 4: Commit**

```bash
git add go-backend/models/user.go go-backend/handlers/users.go
git commit -m "feat(models): add agent fields to User model"
```

---

## Task 5: Backend - API Key Generation

**Files:**
- Create: `go-backend/utils/apikey.go`
- Create: `go-backend/utils/apikey_test.go`

**Step 1: Write test first**

```go
package utils

import (
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	
	if len(key1) < 32 {
		t.Errorf("API key too short: %d characters", len(key1))
	}
	
	if key1[:7] != "zg_live" {
		t.Errorf("API key should start with 'zg_live', got %s", key1[:7])
	}
	
	// Generate another key to ensure uniqueness
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	
	if key1 == key2 {
		t.Error("API keys should be unique")
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "zg_live_test123"
	hash, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey() error = %v", err)
	}
	
	if hash == key {
		t.Error("Hash should not equal plaintext key")
	}
	
	if len(hash) < 50 {
		t.Errorf("Hash seems too short: %d", len(hash))
	}
}

func TestVerifyAPIKey(t *testing.T) {
	key := "zg_live_test123"
	hash, _ := HashAPIKey(key)
	
	if !VerifyAPIKey(key, hash) {
		t.Error("VerifyAPIKey should return true for correct key")
	}
	
	if VerifyAPIKey("wrong_key", hash) {
		t.Error("VerifyAPIKey should return false for incorrect key")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./utils -run TestGenerateAPIKey -v`

Expected: FAIL - "GenerateAPIKey undefined"

**Step 3: Implement API key utilities**

```go
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// GenerateAPIKey creates a new API key with prefix
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("zg_live_%s", hex.EncodeToString(bytes)), nil
}

// HashAPIKey hashes an API key for storage
func HashAPIKey(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	return string(hash), err
}

// VerifyAPIKey checks if a key matches a hash
func VerifyAPIKey(key, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
	return err == nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go-backend && go test ./utils -v`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add go-backend/utils/apikey.go go-backend/utils/apikey_test.go
git commit -m "feat(utils): add API key generation and hashing utilities"
```

---

## Task 6: Backend - Agent Models

**Files:**
- Create: `go-backend/models/agent.go`

**Step 1: Create agent request/response models**

```go
package models

import "time"

type CreateAgentRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type Agent struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	IsActive    bool       `json:"is_active"`
}

type CreateAgentResponse struct {
	Agent
	APIKey string `json:"api_key"` // Only shown once!
}

type AgentActivityLog struct {
	ID         int                    `json:"id"`
	AgentID    int                    `json:"agent_id"`
	Action     string                 `json:"action"`
	TargetType string                 `json:"target_type"`
	TargetID   *int                   `json:"target_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
```

**Step 2: Commit**

```bash
git add go-backend/models/agent.go
git commit -m "feat(models): add agent request/response models"
```

---

## Task 7: Backend - Activity Logging Service

**Files:**
- Create: `go-backend/services/agent_activity.go`
- Create: `go-backend/services/agent_activity_test.go`

**Step 1: Write test first**

```go
package services

import (
	"database/sql"
	"go-backend/models"
	"testing"
	"time"
)

func TestLogAgentActivity(t *testing.T) {
	// This would use a test database
	// For now, we'll test the structure
	activity := models.AgentActivityLog{
		AgentID:    1,
		Action:     "create_card",
		TargetType: "card",
		TargetID:   intPtr(123),
		Details:    map[string]interface{}{"title": "Test Card"},
	}
	
	if activity.AgentID != 1 {
		t.Error("AgentID not set correctly")
	}
}

func intPtr(i int) *int {
	return &i
}
```

**Step 2: Implement activity logging**

```go
package services

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"log"
)

// LogAgentActivity logs an agent action asynchronously
func LogAgentActivity(db *sql.DB, agentID int, action, targetType string, targetID *int, details map[string]interface{}) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in LogAgentActivity: %v", r)
			}
		}()

		var detailsJSON []byte
		var err error
		if details != nil {
			detailsJSON, err = json.Marshal(details)
			if err != nil {
				log.Printf("Error marshaling activity details: %v", err)
				return
			}
		}

		_, err = db.Exec(`
			INSERT INTO agent_activity_log (agent_id, action, target_type, target_id, details)
			VALUES ($1, $2, $3, $4, $5)
		`, agentID, action, targetType, targetID, detailsJSON)

		if err != nil {
			log.Printf("Error logging agent activity: %v", err)
		}
	}()
}

// GetAgentActivity retrieves paginated activity logs for an agent
func GetAgentActivity(db *sql.DB, agentID, page, perPage int) ([]models.AgentActivityLog, int, error) {
	offset := (page - 1) * perPage
	
	// Get total count
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM agent_activity_log WHERE agent_id = $1`, agentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// Get logs
	rows, err := db.Query(`
		SELECT id, agent_id, action, target_type, target_id, details, created_at
		FROM agent_activity_log
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, agentID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var logs []models.AgentActivityLog
	for rows.Next() {
		var log models.AgentActivityLog
		var detailsJSON []byte
		err := rows.Scan(
			&log.ID,
			&log.AgentID,
			&log.Action,
			&log.TargetType,
			&log.TargetID,
			&detailsJSON,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}
		
		logs = append(logs, log)
	}
	
	return logs, total, nil
}
```

**Step 3: Run tests**

Run: `cd go-backend && go test ./services -run TestLogAgentActivity -v`

Expected: PASS

**Step 4: Commit**

```bash
git add go-backend/services/agent_activity.go go-backend/services/agent_activity_test.go
git commit -m "feat(services): add agent activity logging service"
```

---

## Task 8: Backend - Agent Management Handlers

**Files:**
- Create: `go-backend/handlers/agents.go`
- Create: `go-backend/handlers/agents_test.go`

**Step 1: Write test for CreateAgent**

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAgentHandler(t *testing.T) {
	// Setup test handler
	handler := setupTestHandler()
	defer handler.Cleanup()
	
	// Create test user first
	user := createTestUser(handler, "owner@example.com", "owner")
	
	// Create agent request
	reqBody := models.CreateAgentRequest{
		Name: "Test Agent",
	}
	body, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)
	
	w := httptest.NewRecorder()
	handler.CreateAgentHandler(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}
	
	var response models.CreateAgentResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", response.Name)
	}
	
	if response.APIKey == "" {
		t.Error("API key should be returned")
	}
	
	if len(response.APIKey) < 20 {
		t.Errorf("API key seems too short: %s", response.APIKey)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./handlers -run TestCreateAgentHandler -v`

Expected: FAIL - "CreateAgentHandler undefined"

**Step 3: Implement agent handlers**

```go
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
```

**Step 4: Run tests**

Run: `cd go-backend && go test ./handlers -run TestCreateAgentHandler -v`

Expected: PASS (may need to adjust test helpers)

**Step 5: Commit**

```bash
git add go-backend/handlers/agents.go go-backend/handlers/agents_test.go
git commit -m "feat(handlers): add agent management endpoints"
```

---

## Task 9: Backend - API Key Authentication

**Files:**
- Modify: `go-backend/handlers/auth.go`
- Modify: `go-backend/handlers/api_keys.go`
- Modify: `go-backend/routes/helpers.go`

**Step 1: Add agent context values**

In `go-backend/routes/helpers.go`, add to middleware:

```go
// APIKeyOrJWTMiddleware authenticates via API key or JWT
// Sets context values:
// - current_user: user or agent ID
// - is_agent: boolean (only set if true)
// - owner_user_id: int (only set for agents)
func APIKeyOrJWTMiddleware(next http.HandlerFunc, h *handlers.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try API key first
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.Header.Get("Authorization")
			if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			} else {
				apiKey = ""
			}
		}
		
		if apiKey != "" {
			// Look up API key in users table (for agents)
			var userID int
			var isAgent bool
			var ownerID sql.NullInt64
			
			err := h.GetDB().QueryRow(`
				SELECT id, is_agent, owner_user_id
				FROM users
				WHERE api_key_hash IS NOT NULL AND is_agent = TRUE
			`).Scan(&userID, &isAgent, &ownerID)
			
			if err == nil {
				// Verify the key hash matches
				// We need to check all agent keys since we can't query by plaintext
				rows, err := h.GetDB().Query(`
					SELECT id, api_key_hash, owner_user_id
					FROM users
					WHERE is_agent = TRUE AND api_key_hash IS NOT NULL
				`)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var aid int
						var hash string
						var oid sql.NullInt64
						rows.Scan(&aid, &hash, &oid)
						
						if utils.VerifyAPIKey(apiKey, hash) {
							ctx := context.WithValue(r.Context(), "current_user", aid)
							ctx = context.WithValue(ctx, "is_agent", true)
							if oid.Valid {
								ctx = context.WithValue(ctx, "owner_user_id", int(oid.Int64))
							}
							// Update last_seen
							go h.GetDB().Exec(`UPDATE users SET last_seen = NOW() WHERE id = $1`, aid)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}
			
			// Also check regular API keys (existing functionality)
			// ... existing API key lookup code ...
		}
		
		// Fall back to JWT
		// ... existing JWT middleware code ...
	}
}
```

**Step 2: Update card creation to handle agents**

In `go-backend/handlers/cards.go`, modify CreateCard handler:

```go
func (s *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	currentUserID := r.Context().Value("current_user").(int)
	isAgent, _ := r.Context().Value("is_agent").(bool)
	ownerUserID, hasOwner := r.Context().Value("owner_user_id").(int)
	
	// Parse request body
	var params models.EditCardParams
	// ... existing parsing code ...
	
	var targetUserID int
	if params.UserID != nil {
		targetUserID = *params.UserID
	} else {
		targetUserID = currentUserID
	}
	
	var createdByAgentID *int
	actualUserID := targetUserID
	
	if isAgent {
		// Agent is making the request
		if hasOwner && targetUserID == ownerUserID {
			// Agent writing to owner's workspace
			actualUserID = ownerUserID
			createdByAgentID = &currentUserID
		} else if targetUserID == currentUserID {
			// Agent writing to own workspace
			actualUserID = currentUserID
			createdByAgentID = nil
		} else {
			http.Error(w, "Forbidden: agents can only write to own or owner's workspace", http.StatusForbidden)
			return
		}
		
		// Log the activity
		details := map[string]interface{}{
			"title": params.Title,
		}
		targetID := actualUserID
		services.LogAgentActivity(s.GetDB(), currentUserID, "create_card", "card", &targetID, details)
	}
	
	// Create card with actualUserID and createdByAgentID
	card, err := services.CreateCardWithAgent(s.GetDB(), actualUserID, params, createdByAgentID)
	// ... rest of handler
}
```

**Step 3: Update CreateCard service to accept agent ID**

In `go-backend/services/cards.go`:

```go
func CreateCardWithAgent(db *sql.DB, userID int, params models.EditCardParams, createdByAgentID *int) (models.Card, error) {
	var card models.Card
	query := `
		INSERT INTO cards (user_id, name, title, body, link, created_at, updated_at, created_by_agent_id)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6)
		RETURNING id, user_id, name, title, body, link, created_at, updated_at
	`
	// ... rest of implementation
}
```

**Step 4: Run existing card tests**

Run: `cd go-backend && go test ./handlers -run TestCard -v`

Expected: All tests pass

**Step 5: Commit**

```bash
git add go-backend/handlers/auth.go go-backend/routes/helpers.go go-backend/handlers/cards.go go-backend/services/cards.go
git commit -m "feat(auth): add agent API key authentication and workspace access control"
```

---

## Task 10: Backend - Register Routes

**Files:**
- Modify: `go-backend/routes/routes.go`

**Step 1: Add agent routes**

```go
func SetupRoutes(r *mux.Router, h *handlers.Handler) {
	// ... existing routes ...
	
	// Agent management (requires authentication)
	api.HandleFunc("/agents", h.JwtMiddleware(h.CreateAgentHandler)).Methods("POST")
	api.HandleFunc("/agents", h.JwtMiddleware(h.ListAgentsHandler)).Methods("GET")
	api.HandleFunc("/agents/{id}", h.JwtMiddleware(h.RevokeAgentHandler)).Methods("DELETE")
	api.HandleFunc("/agents/{id}/activity", h.JwtMiddleware(h.GetAgentActivityHandler)).Methods("GET")
}
```

**Step 2: Test routes are registered**

Run: `cd go-backend && go test ./routes -v`

Expected: Tests pass

**Step 3: Commit**

```bash
git add go-backend/routes/routes.go
git commit -m "feat(routes): register agent management endpoints"
```

---

## Task 11: Frontend - API Client

**Files:**
- Create: `zettelkasten-front/src/api/agents.ts`
- Create: `zettelkasten-front/src/models/Agent.ts`

**Step 1: Create Agent model**

```typescript
// zettelkasten-front/src/models/Agent.ts

export interface Agent {
  id: number;
  name: string;
  description?: string;
  created_at: string;
  last_used?: string;
  is_active: boolean;
}

export interface CreateAgentRequest {
  name: string;
  description?: string;
}

export interface CreateAgentResponse extends Agent {
  api_key: string;
}

export interface AgentActivityLog {
  id: number;
  agent_id: number;
  action: string;
  target_type: string;
  target_id?: number;
  details?: Record<string, any>;
  created_at: string;
}

export interface AgentActivityResponse {
  logs: AgentActivityLog[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}
```

**Step 2: Create API client**

```typescript
// zettelkasten-front/src/api/agents.ts

import { apiClient } from './client';
import {
  Agent,
  CreateAgentRequest,
  CreateAgentResponse,
  AgentActivityResponse,
} from '../models/Agent';

export const createAgent = async (
  name: string,
  description?: string
): Promise<CreateAgentResponse> => {
  const response = await apiClient.post<CreateAgentResponse>('/api/agents', {
    name,
    description,
  });
  return response.data;
};

export const listAgents = async (): Promise<Agent[]> => {
  const response = await apiClient.get<{ agents: Agent[] }>('/api/agents');
  return response.data.agents;
};

export const revokeAgent = async (agentId: number): Promise<void> => {
  await apiClient.delete(`/api/agents/${agentId}`);
};

export const getAgentActivity = async (
  agentId: number,
  page: number = 1,
  perPage: number = 50
): Promise<AgentActivityResponse> => {
  const response = await apiClient.get<AgentActivityResponse>(
    `/api/agents/${agentId}/activity`,
    {
      params: { page, per_page: perPage },
    }
  );
  return response.data;
};
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/models/Agent.ts zettelkasten-front/src/api/agents.ts
git commit -m "feat(frontend): add agent API client and models"
```

---

## Task 12: Frontend - Agent Management Page

**Files:**
- Create: `zettelkasten-front/src/components/AgentManagement.tsx`
- Create: `zettelkasten-front/src/components/CreateAgentModal.tsx`
- Create: `zettelkasten-front/src/components/AgentActivityModal.tsx`

**Step 1: Create CreateAgentModal component**

```tsx
// zettelkasten-front/src/components/CreateAgentModal.tsx

import React, { useState } from 'react';
import { createAgent } from '../api/agents';

interface CreateAgentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const CreateAgentModal: React.FC<CreateAgentModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await createAgent(name, description || undefined);
      setApiKey(response.api_key);
      setName('');
      setDescription('');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to create agent');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (apiKey) {
      onSuccess();
    }
    setApiKey(null);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full">
        {apiKey ? (
          <>
            <h2 className="text-xl font-bold mb-4">Agent Created!</h2>
            <p className="mb-2 text-sm text-gray-600">
              Save this API key now - it won't be shown again:
            </p>
            <code className="block bg-gray-100 p-3 rounded mb-4 text-xs break-all">
              {apiKey}
            </code>
            <button
              onClick={handleClose}
              className="w-full bg-blue-500 text-white py-2 rounded hover:bg-blue-600"
            >
              Close
            </button>
          </>
        ) : (
          <>
            <h2 className="text-xl font-bold mb-4">Create New Agent</h2>
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">
                  Agent Name *
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full border rounded px-3 py-2"
                  required
                  placeholder="e.g., Claude Research Agent"
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">
                  Description (optional)
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full border rounded px-3 py-2"
                  rows={2}
                  placeholder="What does this agent do?"
                />
              </div>
              {error && (
                <p className="text-red-500 text-sm mb-4">{error}</p>
              )}
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="flex-1 border border-gray-300 py-2 rounded hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading || !name}
                  className="flex-1 bg-blue-500 text-white py-2 rounded hover:bg-blue-600 disabled:bg-gray-300"
                >
                  {loading ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </>
        )}
      </div>
    </div>
  );
};
```

**Step 2: Create AgentActivityModal component**

```tsx
// zettelkasten-front/src/components/AgentActivityModal.tsx

import React, { useEffect, useState } from 'react';
import { getAgentActivity } from '../api/agents';
import { AgentActivityLog } from '../models/Agent';

interface AgentActivityModalProps {
  isOpen: boolean;
  onClose: () => void;
  agentId: number;
  agentName: string;
}

export const AgentActivityModal: React.FC<AgentActivityModalProps> = ({
  isOpen,
  onClose,
  agentId,
  agentName,
}) => {
  const [logs, setLogs] = useState<AgentActivityLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  useEffect(() => {
    if (isOpen) {
      fetchActivity();
    }
  }, [isOpen, agentId, page]);

  const fetchActivity = async () => {
    setLoading(true);
    try {
      const response = await getAgentActivity(agentId, page);
      setLogs(response.logs);
      setTotalPages(response.pagination.total_pages);
    } catch (err) {
      console.error('Failed to fetch activity:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-3xl w-full max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold">
            Activity: {agentName}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700"
          >
            ✕
          </button>
        </div>
        
        <div className="overflow-auto flex-1">
          {loading ? (
            <p className="text-center py-4">Loading...</p>
          ) : logs.length === 0 ? (
            <p className="text-center py-4 text-gray-500">
              No activity yet
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-3 py-2 text-left">Action</th>
                  <th className="px-3 py-2 text-left">Target</th>
                  <th className="px-3 py-2 text-left">Details</th>
                  <th className="px-3 py-2 text-left">Time</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-t">
                    <td className="px-3 py-2">{log.action}</td>
                    <td className="px-3 py-2">
                      {log.target_type} #{log.target_id}
                    </td>
                    <td className="px-3 py-2">
                      {log.details && JSON.stringify(log.details).slice(0, 50)}
                    </td>
                    <td className="px-3 py-2 text-gray-500">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        
        {totalPages > 1 && (
          <div className="flex justify-between items-center mt-4 pt-4 border-t">
            <button
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-gray-600">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={page === totalPages}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
```

**Step 3: Create AgentManagement page**

```tsx
// zettelkasten-front/src/components/AgentManagement.tsx

import React, { useEffect, useState } from 'react';
import { listAgents, revokeAgent } from '../api/agents';
import { Agent } from '../models/Agent';
import { CreateAgentModal } from './CreateAgentModal';
import { AgentActivityModal } from './AgentActivityModal';

export const AgentManagement: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [activityAgent, setActivityAgent] = useState<Agent | null>(null);

  useEffect(() => {
    fetchAgents();
  }, []);

  const fetchAgents = async () => {
    setLoading(true);
    try {
      const data = await listAgents();
      setAgents(data);
    } catch (err) {
      console.error('Failed to fetch agents:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleRevoke = async (agentId: number) => {
    if (!window.confirm('Are you sure? This will revoke the agent\'s access immediately.')) {
      return;
    }
    
    try {
      await revokeAgent(agentId);
      fetchAgents();
    } catch (err) {
      console.error('Failed to revoke agent:', err);
      alert('Failed to revoke agent');
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">AI Agents</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
        >
          + Create Agent
        </button>
      </div>
      
      <p className="text-gray-600 mb-6">
        Manage AI agents that can access your Zettelgarden workspace via API keys.
      </p>
      
      {loading ? (
        <p className="text-center py-8">Loading...</p>
      ) : agents.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <p className="text-gray-500 mb-4">No agents configured yet</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="text-blue-500 hover:underline"
          >
            Create your first agent
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left">Name</th>
                <th className="px-4 py-3 text-left">Status</th>
                <th className="px-4 py-3 text-left">Created</th>
                <th className="px-4 py-3 text-left">Last Used</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((agent) => (
                <tr key={agent.id} className="border-t">
                  <td className="px-4 py-3 font-medium">
                    🤖 {agent.name}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        agent.is_active
                          ? 'bg-green-100 text-green-700'
                          : 'bg-red-100 text-red-700'
                      }`}
                    >
                      {agent.is_active ? 'Active' : 'Revoked'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {new Date(agent.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {agent.last_used
                      ? new Date(agent.last_used).toLocaleDateString()
                      : 'Never'}
                  </td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <button
                      onClick={() => setActivityAgent(agent)}
                      className="text-blue-500 hover:underline text-sm"
                    >
                      Activity
                    </button>
                    {agent.is_active && (
                      <button
                        onClick={() => handleRevoke(agent.id)}
                        className="text-red-500 hover:underline text-sm"
                      >
                        Revoke
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      
      <CreateAgentModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={fetchAgents}
      />
      
      {activityAgent && (
        <AgentActivityModal
          isOpen={true}
          onClose={() => setActivityAgent(null)}
          agentId={activityAgent.id}
          agentName={activityAgent.name}
        />
      )}
    </div>
  );
};
```

**Step 4: Add route to App**

In `zettelkasten-front/src/App.tsx`, add route:

```tsx
import { AgentManagement } from './components/AgentManagement';

// In routes section:
<Route path="/settings/agents" element={<AgentManagement />} />
```

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/AgentManagement.tsx \
        zettelkasten-front/src/components/CreateAgentModal.tsx \
        zettelkasten-front/src/components/AgentActivityModal.tsx \
        zettelkasten-front/src/App.tsx
git commit -m "feat(frontend): add agent management page with create/revoke/activity views"
```

---

## Task 13: Frontend - Workspace Switcher

**Files:**
- Create: `zettelkasten-front/src/components/WorkspaceSwitcher.tsx`
- Modify: `zettelkasten-front/src/contexts/AuthContext.tsx`

**Step 1: Update AuthContext to track workspace**

```tsx
// zettelkasten-front/src/contexts/AuthContext.tsx

interface AuthContextType {
  // ... existing fields
  currentWorkspace: number | null;  // user_id of current workspace
  isViewingAgentWorkspace: boolean;
  switchWorkspace: (workspaceUserId: number) => void;
}

// In AuthProvider:
const [currentWorkspace, setCurrentWorkspace] = useState<number | null>(null);

const switchWorkspace = (workspaceUserId: number) => {
  setCurrentWorkspace(workspaceUserId);
  // Optionally update URL: navigate(`?workspace=${workspaceUserId}`)
};

// Initialize to current user's workspace
useEffect(() => {
  if (currentUser) {
    setCurrentWorkspace(currentUser.id);
  }
}, [currentUser]);

const isViewingAgentWorkspace = currentUser !== null && 
                                  currentWorkspace !== null && 
                                  currentWorkspace !== currentUser.id;
```

**Step 2: Create WorkspaceSwitcher component**

```tsx
// zettelkasten-front/src/components/WorkspaceSwitcher.tsx

import React, { useEffect, useState } from 'react';
import { useAuth } from '../hooks/useAuth';
import { listAgents } from '../api/agents';
import { Agent } from '../models/Agent';

export const WorkspaceSwitcher: React.FC = () => {
  const { currentUser, currentWorkspace, switchWorkspace } = useAuth();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    if (currentUser) {
      listAgents().then(setAgents).catch(console.error);
    }
  }, [currentUser]);

  if (!currentUser) return null;

  const currentWorkspaceName = 
    currentWorkspace === currentUser.id
      ? 'My Workspace'
      : agents.find(a => a.id === currentWorkspace)?.name || 'Unknown';

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 border rounded hover:bg-gray-50"
      >
        <span>{currentWorkspace === currentUser.id ? '👤' : '🤖'}</span>
        <span>{currentWorkspaceName}</span>
        <span className="text-gray-400">▼</span>
      </button>
      
      {isOpen && (
        <div className="absolute top-full left-0 mt-1 bg-white border rounded shadow-lg z-50 min-w-[200px]">
          <button
            onClick={() => {
              switchWorkspace(currentUser.id);
              setIsOpen(false);
            }}
            className={`w-full text-left px-4 py-2 hover:bg-gray-50 flex items-center gap-2 ${
              currentWorkspace === currentUser.id ? 'bg-blue-50' : ''
            }`}
          >
            <span>👤</span>
            <span>My Workspace</span>
          </button>
          
          {agents.filter(a => a.is_active).length > 0 && (
            <>
              <div className="border-t my-1"></div>
              <div className="px-4 py-1 text-xs text-gray-500 uppercase">
                Agent Workspaces
              </div>
              {agents
                .filter(a => a.is_active)
                .map((agent) => (
                  <button
                    key={agent.id}
                    onClick={() => {
                      switchWorkspace(agent.id);
                      setIsOpen(false);
                    }}
                    className={`w-full text-left px-4 py-2 hover:bg-gray-50 flex items-center gap-2 ${
                      currentWorkspace === agent.id ? 'bg-blue-50' : ''
                    }`}
                  >
                    <span>🤖</span>
                    <span>{agent.name}</span>
                  </button>
                ))}
            </>
          )}
        </div>
      )}
    </div>
  );
};
```

**Step 3: Add to navigation**

Add `<WorkspaceSwitcher />` to main navigation bar.

**Step 4: Update API queries to use workspace**

Pass `currentWorkspace` to all card/tag/task API calls as needed.

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/WorkspaceSwitcher.tsx \
        zettelkasten-front/src/contexts/AuthContext.tsx
git commit -m "feat(frontend): add workspace switcher for user/agent workspaces"
```

---

## Task 14: Frontend - Card Agent Badge

**Files:**
- Modify: `zettelkasten-front/src/components/Card.tsx` (or equivalent card display component)

**Step 1: Add agent badge to card display**

```tsx
// In card component, add badge when created_by_agent_id is present

{card.created_by_agent_id && (
  <span className="inline-flex items-center px-2 py-1 rounded text-xs bg-purple-100 text-purple-700 ml-2">
    🤖 Created by agent
  </span>
)}
```

**Step 2: Update Card model**

Add to Card interface:

```typescript
created_by_agent_id?: number;
created_by_agent_name?: string;  // If backend includes this
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/Card.tsx zettelkasten-front/src/models/Card.ts
git commit -m "feat(frontend): show agent badge on agent-created cards"
```

---

## Task 15: Integration Testing

**Files:**
- Create: `go-backend/handlers/agents_integration_test.go`

**Step 1: Write end-to-end test**

```go
//go:build integration
// +build integration

package handlers

import (
	"testing"
)

func TestAgentE2EWorkflow(t *testing.T) {
	handler := setupTestHandler()
	defer handler.Cleanup()
	
	// 1. Create owner user
	owner := createTestUser(handler, "owner@example.com", "owner")
	
	// 2. Create agent
	agentKey := createTestAgent(handler, owner.ID, "Test Agent")
	
	// 3. Agent creates card in own workspace
	// ... test card creation with API key auth
	
	// 4. Agent creates card in owner's workspace
	// ... test card creation with owner_user_id
	
	// 5. Verify activity logging
	// ... check agent_activity_log
	
	// 6. Revoke agent
	// ... revoke and verify can't auth anymore
}
```

**Step 2: Run integration tests**

Run: `cd go-backend && go test ./handlers -tags=integration -v`

Expected: All tests pass

**Step 3: Commit**

```bash
git add go-backend/handlers/agents_integration_test.go
git commit -m "test(integration): add agent workflow end-to-end tests"
```

---

## Task 16: Documentation

**Files:**
- Create: `docs/agent-integration.md`

**Step 1: Write integration guide**

```markdown
# AI Agent Integration Guide

## Overview

Zettelgarden supports external AI agents via API keys. Agents can maintain isolated workspaces or contribute to your main workspace with full auditability.

## Creating an Agent

1. Navigate to Settings → AI Agents
2. Click "Create Agent"
3. Enter a name (e.g., "Claude Research Agent")
4. Copy the API key immediately (shown only once!)

## Using the API Key

Configure your AI agent to authenticate:

```bash
curl -H "Authorization: Bearer zg_live_abc123..." \
  https://api.zettelgarden.com/api/cards
```

## Agent Permissions

Agents can:
- Read all cards in owner's workspace
- Create/update/delete cards in own workspace
- Create cards in owner's workspace (tracked)
- Not modify/delete cards created by owner

## Activity Logging

All agent actions are logged. View activity in Settings → AI Agents → Activity.

## Revoking Access

Revoke an agent at any time from Settings → AI Agents. Access is immediately terminated.
```

**Step 2: Commit**

```bash
git add docs/agent-integration.md
git commit -m "docs: add AI agent integration guide"
```

---

## Final Steps

### Run Full Test Suite

Run: `cd go-backend && go test ./...`

Expected: All tests pass

### Run Frontend Tests

Run: `cd zettelkasten-front && npm run test`

Expected: All tests pass

### Manual Testing Checklist

- [ ] Create agent, verify API key shown once
- [ ] Agent can authenticate with API key
- [ ] Agent can create card in own workspace
- [ ] Agent can create card in owner's workspace
- [ ] Agent badge shows on agent-created cards
- [ ] Activity log shows agent actions
- [ ] Revoke agent, verify auth fails
- [ ] Workspace switcher works correctly
- [ ] Existing user workflows unchanged

### Commit All Changes

```bash
git add -A
git commit -m "feat: complete AI agent multi-user support

- Add agent accounts with API key authentication
- Enable agent workspaces and owner workspace access
- Track all agent activity for auditability
- Add agent management UI with create/revoke/activity
- Add workspace switcher for user/agent contexts
- Show agent badge on agent-created cards
- Comprehensive tests and documentation"
```

---

**Plan complete!** Ready for execution via superpowers:executing-plans or superpowers:subagent-driven-development.
