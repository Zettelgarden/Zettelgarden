package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterSearchRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/search", h.SearchRoute, "POST")
	addProtectedRoute(r, h, "/api/searches/star", h.StarSearchRoute, "POST")
	addProtectedRoute(r, h, "/api/searches/star/{id}", h.UnstarSearchRoute, "DELETE")
	addProtectedRoute(r, h, "/api/searches/starred", h.GetStarredSearchesRoute, "GET")
}