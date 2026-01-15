package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterStatsRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/stats/daily", h.GetDailyStatsRoute, "GET")
	addProtectedRoute(r, h, "/api/stats/day-tasks", h.GetDayTasksRoute, "GET")
	addProtectedRoute(r, h, "/api/stats/day-cards", h.GetDayCardsRoute, "GET")
}