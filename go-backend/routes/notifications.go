package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterNotificationRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/notifications", h.ListNotifications, "GET")
	addProtectedRoute(r, h, "/api/notifications/unread-count", h.GetUnreadCount, "GET")
	addProtectedRoute(r, h, "/api/notifications/{id}/read", h.MarkAsRead, "PATCH")
	addProtectedRoute(r, h, "/api/notifications/{id}/archive", h.ArchiveNotification, "PATCH")
	addProtectedRoute(r, h, "/api/notifications/preferences", h.GetPreferences, "GET")
	addProtectedRoute(r, h, "/api/notifications/preferences", h.UpdatePreferences, "PATCH")
}
