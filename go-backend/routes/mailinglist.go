package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterMailingListRoutes(r *mux.Router, h *handlers.Handler) {
	addRoute(r, "/api/mailing-list", h.AddToMailingListRoute, "POST")
	addProtectedRoute(r, h, "/api/mailing-list", h.GetMailingListSubscribersRoute, "GET")
	addProtectedRoute(r, h, "/api/mailing-list/messages", h.GetMailingListMessagesRoute, "GET")
	addProtectedRoute(r, h, "/api/mailing-list/messages/send", h.SendMailingListMessageRoute, "POST")
	addProtectedRoute(r, h, "/api/mailing-list/messages/recipients", h.GetMessageRecipientsRoute, "GET")
	addProtectedRoute(r, h, "/api/mailing-list/unsubscribe", h.UnsubscribeMailingListRoute, "POST")
}