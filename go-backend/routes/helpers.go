package routes

import (
	"go-backend/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

// Helper functions for route registration
//
// addProtectedRoute wraps the handler with authentication and logging middleware.
// Execution order: APIKeyOrJWTMiddleware -> LogRoute -> handler
//
// This order ensures:
// - APIKeyOrJWTMiddleware runs first to authenticate and set current_user in context
// - LogRoute can then access current_user for logging on protected routes
// - Finally the actual handler executes
func addProtectedRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string) *mux.Route {
	logged := handlers.LogRoute(handler)
	tracked := h.UpdateLastSeenMiddleware(logged)
	protected := h.APIKeyOrJWTMiddleware(tracked)
	return r.HandleFunc(path, protected).Methods(method)
}

// addRoute wraps the handler with only logging middleware.
// Unlike addProtectedRoute, no authentication is applied.
func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
	return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}

// addAdminRoute wraps the handler with authentication, admin check, and logging middleware.
// Execution order: AdminMiddleware -> APIKeyOrJWTMiddleware -> LogRoute -> UpdateLastSeenMiddleware -> handler
//
// Use this for routes that require admin privileges.
// The admin check verifies that the authenticated user has is_admin = true.
func addAdminRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string) *mux.Route {
	logged := handlers.LogRoute(handler)
	tracked := h.UpdateLastSeenMiddleware(logged)
	authenticated := h.APIKeyOrJWTMiddleware(tracked)
	adminOnly := h.AdminMiddleware(authenticated)
	return r.HandleFunc(path, adminOnly).Methods(method)
}

// addAdminOrSelfRoute wraps the handler with authentication and admin-or-self check.
// This allows admins to access any resource, or users to access their own resources.
//
// The idParam should match the URL variable name (e.g., "id" for /api/users/{id})
func addAdminOrSelfRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string, idParam string) *mux.Route {
	logged := handlers.LogRoute(handler)
	tracked := h.UpdateLastSeenMiddleware(logged)
	authenticated := h.APIKeyOrJWTMiddleware(tracked)
	adminOrSelf := h.AdminOrSelfMiddleware(idParam)(authenticated)
	return r.HandleFunc(path, adminOrSelf).Methods(method)
}