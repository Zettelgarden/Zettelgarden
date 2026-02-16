package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Helper function to create a test notification
func createTestNotification(s *Handler, t *testing.T, userID int, sourceType string, sourceID int, title string, isRead bool) *models.Notification {
	preview := "Test preview content"
	timestamp := time.Now()
	importanceScore := 5
	filterTags := models.GetFilterTagsForEmail("unprocessed", "test@example.com")

	notification, err := models.CreateNotification(s.Server.Tx, userID, sourceType, sourceID, title, &preview, timestamp, importanceScore, filterTags)
	if err != nil {
		t.Fatalf("failed to create test notification: %v", err)
	}

	if isRead {
		err = models.MarkNotificationAsRead(s.Server.Tx, notification.ID, userID)
		if err != nil {
			t.Fatalf("failed to mark notification as read: %v", err)
		}
	}

	return notification
}

// Helper function to make a request to list notifications
func makeListNotificationsRequest(s *Handler, t *testing.T, userID int, params string) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	req, err := http.NewRequest("GET", "/api/notifications?"+params, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ListNotifications))
	handler.ServeHTTP(rr, req)

	return rr
}

// Helper function to make a request to get unread count
func makeUnreadCountRequest(s *Handler, t *testing.T, userID int) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	req, err := http.NewRequest("GET", "/api/notifications/unread-count", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetUnreadCount))
	handler.ServeHTTP(rr, req)

	return rr
}

// Helper function to make a request to mark notification as read
func makeMarkAsReadRequest(s *Handler, t *testing.T, userID, notificationID int) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	req, err := http.NewRequest("PATCH", "/api/notifications/"+strconv.Itoa(notificationID)+"/read", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/notifications/{id}/read", s.JwtMiddleware(s.MarkAsRead))
	router.ServeHTTP(rr, req)

	return rr
}

// Helper function to make a request to archive notification
func makeArchiveNotificationRequest(s *Handler, t *testing.T, userID, notificationID int) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	req, err := http.NewRequest("PATCH", "/api/notifications/"+strconv.Itoa(notificationID)+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/notifications/{id}/archive", s.JwtMiddleware(s.ArchiveNotification))
	router.ServeHTTP(rr, req)

	return rr
}

// Helper function to make a request to get notification preferences
func makeGetPreferencesRequest(s *Handler, t *testing.T, userID int) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	req, err := http.NewRequest("GET", "/api/notifications/preferences", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetPreferences))
	handler.ServeHTTP(rr, req)

	return rr
}

// Helper function to make a request to update notification preferences
func makeUpdatePreferencesRequest(s *Handler, t *testing.T, userID int, prefs models.NotificationPreferences) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(userID)

	jsonData, err := json.Marshal(prefs)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/notifications/preferences", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UpdatePreferences))
	handler.ServeHTTP(rr, req)

	return rr
}

func TestListNotificationsSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create test notifications
	createTestNotification(s, t, userID, models.SourceTypeEmail, 1, "Test Email 1", false)
	createTestNotification(s, t, userID, models.SourceTypeRSS, 1, "Test Article 1", false)
	createTestNotification(s, t, userID, models.SourceTypeTask, 1, "Test Task 1", true)

	rr := makeListNotificationsRequest(s, t, userID, "")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if notifications, ok := response["notifications"].([]interface{}); ok {
		if len(notifications) < 3 {
			t.Errorf("expected at least 3 notifications, got %d", len(notifications))
		}
	} else {
		t.Error("response should contain notifications array")
	}
}

func TestListNotificationsWithFilters(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create test notifications
	createTestNotification(s, t, userID, models.SourceTypeEmail, 1, "Test Email 1", false)
	createTestNotification(s, t, userID, models.SourceTypeRSS, 1, "Test Article 1", false)
	createTestNotification(s, t, userID, models.SourceTypeTask, 1, "Test Task 1", true)

	// Test filtering by source_type
	rr := makeListNotificationsRequest(s, t, userID, "source_type=email")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if notifications, ok := response["notifications"].([]interface{}); ok {
		if len(notifications) != 1 {
			t.Errorf("expected 1 notification for email source_type, got %d", len(notifications))
		}
	}

	// Test filtering by unread_only
	rr = makeListNotificationsRequest(s, t, userID, "unreadOnly=true")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if notifications, ok := response["notifications"].([]interface{}); ok {
		if len(notifications) != 2 {
			t.Errorf("expected 2 unread notifications, got %d", len(notifications))
		}
	}
}

func TestGetUnreadCountSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create test notifications - 2 unread, 1 read
	createTestNotification(s, t, userID, models.SourceTypeEmail, 1, "Test Email 1", false)
	createTestNotification(s, t, userID, models.SourceTypeRSS, 1, "Test Article 1", false)
	createTestNotification(s, t, userID, models.SourceTypeTask, 1, "Test Task 1", true)

	rr := makeUnreadCountRequest(s, t, userID)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]int
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if count, ok := response["count"]; !ok || count != 2 {
		t.Errorf("expected unread count 2, got %v", response["count"])
	}
}

func TestMarkAsReadSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create a test notification
	notification := createTestNotification(s, t, userID, models.SourceTypeEmail, 1, "Test Email 1", false)

	// Mark it as read
	rr := makeMarkAsReadRequest(s, t, userID, notification.ID)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify it's marked as read
	var notificationResp models.Notification
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &notificationResp)

	if !notificationResp.IsRead {
		t.Error("notification should be marked as read")
	}
}

func TestMarkAsReadNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Try to mark a non-existent notification as read
	rr := makeMarkAsReadRequest(s, t, userID, 99999)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestArchiveNotificationSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create a test notification
	notification := createTestNotification(s, t, userID, models.SourceTypeEmail, 1, "Test Email 1", false)

	// Archive it
	rr := makeArchiveNotificationRequest(s, t, userID, notification.ID)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify it's archived
	var notificationResp models.Notification
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &notificationResp)

	if !notificationResp.IsArchived {
		t.Error("notification should be archived")
	}
}

func TestArchiveNotificationNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Try to archive a non-existent notification
	rr := makeArchiveNotificationRequest(s, t, userID, 99999)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestGetPreferencesSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	rr := makeGetPreferencesRequest(s, t, userID)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var prefs models.NotificationPreferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &prefs)

	if prefs.UserID != userID {
		t.Errorf("expected user_id %d, got %d", userID, prefs.UserID)
	}
}

func TestUpdatePreferencesSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Get current preferences
	rr := makeGetPreferencesRequest(s, t, userID)
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to get preferences: %v", status)
	}

	var currentPrefs models.NotificationPreferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &currentPrefs)

	// Update preferences
	updatedPrefs := currentPrefs
	updatedPrefs.ItemsPerPage = 50

	rr = makeUpdatePreferencesRequest(s, t, userID, updatedPrefs)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var responsePrefs models.NotificationPreferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &responsePrefs)

	if responsePrefs.ItemsPerPage != 50 {
		t.Errorf("expected items_per_page 50, got %d", responsePrefs.ItemsPerPage)
	}
}

func TestNotificationsUserIsolation(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID1 := 1
	userID2 := 2

	// Create notifications for user 1
	createTestNotification(s, t, userID1, models.SourceTypeEmail, 1, "User 1 Email", false)

	// Create notifications for user 2
	createTestNotification(s, t, userID2, models.SourceTypeEmail, 2, "User 2 Email", false)

	// User 1 should only see their own notifications
	rr := makeListNotificationsRequest(s, t, userID1, "")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if notifications, ok := response["notifications"].([]interface{}); ok {
		for _, n := range notifications {
			notifMap := n.(map[string]interface{})
			if userID := int(notifMap["user_id"].(float64)); userID != userID1 {
				t.Errorf("user 1 should not see notifications for user %d", userID)
			}
		}
	}
}
