package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterEmailRoutes(r *mux.Router, h *handlers.Handler) {
	// Email account management routes
	addProtectedRoute(r, h, "/api/email/accounts", h.ListEmailAccountsRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts", h.CreateEmailAccountRoute, "POST")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.GetEmailAccountRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.DeleteEmailAccountRoute, "DELETE")
	addProtectedRoute(r, h, "/api/email/accounts/{id}/sync", h.SyncEmailAccountRoute, "POST")

	// Email message routes
	addProtectedRoute(r, h, "/api/emails", h.ListEmailsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/{id}", h.GetEmailRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/{id}/status", h.UpdateEmailStatusRoute, "PATCH")
	addProtectedRoute(r, h, "/api/emails/{id}/convert", h.ConvertEmailToCardRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/stats", h.GetEmailStatsRoute, "GET")
}
