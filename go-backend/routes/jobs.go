package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

// RegisterJobRoutes registers all job-related routes
func RegisterJobRoutes(r *mux.Router, h *handlers.Handler) {
	// Create a new job
	addProtectedRoute(r, h, "/api/jobs", h.CreateJobRoute, "POST")

	// Get a specific job
	addProtectedRoute(r, h, "/api/jobs/{id}", h.GetJobRoute, "GET")

	// List jobs for current user
	addProtectedRoute(r, h, "/api/jobs", h.ListJobsRoute, "GET")

	// Cancel a pending job
	addProtectedRoute(r, h, "/api/jobs/{id}/cancel", h.CancelJobRoute, "DELETE")

	// Get job statistics for current user
	addProtectedRoute(r, h, "/api/jobs/stats", h.GetJobStatsRoute, "GET")
}
