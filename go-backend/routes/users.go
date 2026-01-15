package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterUserRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/users/{id}", h.GetUserRoute, "GET")
	addProtectedRoute(r, h, "/api/users/{id}", h.UpdateUserRoute, "PUT")
	addProtectedRoute(r, h, "/api/users", h.GetUsersRoute, "GET")
	addRoute(r, "/api/users", h.CreateUserRoute, "POST")
	addProtectedRoute(r, h, "/api/users/{id}/subscription", h.GetUserSubscriptionRoute, "GET")
	addProtectedRoute(r, h, "/api/billing/subscribe", h.CreateSubscriptionRoute, "POST")
	addProtectedRoute(r, h, "/api/billing/portal", h.BillingPortalRoute, "GET")
	addProtectedRoute(r, h, "/api/billing/public-key", h.StripePublicKeyRoute, "GET")
	addRoute(r, "/api/stripe/webhook", h.StripeWebhookRoute, "POST")

	addProtectedRoute(r, h, "/api/user/memory", h.GetUserMemoryRoute, "GET")
	addProtectedRoute(r, h, "/api/user/memory", h.UpdateUserMemoryRoute, "PUT")
	addProtectedRoute(r, h, "/api/current", h.GetCurrentUserRoute, "GET")
	addProtectedRoute(r, h, "/api/admin", h.GetUserAdminRoute, "GET")
}