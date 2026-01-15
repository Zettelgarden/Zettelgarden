package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterTaskStatusRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/task-statuses", h.GetTaskStatusesRoute, "GET")
	addProtectedRoute(r, h, "/api/task-statuses", h.CreateTaskStatusRoute, "POST")
	addProtectedRoute(r, h, "/api/task-statuses/{id}", h.GetTaskStatusRoute, "GET")
	addProtectedRoute(r, h, "/api/task-statuses/{id}", h.UpdateTaskStatusRoute, "PUT")
	addProtectedRoute(r, h, "/api/task-statuses/{id}", h.DeleteTaskStatusRoute, "DELETE")
	addProtectedRoute(r, h, "/api/task-statuses/reorder", h.ReorderTaskStatusesRoute, "POST")
}