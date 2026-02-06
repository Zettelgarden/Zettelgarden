package routes

import (
	"go-backend/handlers"
	"go-backend/handlers/chat"
	"github.com/gorilla/mux"
)

func RegisterChatRoutes(r *mux.Router, h *handlers.Handler) {
	// Create chat handler
	chatHandler := chat.NewHandler(h.Server, h.ChatService)

	// Register all chat routes using the new chat package
	chat.RegisterRoutes(r, chatHandler)

	// Additional routes that haven't been migrated yet
	// TODO: Migrate these to the new chat package
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages/{messageId}/regenerate", h.RegenerateMessageRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/messages/{messageId}/edit", h.EditUserMessageRoute, "PUT")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/tools/retry", h.RetryToolCallRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/status", h.GetConversationStatusRoute, "GET")
	addProtectedRoute(r, h, "/api/chat/conversations/{id}/star", h.StarConversationRoute, "POST")
	addProtectedRoute(r, h, "/api/chat/models", h.GetChatModelsRoute, "GET")
}