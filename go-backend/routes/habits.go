package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterHabitRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/habits", h.GetHabitsRoute, "GET")
	addProtectedRoute(r, h, "/api/habits", h.CreateHabitRoute, "POST")
	addProtectedRoute(r, h, "/api/habits/today", h.GetTodaysHabitsRoute, "GET")
	addProtectedRoute(r, h, "/api/habits/{id}", h.GetHabitRoute, "GET")
	addProtectedRoute(r, h, "/api/habits/{id}", h.DeleteHabitRoute, "DELETE")
	addProtectedRoute(r, h, "/api/habits/{id}/checkin", h.CheckinHabitRoute, "POST")
	addProtectedRoute(r, h, "/api/habits/{id}/stats", h.GetHabitStatsRoute, "GET")
}
