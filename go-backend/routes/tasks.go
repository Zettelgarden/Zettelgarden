package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterTaskRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/tasks/{id}", h.GetTaskRoute, "GET")
	addProtectedRoute(r, h, "/api/tasks", h.GetTasksRoute, "GET")
	addProtectedRoute(r, h, "/api/tasks", h.CreateTaskRoute, "POST")
	addProtectedRoute(r, h, "/api/tasks/{id}", h.UpdateTaskRoute, "PUT")
	addProtectedRoute(r, h, "/api/tasks/{id}", h.DeleteTaskRoute, "DELETE")
	addProtectedRoute(r, h, "/api/tasks/{id}/audit", h.GetTaskAuditEventsRoute, "GET")
	addProtectedRoute(r, h, "/api/tasks/{id}/dependencies", h.AddTaskDependencyRoute, "POST")
	addProtectedRoute(r, h, "/api/tasks/{id}/dependencies/{blocking_id}", h.RemoveTaskDependencyRoute, "DELETE")
	addProtectedRoute(r, h, "/api/tasks/{id}/complete-and-schedule", h.CompleteAndScheduleTaskRoute, "POST")

	// Calendar iCal feed - public route with token-based auth in handler
	// This allows external calendar apps to subscribe via ?token=XYZ
	addRoute(r, "/api/user/calendar.ics", h.CalendarICSRoute, "GET")

	// Regenerate CalDAV token - protected route
	addProtectedRoute(r, h, "/api/user/regenerate-caldav-token", h.RegenerateCalDAVTokenRoute, "POST")
}