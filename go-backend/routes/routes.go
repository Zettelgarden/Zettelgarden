package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

// RegisterAllRoutes registers all API routes for the application
func RegisterAllRoutes(r *mux.Router, h *handlers.Handler) {
	// Authentication routes
	RegisterAuthRoutes(r, h)

	// API key routes
	RegisterAPIKeyRoutes(r, h)

	// File management routes
	RegisterFileRoutes(r, h)

	// Card management routes
	RegisterCardRoutes(r, h)

	// Template routes
	RegisterTemplateRoutes(r, h)

	// Search routes
	RegisterSearchRoutes(r, h)

	// User management routes
	RegisterUserRoutes(r, h)

	// Task management routes
	RegisterTaskRoutes(r, h)

	// Task status routes
	RegisterTaskStatusRoutes(r, h)

	// Statistics routes
	RegisterStatsRoutes(r, h)

	// Tag routes
	RegisterTagRoutes(r, h)

	// URL parsing routes
	RegisterURLRoutes(r, h)

	// Mailing list routes
	RegisterMailingListRoutes(r, h)

	// Entity routes
	RegisterEntityRoutes(r, h)

	// Fact routes
	RegisterFactRoutes(r, h)

	// Summarize routes
	RegisterSummarizeRoutes(r, h)

	// Chat routes
	RegisterChatRoutes(r, h)
}