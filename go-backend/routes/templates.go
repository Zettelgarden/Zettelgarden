package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterTemplateRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/templates", h.GetTemplatesRoute, "GET")
	addProtectedRoute(r, h, "/api/templates", h.CreateTemplateRoute, "POST")
	addProtectedRoute(r, h, "/api/templates/{id}", h.GetTemplateRoute, "GET")
	addProtectedRoute(r, h, "/api/templates/{id}", h.UpdateTemplateRoute, "PUT")
	addProtectedRoute(r, h, "/api/templates/{id}", h.DeleteTemplateRoute, "DELETE")
}