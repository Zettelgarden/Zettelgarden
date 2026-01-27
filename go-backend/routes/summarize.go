package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterSummarizeRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/summarize", h.CreateSummarizationRoute, "POST")
	addProtectedRoute(r, h, "/api/summarize/{id}", h.GetSummarizationRoute, "GET")
	addProtectedRoute(r, h, "/api/summarize/{id}", h.CancelSummarizationRoute, "DELETE")
	addProtectedRoute(r, h, "/api/summarizations", h.ListSummarizationsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{card_pk:[0-9]+}/summaries", h.GetSummariesByCardRoute, "GET")
}