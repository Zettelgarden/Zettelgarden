package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterCardRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/cards", h.CreateCardRoute, "POST")
	addProtectedRoute(r, h, "/api/cards/next-root-id", h.GetNextRootCardIDRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/suggest-title", h.SuggestCardTitleRoute, "POST")
	addProtectedRoute(r, h, "/api/cards/starred", h.GetStarredCardsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/unsorted", h.GetUnsortedCardsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}", h.GetCardRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}", h.UpdateCardRoute, "PUT")
	addProtectedRoute(r, h, "/api/cards/{id}", h.DeleteCardRoute, "DELETE")
	addProtectedRoute(r, h, "/api/cards/{id}/audit", h.GetCardAuditEventsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/star", h.StarCardRoute, "POST")
	addProtectedRoute(r, h, "/api/cards/{id}/star", h.UnstarCardRoute, "DELETE")
	addProtectedRoute(r, h, "/api/cards/{id}/facts", h.GetCardFacts, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/references", h.GetCardReferencesRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/children", h.GetCardChildrenRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/tree", h.GetCardWithDescendantsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/tree/depth/{depth}", h.GetCardWithDescendantsPaginatedRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/next-child-id", h.GetNextChildCardIDRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/files", h.GetCardFilesRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/tags", h.GetCardTagsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/tasks", h.GetCardTasksRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/entities", h.GetCardEntitiesRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{card_pk:[0-9]+}/linked-entities", h.GetEntityByLinkedCardPKRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{card_pk:[0-9]+}/analysis", h.GetCardAnalysisRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{id}/audit/{auditEventId}/restore", h.RestoreCardToAuditEventRoute, "POST")
}