package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterTagRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/tags", h.GetTagsRoute, "GET")
	addProtectedRoute(r, h, "/api/tags", h.CreateTagRoute, "POST")
	addProtectedRoute(r, h, "/api/tags/id/{id}", h.DeleteTagRoute, "DELETE")
}