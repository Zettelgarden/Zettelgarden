package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterTaskSavedSearchRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/task-saved-searches", h.GetTaskSavedSearchesRoute, "GET")
	addProtectedRoute(r, h, "/api/task-saved-searches", h.CreateTaskSavedSearchRoute, "POST")
	addProtectedRoute(r, h, "/api/task-saved-searches/{id}", h.GetTaskSavedSearchRoute, "GET")
	addProtectedRoute(r, h, "/api/task-saved-searches/{id}", h.UpdateTaskSavedSearchRoute, "PUT")
	addProtectedRoute(r, h, "/api/task-saved-searches/{id}", h.DeleteTaskSavedSearchRoute, "DELETE")
}
