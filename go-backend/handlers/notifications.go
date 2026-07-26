package handlers

import (
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// ListNotifications handles GET /api/notifications
// Retrieves notifications for the authenticated user with optional filters
// Query parameters:
// - source_type: Filter by source type ("rss", "task")
// - unreadOnly: If "true", only return unread notifications
// - limit: Maximum number of results to return
// - offset: Number of results to skip for pagination
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Build filters from query parameters
	var filters models.NotificationListFilters

	if sourceType := r.URL.Query().Get("source_type"); sourceType != "" {
		filters.SourceType = &sourceType
	}

	if unreadOnlyStr := r.URL.Query().Get("unreadOnly"); unreadOnlyStr == "true" {
		isRead := false
		filters.IsRead = &isRead
	}

	// Default: exclude archived notifications
	isArchived := false
	filters.IsArchived = &isArchived

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = &limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = &offset
		}
	}

	// Get notifications from the database
	notifications, err := models.GetNotificationsByUser(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("[notifications] failed to get notifications: %v", err)
		http.Error(w, "Failed to get notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"notifications": notifications,
	}
	json.NewEncoder(w).Encode(response)
}

// GetUnreadCount handles GET /api/notifications/unread-count
// Returns the count of unread, unarchived notifications for the authenticated user
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	count, err := models.GetUnreadCount(h.GetDB(), userID)
	if err != nil {
		log.Printf("[notifications] failed to get unread count: %v", err)
		http.Error(w, "Failed to get unread count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]int{
		"count": count,
	}
	json.NewEncoder(w).Encode(response)
}

// MarkAsRead handles PATCH /api/notifications/{id}/read
// Marks a specific notification as read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get notification ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	notificationID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	// Verify notification exists and belongs to user
	notification, err := models.GetNotificationByID(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to get notification: %v", err)
		http.Error(w, "Failed to get notification", http.StatusInternalServerError)
		return
	}
	if notification == nil {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	// Mark as read
	err = models.MarkNotificationAsRead(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to mark notification as read: %v", err)
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	// Get updated notification to return
	notification, err = models.GetNotificationByID(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to get updated notification: %v", err)
		http.Error(w, "Failed to get notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notification)
}

// ArchiveNotification handles PATCH /api/notifications/{id}/archive
// Marks a specific notification as archived
func (h *Handler) ArchiveNotification(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get notification ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	notificationID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	// Verify notification exists and belongs to user
	notification, err := models.GetNotificationByID(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to get notification: %v", err)
		http.Error(w, "Failed to get notification", http.StatusInternalServerError)
		return
	}
	if notification == nil {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	// Mark as archived
	err = models.MarkNotificationAsArchived(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to archive notification: %v", err)
		http.Error(w, "Failed to archive notification", http.StatusInternalServerError)
		return
	}

	// Get updated notification to return
	notification, err = models.GetNotificationByID(h.GetDB(), notificationID, userID)
	if err != nil {
		log.Printf("[notifications] failed to get updated notification: %v", err)
		http.Error(w, "Failed to get notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notification)
}

// GetPreferences handles GET /api/notifications/preferences
// Retrieves notification preferences for the authenticated user
// Returns default preferences if none are set
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	prefs, err := models.GetNotificationPreferences(h.GetDB(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			// Return default preferences
			defaultPrefs := models.NotificationPreferences{
				UserID:              userID,
				ShowStarredArticles: true,
				ShowPriorityTasks:   true,
				ShowPriorityFeeds:   true,
				ItemsPerPage:        20,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(defaultPrefs)
			return
		}
		log.Printf("[notifications] failed to get preferences: %v", err)
		http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(prefs)
}

// UpdatePreferences handles PATCH /api/notifications/preferences
// Updates notification preferences for the authenticated user
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var prefs models.NotificationPreferences
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&prefs); err != nil {
		log.Printf("[notifications] failed to decode request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Ensure user_id matches authenticated user
	prefs.UserID = userID

	// Validate items_per_page
	if prefs.ItemsPerPage <= 0 || prefs.ItemsPerPage > 100 {
		http.Error(w, "items_per_page must be between 1 and 100", http.StatusBadRequest)
		return
	}

	err := models.UpdateNotificationPreferences(h.GetDB(), userID, prefs)
	if err != nil {
		log.Printf("[notifications] failed to update preferences: %v", err)
		http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	// Get updated preferences to return
	updatedPrefs, err := models.GetNotificationPreferences(h.GetDB(), userID)
	if err != nil {
		log.Printf("[notifications] failed to get updated preferences: %v", err)
		http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedPrefs)
}
