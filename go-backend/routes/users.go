package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterUserRoutes(r *mux.Router, h *handlers.Handler) {
	// Admin-only routes
	addAdminRoute(r, h, "/api/users/{id}", h.GetUserRoute, "GET") // View any user
	addAdminRoute(r, h, "/api/users", h.GetUsersRoute, "GET")     // List all users

	// Admin-or-self routes (admin can access any, users can access their own)
	addAdminOrSelfRoute(r, h, "/api/users/{id}", h.UpdateUserRoute, "PUT", "id")
	addAdminOrSelfRoute(r, h, "/api/users/{id}/subscription", h.GetUserSubscriptionRoute, "GET", "id")

	// Admin-only account deletion
	addAdminRoute(r, h, "/api/users/{id}", h.DeleteUserRoute, "DELETE")

	// Protected routes for current user
	addProtectedRoute(r, h, "/api/current", h.GetCurrentUserRoute, "GET")
	addProtectedRoute(r, h, "/api/user/memory", h.GetUserMemoryRoute, "GET")
	addProtectedRoute(r, h, "/api/user/memory", h.UpdateUserMemoryRoute, "PUT")
	addProtectedRoute(r, h, "/api/admin", h.GetUserAdminRoute, "GET")

	// Self-serve data export + account deletion (6er.9)
	addProtectedRoute(r, h, "/api/user/export", h.ExportUserDataRoute, "GET")
	addProtectedRoute(r, h, "/api/user", h.DeleteAccountRoute, "DELETE")

	// User signup/registration (public for new account creation)
	addRoute(r, "/api/users", h.CreateUserRoute, "POST")

	// Billing routes (protected but not admin-only). Handlers return 404 when
	// billing is disabled via STRIPE_ENABLED=false.
	addProtectedRoute(r, h, "/api/billing/subscribe", h.CreateSubscriptionRoute, "POST")
	addProtectedRoute(r, h, "/api/billing/portal", h.BillingPortalRoute, "GET")
	addProtectedRoute(r, h, "/api/billing/status", h.BillingStatusRoute, "GET")
	addProtectedRoute(r, h, "/api/billing/public-key", h.StripePublicKeyRoute, "GET")

	// Stripe webhook (public for payment processing, verified via signature)
	addRoute(r, "/api/stripe/webhook", h.StripeWebhookRoute, "POST")
}
