package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterFactRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/facts/{id}/entities", h.GetFactEntities, "GET")
	addProtectedRoute(r, h, "/api/facts", h.GetAllFacts, "GET")
	addProtectedRoute(r, h, "/api/facts/{id}", h.GetFact, "GET")
	addProtectedRoute(r, h, "/api/facts/{id}", h.UpdateFact, "PUT")
	addProtectedRoute(r, h, "/api/facts/{factID}/cards/{cardID}", h.LinkFactToCardHandler, "POST")
	addProtectedRoute(r, h, "/api/facts/merge", h.MergeFactsRoute, "POST")
	addProtectedRoute(r, h, "/api/facts/{id}/cards", h.GetFactCards, "GET")
	addProtectedRoute(r, h, "/api/facts/{id}/similar", h.GetSimilarFacts, "GET")
	addProtectedRoute(r, h, "/api/facts/{id}", h.DeleteFactRoute, "DELETE")
}