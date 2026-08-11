package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterEntityRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/entities", h.GetEntitiesRoute, "GET")
	addProtectedRoute(r, h, "/api/entities/id/{id}", h.GetEntityByIDRoute, "GET")
	addProtectedRoute(r, h, "/api/entities/id/{id}/cards", h.GetEntityCardsRoute, "GET")
	addProtectedRoute(r, h, "/api/entities/name/{name}", h.GetEntityByNameRoute, "GET")
	addProtectedRoute(r, h, "/api/entities/merge", h.MergeEntitiesRoute, "POST")
	addProtectedRoute(r, h, "/api/entities/id/{id}", h.DeleteEntityRoute, "DELETE")
	addProtectedRoute(r, h, "/api/entities/id/{id}", h.UpdateEntityRoute, "PUT")
	addProtectedRoute(r, h, "/api/entities/{entityId}/cards/{cardId}", h.AddEntityToCardRoute, "POST")
	addProtectedRoute(r, h, "/api/entities/{entityId}/cards/{cardId}", h.RemoveEntityFromCardRoute, "DELETE")
	addProtectedRoute(r, h, "/api/entities/{id}/facts", h.GetEntityFacts, "GET")
	addProtectedRoute(r, h, "/api/entities/{id}/similar", h.GetSimilarEntitiesRoute, "GET")
}
