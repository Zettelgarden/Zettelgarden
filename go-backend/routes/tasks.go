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

	// Subtask routes
	addProtectedRoute(r, h, "/api/tasks/{id}/subtasks", h.CreateSubtaskRoute, "POST")
	addProtectedRoute(r, h, "/api/tasks/{id}/subtasks", h.GetSubtasksRoute, "GET")
	addProtectedRoute(r, h, "/api/tasks/{id}/parent", h.SetTaskParentRoute, "PATCH")

}