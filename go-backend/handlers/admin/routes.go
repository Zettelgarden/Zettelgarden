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
// - Job queue: /api/admin/jobs/*
// - Scheduler: /api/admin/scheduler/*
// - Audit logs: /api/admin/audit-logs
// - Statistics: /api/admin/stats
func RegisterAllAdminRoutes(r *mux.Router, h *handlers.Handler, scheduler handlers.SchedulerAPI) {
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

	// Job queue management (admin-only)
	adminAPI.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		GetAllJobsRoute(h, w, r)
	}).Methods("GET")
	adminAPI.HandleFunc("/jobs/health", func(w http.ResponseWriter, r *http.Request) {
		GetJobQueueHealthRoute(h, w, r)
	}).Methods("GET")
	adminAPI.HandleFunc("/jobs/workers", func(w http.ResponseWriter, r *http.Request) {
		GetJobWorkersStatsRoute(h, w, r)
	}).Methods("GET")
	adminAPI.HandleFunc("/jobs/pause", func(w http.ResponseWriter, r *http.Request) {
		PauseJobQueueRoute(h, w, r)
	}).Methods("POST")
	adminAPI.HandleFunc("/jobs/resume", func(w http.ResponseWriter, r *http.Request) {
		ResumeJobQueueRoute(h, w, r)
	}).Methods("POST")
	adminAPI.HandleFunc("/jobs/{id}/retry", func(w http.ResponseWriter, r *http.Request) {
		RetryJobRoute(h, w, r)
	}).Methods("POST")

	// Audit logs (admin-only)
	// TODO: Add GetAdminAuditLogsRoute handler
	// adminAPI.HandleFunc("/audit-logs", h.GetAdminAuditLogsRoute).Methods("GET")

	// Statistics (admin-only)
	adminAPI.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		GetAdminStatsRoute(h, w, r)
	}).Methods("GET")

	// Scheduler management routes (admin-only)
	// These routes require admin authentication - middleware chain follows same pattern as addAdminRoute:
	// APIKeyOrJWTMiddleware -> AdminMiddleware -> UpdateLastSeenMiddleware -> LogRoute -> handler
	if scheduler != nil {
		adminAPI.HandleFunc("/scheduler/jobs",
			h.APIKeyOrJWTMiddleware(
				h.AdminMiddleware(
					h.UpdateLastSeenMiddleware(
						handlers.LogRoute(handlers.ListScheduledJobs(scheduler)))))).Methods("GET")
		adminAPI.HandleFunc("/scheduler/jobs/{jobName}/summary",
			h.APIKeyOrJWTMiddleware(
				h.AdminMiddleware(
					h.UpdateLastSeenMiddleware(
						handlers.LogRoute(handlers.GetJobSummaryHandler(scheduler)))))).Methods("GET")
		adminAPI.HandleFunc("/scheduler/jobs/{jobName}/history",
			h.APIKeyOrJWTMiddleware(
				h.AdminMiddleware(
					h.UpdateLastSeenMiddleware(
						handlers.LogRoute(handlers.GetJobHistory(scheduler)))))).Methods("GET")
		adminAPI.HandleFunc("/scheduler/health",
			h.APIKeyOrJWTMiddleware(
				h.AdminMiddleware(
					h.UpdateLastSeenMiddleware(
						handlers.LogRoute(handlers.GetSchedulerHealth(scheduler)))))).Methods("GET")
	}
}
