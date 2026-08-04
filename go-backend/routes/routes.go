package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
	"go-backend/handlers/admin"
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
// - GET /api/auth/oidc/start - OIDC start (redirects user to the IdP, e.g. Pocket ID)
// - GET /api/auth/oidc/callback - OIDC callback (IdP redirects back)
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
func RegisterAllRoutes(r *mux.Router, h *handlers.Handler, scheduler handlers.SchedulerAPI) {
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

	// Task saved search routes
	RegisterTaskSavedSearchRoutes(r, h)

	// Statistics routes
	RegisterStatsRoutes(r, h)

	// Tag routes
	RegisterTagRoutes(r, h)

	// URL parsing routes
	RegisterURLRoutes(r, h)

	// Mailing list routes
	RegisterMailingListRoutes(r, h)

	// RSS feed routes
	RegisterRSSRoutes(r, h)

	// Entity routes
	RegisterEntityRoutes(r, h)

	// Fact routes
	RegisterFactRoutes(r, h)

	// Summarize routes
	RegisterSummarizeRoutes(r, h)

	// Schema routes
	RegisterSchemaRoutes(r, h)

	// Job queue routes
	RegisterJobRoutes(r, h)

	// Admin-specific routes (dashboard, audit logs, etc.)
	admin.RegisterAllAdminRoutes(r, h, scheduler)

	// Notification routes
	RegisterNotificationRoutes(r, h)
}
