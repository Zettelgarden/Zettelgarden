package admin

import (
	"go-backend/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterAllAdminRoutes registers all admin routes under /api/admin prefix.
// This centralizes admin route registration for easier discovery and management.
//
// Routes are organized by feature:
// - User management: /api/admin/users/*
// - Mailing list: /api/admin/mailing-list/*
// - Audit logs: /api/admin/audit-logs
// - Statistics: /api/admin/stats
func RegisterAllAdminRoutes(r *mux.Router, h *handlers.Handler) {
	// Admin statistics and overview
	adminAPI := r.PathPrefix("/api/admin").Subrouter()

	// User management routes
	// Note: Some routes like /api/users are registered at the root level
	// but protected by admin middleware. This centralizes the
	// explicitly admin-prefixed routes.

	// Mailing list management (admin-only)
	// These are moved under /api/admin for clarity
	adminAPI.HandleFunc("/mailing-list/subscribers", h.GetMailingListSubscribersRoute).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/messages", h.GetMailingListMessagesRoute).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/messages/send", h.SendMailingListMessageRoute).Methods("POST")
	adminAPI.HandleFunc("/mailing-list/messages/recipients", h.GetMessageRecipientsRoute).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/unsubscribe", h.UnsubscribeMailingListRoute).Methods("POST")

	// Audit logs (admin-only)
	// TODO: Add GetAdminAuditLogsRoute handler
	// adminAPI.HandleFunc("/audit-logs", h.GetAdminAuditLogsRoute).Methods("GET")

	// Statistics (admin-only)
	adminAPI.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		GetAdminStatsRoute(h, w, r)
	}).Methods("GET")
}
