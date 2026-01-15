package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterURLRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/url/parse", h.ParseURLRoute, "POST")
}