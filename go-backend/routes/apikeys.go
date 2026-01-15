package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterAPIKeyRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/api-keys", h.ListAPIKeys, "GET")
	addProtectedRoute(r, h, "/api/api-keys", h.CreateAPIKey, "POST")
	addProtectedRoute(r, h, "/api/api-keys/{id}", h.RevokeAPIKey, "DELETE")
}