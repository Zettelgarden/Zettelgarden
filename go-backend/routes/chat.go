package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterChatRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/chat/conversations", h.CreateConversationRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations", h.GetConversationsRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}", h.GetConversationRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages", h.SendMessageRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages/stream", h.StreamMessageRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages/{messageId}/regenerate", h.RegenerateMessageRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages/{messageId}/edit", h.EditUserMessageRoute, "PUT")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/tools/retry", h.RetryToolCallRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/status", h.GetConversationStatusRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}", h.DeleteConversationRoute, "DELETE")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/star", h.StarConversationRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/title", h.UpdateConversationTitleRoute, "PUT")
	addProtectedRoute(r, h, "/api/chat/usage", h.GetUsageQuotaRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/instructions", h.GetInstructionsRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/instructions", h.UpdateInstructionsRoute, "PUT")
}