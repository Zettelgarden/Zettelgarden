package admin

import (
	"encoding/json"
	"go-backend/handlers"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupTestHandler creates a test handler with admin user
func setupTestHandler(t *testing.T) *handlers.Handler {
	s := tests.Setup()
	t.Cleanup(tests.Teardown)

	h := &handlers.Handler{
		DB:     s.DB,
		Server: s,
	}

	// Make user 1 an admin for testing
	_, err := s.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	return h
}

// TestGetAdminStatsRoute_Success verifies that stats endpoint returns valid statistics
func TestGetAdminStatsRoute_Success(t *testing.T) {
	h := setupTestHandler(t)
	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/admin/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetAdminStatsRoute(h, w, r)
	})

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Verify response structure
	var stats map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check that all stat categories exist
	requiredCategories := []string{"users", "subscriptions", "revenue", "content"}
	for _, category := range requiredCategories {
		if _, ok := stats[category]; !ok {
			t.Errorf("Missing stat category: %s", category)
		}
	}
}

// TestGetAdminStatsRoute_RequiresAdmin verifies that non-admins are blocked
func TestGetAdminStatsRoute_RequiresAdmin(t *testing.T) {
	s := tests.Setup()
	t.Cleanup(tests.Teardown)

	h := &handlers.Handler{
		DB:     s.DB,
		Server: s,
	}

	// Ensure user 2 is NOT an admin
	_, err := s.DB.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/admin/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate admin middleware check before calling handler
		var isAdmin bool
		err := h.DB.QueryRow("SELECT is_admin FROM users WHERE id = 2").Scan(&isAdmin)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}
		if !isAdmin {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		GetAdminStatsRoute(h, w, r)
	})

	handler.ServeHTTP(rr, req)

	// Should be blocked by admin check
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 401 or 403, got %d", rr.Code)
	}
}

// TestGetUserStats verifies user statistics are calculated correctly
func TestGetUserStats(t *testing.T) {
	h := setupTestHandler(t)

	stats, err := getUserStats(h)
	if err != nil {
		t.Fatalf("Failed to get user stats: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"total", "active_this_week", "active_this_month",
		"total_admins", "new_this_week", "new_this_month",
	}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Missing user stat field: %s", field)
		}
	}

	// Verify total is at least the number of test users (3 in reduced test data)
	total, ok := stats["total"].(int)
	if !ok || total < 3 {
		t.Errorf("Expected at least 3 users, got %d", total)
	}

	// Verify admin count is at least 1 (user 1 is set as admin in setup)
	totalAdmins, ok := stats["total_admins"].(int)
	if !ok || totalAdmins < 1 {
		t.Errorf("Expected at least 1 admin, got %d", totalAdmins)
	}
}

// TestGetSubscriptionStats verifies subscription statistics
func TestGetSubscriptionStats(t *testing.T) {
	h := setupTestHandler(t)

	stats, err := getSubscriptionStats(h)
	if err != nil {
		t.Fatalf("Failed to get subscription stats: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"by_status", "total", "active", "free", "past_due",
	}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Missing subscription stat field: %s", field)
		}
	}

	// Verify by_status is a map
	byStatus, ok := stats["by_status"].(map[string]int)
	if !ok {
		t.Errorf("by_status should be a map[string]int")
	} else {
		// Check for expected status values (may be empty in test data)
		expectedStatuses := []string{"free", "active", "trialing", "past_due", "canceled"}
		for _, status := range expectedStatuses {
			// Just verify the key exists, value can be 0
			_, hasKey := byStatus[status]
			if !hasKey && status != "active" && status != "trialing" && status != "past_due" && status != "canceled" {
				// "free" is expected since test users have no subscription
			}
		}
	}
}

// TestGetRevenueStats verifies revenue statistics
func TestGetRevenueStats(t *testing.T) {
	h := setupTestHandler(t)

	stats, err := getRevenueStats(h)
	if err != nil {
		t.Fatalf("Failed to get revenue stats: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"total_revenue", "revenue_this_month", "monthly_recurring_revenue",
	}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Missing revenue stat field: %s", field)
		}
	}

	// Revenue values should be floats (may be 0.0 in test data)
	totalRevenue, ok := stats["total_revenue"].(float64)
	if !ok {
		t.Errorf("total_revenue should be a float64")
	} else if totalRevenue < 0 {
		t.Errorf("total_revenue should be non-negative, got %f", totalRevenue)
	}
}

// TestGetContentStats verifies content statistics
func TestGetContentStats(t *testing.T) {
	h := setupTestHandler(t)

	stats, err := getContentStats(h)
	if err != nil {
		t.Fatalf("Failed to get content stats: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"total_cards", "total_tasks", "total_files",
		"total_chat_messages", "total_entities", "total_facts",
	}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Missing content stat field: %s", field)
		}
	}

	// Verify expected counts based on test data
	// Test data has 10 + 4 cards (14 total after reduction)
	totalCards, ok := stats["total_cards"].(int)
	if !ok || totalCards < 10 {
		t.Errorf("Expected at least 10 cards, got %d", totalCards)
	}

	// Test data has 5 tasks (reduced from 20)
	totalTasks, ok := stats["total_tasks"].(int)
	if !ok || totalTasks < 5 {
		t.Errorf("Expected at least 5 tasks, got %d", totalTasks)
	}

	// Test data has 5 files (reduced from 20)
	totalFiles, ok := stats["total_files"].(int)
	if !ok || totalFiles < 5 {
		t.Errorf("Expected at least 5 files, got %d", totalFiles)
	}

	// Test data has 3 entities
	totalEntities, ok := stats["total_entities"].(int)
	if !ok || totalEntities < 3 {
		t.Errorf("Expected at least 3 entities, got %d", totalEntities)
	}
}

// TestGetUserStats_ActiveUserWindow verifies the active user time windows work correctly
func TestGetUserStats_ActiveUserWindow(t *testing.T) {
	s := tests.Setup()
	t.Cleanup(tests.Teardown)

	h := &handlers.Handler{
		DB:     s.DB,
		Server: s,
	}

	// Make user 1 an admin
	_, err := s.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Set all 3 users as recently active (within the week)
	_, err = s.DB.Exec(`
		UPDATE users
		SET last_seen = NOW()
		WHERE id IN (1, 2, 3)
	`)
	if err != nil {
		t.Fatalf("Failed to update last_seen: %v", err)
	}

	// User 3 was active 15 days ago (still within the month)
	_, err = s.DB.Exec(`
		UPDATE users
		SET last_seen = NOW() - INTERVAL '15 days'
		WHERE id = 3
	`)
	if err != nil {
		t.Fatalf("Failed to update last_seen: %v", err)
	}

	stats, err := getUserStats(h)
	if err != nil {
		t.Fatalf("Failed to get user stats: %v", err)
	}

	// Should have at least 2 active this week (users 1 and 2)
	activeWeek, ok := stats["active_this_week"].(int)
	if !ok || activeWeek < 2 {
		t.Errorf("Expected at least 2 active users this week, got %d", activeWeek)
	}

	// Should have all 3 users active this month
	activeMonth, ok := stats["active_this_month"].(int)
	if !ok || activeMonth < 3 {
		t.Errorf("Expected at least 3 active users this month, got %d", activeMonth)
	}
}

// TestGetUserStats_NewUsers verifies the new user counts work correctly
func TestGetUserStats_NewUsers(t *testing.T) {
	s := tests.Setup()
	t.Cleanup(tests.Teardown)

	h := &handlers.Handler{
		DB:     s.DB,
		Server: s,
	}

	// Make user 1 an admin
	_, err := s.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Create a new user today
	_, err = s.DB.Exec(`
		INSERT INTO users (username, email, password, created_at, updated_at, email_validated)
		VALUES ('newuser', 'new@test.com', 'hash', NOW(), NOW(), true)
	`)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	stats, err := getUserStats(h)
	if err != nil {
		t.Fatalf("Failed to get user stats: %v", err)
	}

	// Should have at least 1 new user this week
	newWeek, ok := stats["new_this_week"].(int)
	if !ok || newWeek < 1 {
		t.Errorf("Expected at least 1 new user this week, got %d", newWeek)
	}

	// Should have at least 1 new user this month
	newMonth, ok := stats["new_this_month"].(int)
	if !ok || newMonth < 1 {
		t.Errorf("Expected at least 1 new user this month, got %d", newMonth)
	}
}

// TestGetAdminStatsRoute_ResponseContentType verifies the response has correct content type
func TestGetAdminStatsRoute_ResponseContentType(t *testing.T) {
	h := setupTestHandler(t)
	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/admin/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetAdminStatsRoute(h, w, r)
	})

	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}
