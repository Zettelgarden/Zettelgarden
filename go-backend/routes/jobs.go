package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

// RegisterJobRoutes registers all job-related routes.
//
// Jobs are no longer created through the API: they are an audit record of LLM
// work executed inline by services.JobRunner. Only read-only audit views are
// exposed here (manual job creation and cancellation have been removed).
func RegisterJobRoutes(r *mux.Router, h *handlers.Handler) {
	// Get a specific job (audit record)
	addProtectedRoute(r, h, "/api/jobs/{id}", h.GetJobRoute, "GET")

	// List jobs for current user
	addProtectedRoute(r, h, "/api/jobs", h.ListJobsRoute, "GET")

	// Get job statistics for current user
	addProtectedRoute(r, h, "/api/jobs/stats", h.GetJobStatsRoute, "GET")
}
