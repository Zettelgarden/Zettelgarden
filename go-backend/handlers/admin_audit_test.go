package handlers

import (
	"context"
	"encoding/json"
	"go-backend/tests"
	"net/http"
	"testing"
	"time"
)

// TestLogAdminAction_Success verifies that admin actions are logged correctly
func TestLogAdminAction_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Create a request with user context
	req, err := http.NewRequest("POST", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Real-IP", "192.168.1.100")
	req.Header.Set("User-Agent", "test-agent/1.0")

	// Add user to context
	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	// Log admin action
	details := map[string]interface{}{
		"action":   "test_action",
		"test_key": "test_value",
		"before":   map[string]string{"status": "active"},
		"after":    map[string]string{"status": "inactive"},
	}
	s.LogAdminAction(req, "user.update", "user", 2, details)

	// Verify the audit log was created
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM admin_audit_log
		WHERE admin_user_id = 1
		AND action = 'user.update'
		AND target_type = 'user'
		AND target_id = 2
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", count)
	}

	// Verify the details were stored correctly
	var detailsJSON string
	err = s.Server.Tx.QueryRow(`
		SELECT CAST(details AS text) FROM admin_audit_log
		WHERE admin_user_id = 1
		AND action = 'user.update'
		LIMIT 1
	`).Scan(&detailsJSON)
	if err != nil {
		t.Fatalf("Failed to query audit log details: %v", err)
	}

	// Check that key fields exist in details
	if detailsJSON == "{}" {
		t.Errorf("Expected non-empty details, got empty object")
	}

	// Verify IP address and user agent were captured
	var ipAddress, userAgent string
	err = s.Server.Tx.QueryRow(`
		SELECT ip_address, user_agent FROM admin_audit_log
		WHERE admin_user_id = 1
		AND action = 'user.update'
		LIMIT 1
	`).Scan(&ipAddress, &userAgent)
	if err != nil {
		t.Fatalf("Failed to query audit log metadata: %v", err)
	}

	if ipAddress != "192.168.1.100" {
		t.Errorf("Expected IP address 192.168.1.100, got %s", ipAddress)
	}

	if userAgent != "test-agent/1.0" {
		t.Errorf("Expected user agent 'test-agent/1.0', got %s", userAgent)
	}
}

// TestLogAdminAction_XForwardedFor verifies X-Forwarded-For header is used for IP
func TestLogAdminAction_XForwardedFor(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-For", "10.0.0.50")
	req.Header.Set("User-Agent", "test-agent")

	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	s.LogAdminAction(req, "test.action", "test", 1, map[string]interface{}{})

	var ipAddress string
	err = s.Server.Tx.QueryRow(`
		SELECT ip_address FROM admin_audit_log
		WHERE action = 'test.action'
		LIMIT 1
	`).Scan(&ipAddress)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if ipAddress != "10.0.0.50" {
		t.Errorf("Expected IP address from X-Forwarded-For, got %s", ipAddress)
	}
}

// TestLogAdminActionAsync verifies async logging works
func TestLogAdminActionAsync(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("User-Agent", "test-agent")

	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	s.LogAdminActionAsync(req, "async.action", "test", 1, map[string]interface{}{
		"async": true,
	})

	// Wait a bit for the goroutine to complete
	// In production, you might use a wait group or channel
	var count int
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		err = s.Server.Tx.QueryRow(`
			SELECT COUNT(*) FROM admin_audit_log
			WHERE action = 'async.action'
		`).Scan(&count)
		if err == nil && count == 1 {
			break
		}
	}

	if count != 1 {
		t.Errorf("Expected async audit log to be created, got count %d", count)
	}
}

// TestGetAdminAuditLogs_Success verifies retrieving audit logs
func TestGetAdminAuditLogs_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Clean up any existing test data first
	_, err = s.Server.Tx.Exec(`DELETE FROM admin_audit_log WHERE action IN ('user.update', 'mailing_list.send', 'user.delete')`)
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	// Create some test audit logs
	_, err = s.Server.Tx.Exec(`
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, details, ip_address, user_agent)
		VALUES (1, 'user.update', 'user', 2, '{"test": "data"}', '127.0.0.1', 'test-agent'),
		       (1, 'mailing_list.send', 'mailing_list', 1, '{"recipients": 100}', '127.0.0.1', 'test-agent'),
		       (1, 'user.delete', 'user', 3, '{"reason": "spam"}', '127.0.0.1', 'test-agent')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test audit logs: %v", err)
	}

	// Get all logs
	logs, err := s.GetAdminAuditLogs(10, 0, "", "")
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 audit logs, got %d", len(logs))
	}

	// Verify logs are in descending order by created_at
	if logs[0].Action != "user.delete" {
		t.Errorf("Expected first log to be 'user.delete', got %s", logs[0].Action)
	}
}

// TestGetAdminAuditLogs_WithFilters verifies filtering works correctly
func TestGetAdminAuditLogs_WithFilters(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Create test audit logs with different actions
	_, err = s.Server.Tx.Exec(`
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, details, ip_address, user_agent)
		VALUES (1, 'user.update', 'user', 2, '{}', '127.0.0.1', 'test-agent'),
		       (1, 'user.delete', 'user', 3, '{}', '127.0.0.1', 'test-agent'),
		       (1, 'mailing_list.send', 'mailing_list', 1, '{}', '127.0.0.1', 'test-agent')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test audit logs: %v", err)
	}

	// Filter by action
	logs, err := s.GetAdminAuditLogs(10, 0, "user.update", "")
	if err != nil {
		t.Fatalf("Failed to get filtered audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 audit log for action 'user.update', got %d", len(logs))
	}

	if logs[0].Action != "user.update" {
		t.Errorf("Expected action 'user.update', got %s", logs[0].Action)
	}

	// Filter by target type
	logs, err = s.GetAdminAuditLogs(10, 0, "", "user")
	if err != nil {
		t.Fatalf("Failed to get audit logs filtered by target_type: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 audit logs for target_type 'user', got %d", len(logs))
	}

	// Filter by both
	logs, err = s.GetAdminAuditLogs(10, 0, "user.delete", "user")
	if err != nil {
		t.Fatalf("Failed to get audit logs filtered by both: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 audit log for action='user.delete' and target_type='user', got %d", len(logs))
	}
}

// TestGetAdminAuditLogs_Pagination verifies pagination works correctly
func TestGetAdminAuditLogs_Pagination(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Create 5 test audit logs
	for i := 1; i <= 5; i++ {
		_, err = s.Server.Tx.Exec(`
			INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, details, ip_address, user_agent)
			VALUES (1, $1, 'test', $2, '{}', '127.0.0.1', 'test-agent')
		`, "action_"+string(rune('0'+i)), i)
		if err != nil {
			t.Fatalf("Failed to insert test audit log: %v", err)
		}
	}

	// Get first page
	logs, err := s.GetAdminAuditLogs(2, 0, "", "")
	if err != nil {
		t.Fatalf("Failed to get first page: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 logs on first page, got %d", len(logs))
	}

	// Get second page
	logs, err = s.GetAdminAuditLogs(2, 2, "", "")
	if err != nil {
		t.Fatalf("Failed to get second page: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 logs on second page, got %d", len(logs))
	}

	// Get third page (should have 1 log)
	logs, err = s.GetAdminAuditLogs(2, 4, "", "")
	if err != nil {
		t.Fatalf("Failed to get third page: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log on third page, got %d", len(logs))
	}
}

// TestLogAdminAction_NoContext verifies handling when current_user is not in context
func TestLogAdminAction_NoContext(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	req, err := http.NewRequest("POST", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	// No current_user in context

	// This should not panic, just log an error
	s.LogAdminAction(req, "test.action", "test", 1, map[string]interface{}{})

	// Verify no audit log was created (should fail gracefully)
	var count int
	err = s.Server.Tx.QueryRow(`
		SELECT COUNT(*) FROM admin_audit_log
		WHERE action = 'test.action'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 audit log entries when no user in context, got %d", count)
	}
}

// TestAdminAuditLogStructure verifies the audit log structure matches expectations
func TestAdminAuditLogStructure(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1:1234"

	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)

	details := map[string]interface{}{
		"string": "value",
		"number": 123,
		"nested": map[string]string{"key": "value"},
		"array":  []int{1, 2, 3},
	}

	s.LogAdminAction(req, "structure.test", "test", 1, details)

	var log AdminAuditLog
	var detailsJSON []byte
	err = s.Server.Tx.QueryRow(`
		SELECT id, admin_user_id, action, target_type, target_id, details, ip_address, user_agent, created_at
		FROM admin_audit_log
		WHERE action = 'structure.test'
		LIMIT 1
	`).Scan(
		&log.ID,
		&log.AdminUserID,
		&log.Action,
		&log.TargetType,
		&log.TargetID,
		&detailsJSON,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	// Unmarshal details JSON
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			t.Fatalf("Failed to unmarshal details: %v", err)
		}
	}

	// Verify all fields are populated
	if log.AdminUserID != 1 {
		t.Errorf("Expected admin_user_id 1, got %d", log.AdminUserID)
	}

	if log.Action != "structure.test" {
		t.Errorf("Expected action 'structure.test', got %s", log.Action)
	}

	if log.TargetType != "test" {
		t.Errorf("Expected target_type 'test', got %s", log.TargetType)
	}

	if !log.TargetID.Valid || log.TargetID.Int32 != 1 {
		t.Errorf("Expected target_id 1, got %v", log.TargetID)
	}

	// Verify details were unmarshaled correctly
	if log.Details == nil {
		t.Error("Expected details to be unmarshaled, got nil")
	} else {
		if log.Details["string"] != "value" {
			t.Errorf("Expected details['string'] = 'value', got %v", log.Details["string"])
		}
	}
}
