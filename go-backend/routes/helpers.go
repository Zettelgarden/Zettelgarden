package routes

import (
	"go-backend/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

// Helper functions for route registration
//
// addProtectedRoute wraps the handler with authentication and logging middleware.
// Execution order: APIKeyOrJWTMiddleware -> UpdateLastSeenMiddleware -> LogRoute -> handler
//
// This order ensures:
// - APIKeyOrJWTMiddleware runs first to authenticate and set current_user in context
// - UpdateLastSeenMiddleware updates the user's last_seen timestamp
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
// Execution order: APIKeyOrJWTMiddleware -> AdminMiddleware -> UpdateLastSeenMiddleware -> LogRoute -> handler
//
// Use this for routes that require admin privileges.
// The admin check verifies that the authenticated user has is_admin = true.
func addAdminRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string) *mux.Route {
	logged := handlers.LogRoute(handler)
	tracked := h.UpdateLastSeenMiddleware(logged)
	adminOnly := h.AdminMiddleware(tracked)
	authenticated := h.APIKeyOrJWTMiddleware(adminOnly)
	return r.HandleFunc(path, authenticated).Methods(method)
}

// addAdminOrSelfRoute wraps the handler with authentication and admin-or-self check.
// This allows admins to access any resource, or users to access their own resources.
// Execution order: APIKeyOrJWTMiddleware -> AdminOrSelfMiddleware -> UpdateLastSeenMiddleware -> LogRoute -> handler
//
// The idParam should match the URL variable name (e.g., "id" for /api/users/{id})
func addAdminOrSelfRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string, idParam string) *mux.Route {
	logged := handlers.LogRoute(handler)
	tracked := h.UpdateLastSeenMiddleware(logged)
	adminOrSelf := h.AdminOrSelfMiddleware(idParam)(tracked)
	authenticated := h.APIKeyOrJWTMiddleware(adminOrSelf)
	return r.HandleFunc(path, authenticated).Methods(method)
}