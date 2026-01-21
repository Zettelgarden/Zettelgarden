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