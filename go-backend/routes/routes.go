package routes

import (
	"go-backend/handlers"
	"go-backend/handlers/admin"
	"github.com/gorilla/mux"
)

// RegisterAllRoutes registers all API routes for the application with consistent
// authentication and security model:
//
// Nearly all endpoints are registered via addProtectedRoute (authenticated).
// The following endpoints are intentionally public (via addRoute) to allow access
// without authentication for user onboarding/workflows:
//
// Authentication endpoints:
// - GET /api/auth/github - OAuth start (redirects user to GitHub)
// - GET /api/auth/github/callback - OAuth callback (GitHub redirects back)
// - POST /api/login - User login (creates JWT token)
// - POST /api/reset-password - Password reset completion
// - POST /api/email-validate - Email validation links
// - POST /api/request-reset - Request password reset email
//
// User creation:
// - POST /api/users - User signup/registration (creates new account)
//
// External integrations:
// - POST /api/stripe/webhook - Stripe payment webhooks (verified via webhook signature)
// - POST /api/mailing-list - Mailing list signup (public subscription)
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

	// External calendar event routes
	RegisterExternalEventRoutes(r, h)

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

	// Schema routes
	RegisterSchemaRoutes(r, h)

	// Job queue routes
	RegisterJobRoutes(r, h)

	// Admin email management routes
	RegisterAdminEmailRoutes(r, h)

	// Admin-specific routes (dashboard, audit logs, etc.)
	// Scheduler is nil until Task 7 (Integrate scheduler into main.go)
	admin.RegisterAllAdminRoutes(r, h, nil)
}