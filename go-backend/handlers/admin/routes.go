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
// - Settings: /api/admin/settings (file-backed config.yaml)
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
	// All routes require authentication and admin authorization
	adminAPI.HandleFunc("/mailing-list/subscribers",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(h.GetMailingListSubscribersRoute))))).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/messages",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(h.GetMailingListMessagesRoute))))).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/messages/send",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(h.SendMailingListMessageRoute))))).Methods("POST")
	adminAPI.HandleFunc("/mailing-list/messages/recipients",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(h.GetMessageRecipientsRoute))))).Methods("GET")
	adminAPI.HandleFunc("/mailing-list/unsubscribe",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(h.UnsubscribeMailingListRoute))))).Methods("POST")

	// Job audit log (admin-only)
	// Jobs are now executed inline (see services.JobRunner); there is no
	// queue to pause/resume or worker pool to inspect. These endpoints
	// surface the audit log and allow re-running failed jobs.
	adminAPI.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		GetAllJobsRoute(h, w, r)
	}).Methods("GET")
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

	// Admin settings (file-backed config.yaml; admin-only)
	// Read all settings (incl. admin_email, which the public endpoint hides)
	// and apply partial updates that hot-reload without a restart.
	adminAPI.HandleFunc("/settings",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(func(w http.ResponseWriter, r *http.Request) {
						GetAdminSettingsRoute(h, w, r)
					}))))).Methods("GET")
	adminAPI.HandleFunc("/settings",
		h.APIKeyOrJWTMiddleware(
			h.AdminMiddleware(
				h.UpdateLastSeenMiddleware(
					handlers.LogRoute(func(w http.ResponseWriter, r *http.Request) {
						UpdateAdminSettingsRoute(h, w, r)
					}))))).Methods("PUT")

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
