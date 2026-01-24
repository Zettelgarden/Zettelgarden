package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

// RegisterSchemaRoutes registers all schema definition management routes
func RegisterSchemaRoutes(r *mux.Router, h *handlers.Handler) {
	// Schema CRUD endpoints
	addProtectedRoute(r, h, "/api/schemas", h.CreateSchemaRoute, "POST")
	addProtectedRoute(r, h, "/api/schemas", h.GetSchemasRoute, "GET")
	addProtectedRoute(r, h, "/api/schemas/{id}", h.GetSchemaRoute, "GET")
	addProtectedRoute(r, h, "/api/schemas/{id}", h.UpdateSchemaRoute, "PUT")
	addProtectedRoute(r, h, "/api/schemas/{id}", h.DeleteSchemaRoute, "DELETE")
	addProtectedRoute(r, h, "/api/schemas/{id}/cards", h.GetCardsBySchemaRoute, "GET")
}
