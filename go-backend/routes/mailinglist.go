package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterMailingListRoutes(r *mux.Router, h *handlers.Handler) {
	// Mailing list signup (public subscription)
	addRoute(r, "/api/mailing-list", h.AddToMailingListRoute, "POST")

	// Admin-only routes for managing mailing list
	addAdminRoute(r, h, "/api/mailing-list", h.GetMailingListSubscribersRoute, "GET")
	addAdminRoute(r, h, "/api/mailing-list/messages", h.GetMailingListMessagesRoute, "GET")
	addAdminRoute(r, h, "/api/mailing-list/messages/send", h.SendMailingListMessageRoute, "POST")
	addAdminRoute(r, h, "/api/mailing-list/messages/recipients", h.GetMessageRecipientsRoute, "GET")
	addAdminRoute(r, h, "/api/mailing-list/unsubscribe", h.UnsubscribeMailingListRoute, "POST")
}