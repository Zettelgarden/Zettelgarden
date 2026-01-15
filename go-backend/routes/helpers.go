package routes

import (
	"go-backend/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

// Helper functions for route registration
func addProtectedRoute(r *mux.Router, h *handlers.Handler, path string, handler http.HandlerFunc, method string) *mux.Route {
	return r.HandleFunc(path, h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))).Methods(method)
}

func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
	return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}