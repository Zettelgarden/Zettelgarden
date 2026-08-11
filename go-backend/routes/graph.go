package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterGraphRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/graph", h.GetGraphRoute, "GET")
	addProtectedRoute(r, h, "/api/graph/stats", h.GetNetworkStatsRoute, "GET")
}
