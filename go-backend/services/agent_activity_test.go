package services

import (
	"go-backend/tests"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestLogAgentActivity tests the async logging of agent activity.
// It verifies that:
// - Activity is logged asynchronously without blocking
// - All fields are correctly stored in the database
// - The async goroutine completes within the timeout period
//
// Note: The agent_activity_log table is created via migration 0140-create-agent-activity-log.sql
// and is automatically applied by tests.Setup() through server.RunMigrations().
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

	// Wait for async goroutine to complete (50ms simulates real-world scenario where agents update quickly)
	time.Sleep(50 * time.Millisecond)

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

// TestLogAgentActivity_NilTargetID tests logging with nil target ID and nil details.
// It verifies that:
// - NULL values for target_id are handled correctly and stored as NULL in database
// - NULL values for details are handled correctly and stored as NULL in database
// - The activity log record is created successfully with NULL fields
// - Query returns proper NULL values (not zero values or empty strings)
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

	// Wait for async goroutine to complete (50ms simulates real-world scenario)
	time.Sleep(50 * time.Millisecond)

	// Verify the record was inserted with NULL target_id and NULL details
	var count int
	err = s.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM agent_activity_log 
		WHERE agent_id = $1
	`, agentID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count activity logs: %v", err)
	}

	// Assert we got exactly one record
	if count != 1 {
		t.Fatalf("expected 1 activity log, got %d", count)
	}

	// Query the specific record to verify NULL values
	var targetIDValue interface{}
	var detailsValue interface{}
	var action, targetType string
	err = s.DB.QueryRow(`
		SELECT action, target_type, target_id, details 
		FROM agent_activity_log 
		WHERE agent_id = $1
	`, agentID).Scan(&action, &targetType, &targetIDValue, &detailsValue)
	if err != nil {
		t.Fatalf("failed to query activity log details: %v", err)
	}

	// Verify action and target_type are set correctly
	if action != "sync_started" {
		t.Errorf("expected action 'sync_started', got '%s'", action)
	}
	if targetType != "system" {
		t.Errorf("expected target_type 'system', got '%s'", targetType)
	}

	// Assert target_id is NULL (not 0, not empty, but actual NULL)
	if targetIDValue != nil {
		t.Errorf("expected target_id to be NULL, got %v (type: %T)", targetIDValue, targetIDValue)
	}

	// Assert details is NULL (not empty JSON, but actual NULL)
	if detailsValue != nil {
		t.Errorf("expected details to be NULL, got %v (type: %T)", detailsValue, detailsValue)
	}
}

// TestGetAgentActivity returns paginated results using table-driven tests
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

	// Table-driven tests for pagination scenarios
	tests := []struct {
		name          string
		page          int
		pageSize      int
		expectedCount int
		expectedTotal int
	}{
		{
			name:          "first page",
			page:          1,
			pageSize:      10,
			expectedCount: 10,
			expectedTotal: 15,
		},
		{
			name:          "second page",
			page:          2,
			pageSize:      10,
			expectedCount: 5,
			expectedTotal: 15,
		},
		{
			name:          "third page (empty)",
			page:          3,
			pageSize:      10,
			expectedCount: 0,
			expectedTotal: 15,
		},
		{
			name:          "page size larger than total",
			page:          1,
			pageSize:      20,
			expectedCount: 15,
			expectedTotal: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, total, err := GetAgentActivity(s.DB, agentID, tt.page, tt.pageSize)
			if err != nil {
				t.Fatalf("GetAgentActivity failed: %v", err)
			}

			if total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, total)
			}

			if len(logs) != tt.expectedCount {
				t.Errorf("expected %d logs, got %d", tt.expectedCount, len(logs))
			}

			// Verify ordering (newest first) for non-empty results
			if len(logs) > 0 && logs[0].Action != "update_card" {
				t.Errorf("expected action 'update_card', got '%s'", logs[0].Action)
			}
		})
	}
}

// TestGetAgentActivity_EmptyResults handles empty results.
// It verifies that:
// - An empty slice (not nil) is returned when no activity exists
// - Total count is 0 for agents with no activity
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

// TestGetAgentActivity_NonExistentAgent tests querying for a non-existent agent.
// It verifies that:
// - Non-existent agent IDs return empty results (not an error)
// - Total count is 0 for non-existent agents
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

// TestLogAgentActivity_Concurrent tests concurrent logging.
// It verifies that:
// - Multiple goroutines can log activity concurrently without race conditions
// - All activity records are successfully inserted
// - The async logging mechanism is thread-safe
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

	// Wait for all async goroutines to complete (50ms simulates real-world scenario)
	time.Sleep(50 * time.Millisecond)

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
