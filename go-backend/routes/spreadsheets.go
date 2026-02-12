package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterSpreadsheetRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/cards/{cardId}/spreadsheets", h.GetSpreadsheetsRoute, "GET")
	addProtectedRoute(r, h, "/api/cards/{cardId}/spreadsheets", h.CreateSpreadsheetRoute, "POST")
	addProtectedRoute(r, h, "/api/spreadsheets/{id}", h.GetSpreadsheetRoute, "GET")
	addProtectedRoute(r, h, "/api/spreadsheets/{id}", h.UpdateSpreadsheetRoute, "PUT")
	addProtectedRoute(r, h, "/api/spreadsheets/{id}", h.DeleteSpreadsheetRoute, "DELETE")
}
