package services

import (
	"go-backend/tests"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestLogAgentActivity inserts a record asynchronously
func TestLogAgentActivity(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Create a test agent (user with is_agent = TRUE)
	var agentID int
	err := s.DB.QueryRow(`
		INSERT INTO users (username, email, password, is_agent, api_key_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, "Test Agent", "agent@test.com", "", true, "test-hash").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	targetID := 42
	details := map[string]interface{}{
		"card_title": "Test Card",
		"changes":    []string{"title", "body"},
	}

	// Log activity asynchronously
	LogAgentActivity(s.DB, agentID, "create_card", "card", &targetID, details)

	// Wait for async goroutine to complete (increased timeout for reliability)
	time.Sleep(200 * time.Millisecond)

	// Verify the record was inserted
	var count int
	err = s.DB.QueryRow(`
		SELECT COUNT(*) FROM agent_activity_log WHERE agent_id = $1
	`, agentID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count activity logs: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 activity log, got %d", count)
	}

	// Verify the record contents
	var action, targetType string
	var retrievedTargetID int
	err = s.DB.QueryRow(`
		SELECT action, target_type, target_id 
		FROM agent_activity_log 
		WHERE agent_id = $1
	`, agentID).Scan(&action, &targetType, &retrievedTargetID)
	if err != nil {
		t.Fatalf("failed to retrieve activity log: %v", err)
	}

	if action != "create_card" {
		t.Errorf("expected action 'create_card', got '%s'", action)
	}
	if targetType != "card" {
		t.Errorf("expected target_type 'card', got '%s'", targetType)
	}
	if retrievedTargetID != targetID {
		t.Errorf("expected target_id %d, got %d", targetID, retrievedTargetID)
	}
}

// TestLogAgentActivity_NilTargetID tests logging with nil target ID
func TestLogAgentActivity_NilTargetID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Create a test agent (user with is_agent = TRUE)
	var agentID int
	err := s.DB.QueryRow(`
		INSERT INTO users (username, email, password, is_agent, api_key_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, "Test Agent 2", "agent2@test.com", "", true, "test-hash-2").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Log activity with nil target ID and nil details
	LogAgentActivity(s.DB, agentID, "sync_started", "system", nil, nil)

	// Wait for async goroutine to complete (increased timeout for reliability)
	time.Sleep(200 * time.Millisecond)

	// Verify the record was inserted with NULL target_id and NULL details
	var count int
	var targetIDValue interface{}
	var detailsValue interface{}
	err = s.DB.QueryRow(`
		SELECT COUNT(*), target_id, details 
		FROM agent_activity_log 
		WHERE agent_id = $1
		GROUP BY target_id, details
	`, agentID).Scan(&count, &targetIDValue, &detailsValue)
	if err != nil {
		t.Fatalf("failed to query activity logs: %v", err)
	}

	// Assert we got exactly one record
	if count != 1 {
		t.Errorf("expected 1 activity log, got %d", count)
	}

	// Assert target_id is NULL
	if targetIDValue != nil {
		t.Errorf("expected target_id to be NULL, got %v", targetIDValue)
	}

	// Assert details is NULL
	if detailsValue != nil {
		t.Errorf("expected details to be NULL, got %v", detailsValue)
	}
}

// TestGetAgentActivity returns paginated results
func TestGetAgentActivity(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Create a test agent (user with is_agent = TRUE)
	var agentID int
	err := s.DB.QueryRow(`
		INSERT INTO users (username, email, password, is_agent, api_key_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, "Test Agent 3", "agent3@test.com", "", true, "test-hash-3").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Insert multiple activity logs directly
	for i := 1; i <= 15; i++ {
		targetID := i
		_, err := s.DB.Exec(`
			INSERT INTO agent_activity_log (agent_id, action, target_type, target_id, details, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 second' * $6)
		`, agentID, "update_card", "card", &targetID, `{"iteration": `+strconv.Itoa(i)+`}`, i)
		if err != nil {
			t.Fatalf("failed to insert activity log %d: %v", i, err)
		}
	}

	// Test first page
	logs, total, err := GetAgentActivity(s.DB, agentID, 1, 10)
	if err != nil {
		t.Fatalf("GetAgentActivity failed: %v", err)
	}

	if total != 15 {
		t.Errorf("expected total 15, got %d", total)
	}

	if len(logs) != 10 {
		t.Errorf("expected 10 logs on first page, got %d", len(logs))
	}

	// Verify ordering (newest first)
	if logs[0].Action != "update_card" {
		t.Errorf("expected action 'update_card', got '%s'", logs[0].Action)
	}

	// Test second page
	logs2, _, err := GetAgentActivity(s.DB, agentID, 2, 10)
	if err != nil {
		t.Fatalf("GetAgentActivity page 2 failed: %v", err)
	}

	if len(logs2) != 5 {
		t.Errorf("expected 5 logs on second page, got %d", len(logs2))
	}

	// Test third page (empty)
	logs3, _, err := GetAgentActivity(s.DB, agentID, 3, 10)
	if err != nil {
		t.Fatalf("GetAgentActivity page 3 failed: %v", err)
	}

	if len(logs3) != 0 {
		t.Errorf("expected 0 logs on third page, got %d", len(logs3))
	}
}

// TestGetAgentActivity_EmptyResults handles empty results
func TestGetAgentActivity_EmptyResults(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Create a test agent with no activity (user with is_agent = TRUE)
	var agentID int
	err := s.DB.QueryRow(`
		INSERT INTO users (username, email, password, is_agent, api_key_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, "Test Agent 4", "agent4@test.com", "", true, "test-hash-4").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Query for activity logs
	logs, total, err := GetAgentActivity(s.DB, agentID, 1, 10)
	if err != nil {
		t.Fatalf("GetAgentActivity failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total 0 for agent with no activity, got %d", total)
	}

	if logs == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

// TestGetAgentActivity_NonExistentAgent tests querying for a non-existent agent
func TestGetAgentActivity_NonExistentAgent(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Query for activity logs for a non-existent agent
	logs, total, err := GetAgentActivity(s.DB, 99999, 1, 10)
	if err != nil {
		t.Fatalf("GetAgentActivity failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total 0 for non-existent agent, got %d", total)
	}

	if len(logs) != 0 {
		t.Errorf("expected 0 logs for non-existent agent, got %d", len(logs))
	}
}

// TestLogAgentActivity_Concurrent tests concurrent logging
func TestLogAgentActivity_Concurrent(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Create a test agent (user with is_agent = TRUE)
	var agentID int
	err := s.DB.QueryRow(`
		INSERT INTO users (username, email, password, is_agent, api_key_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, "Test Agent 5", "agent5@test.com", "", true, "test-hash-5").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Log multiple activities concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			targetID := iteration
			LogAgentActivity(s.DB, agentID, "concurrent_action", "test", &targetID, map[string]interface{}{"iteration": iteration})
		}(i)
	}

	wg.Wait()

	// Wait for all async goroutines to complete
	time.Sleep(200 * time.Millisecond)

	// Verify all records were inserted
	var count int
	err = s.DB.QueryRow(`
		SELECT COUNT(*) FROM agent_activity_log WHERE agent_id = $1
	`, agentID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count activity logs: %v", err)
	}

	if count != 10 {
		t.Errorf("expected 10 activity logs, got %d", count)
	}
}
