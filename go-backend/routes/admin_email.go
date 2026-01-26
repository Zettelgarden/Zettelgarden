package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

// RegisterAdminEmailRoutes registers all admin email management routes
func RegisterAdminEmailRoutes(r *mux.Router, h *handlers.Handler) {
	// Get email queue statistics
	addAdminRoute(r, h, "/api/admin/email/stats", h.GetEmailQueueStatsRoute, "GET")

	// Get failed emails (dead-letter queue)
	addAdminRoute(r, h, "/api/admin/email/failed", h.GetFailedEmailsRoute, "GET")

	// Retry a failed email
	addAdminRoute(r, h, "/api/admin/email/failed/{id}/retry", h.RetryFailedEmailRoute, "POST")

	// Delete a failed email permanently
	addAdminRoute(r, h, "/api/admin/email/failed/{id}", h.DeleteFailedEmailRoute, "DELETE")
}
