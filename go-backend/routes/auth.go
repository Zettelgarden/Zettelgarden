package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterAuthRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/auth", h.CheckTokenRoute, "GET")
	// OAuth start - redirects to GitHub (public for auth flow)
	addRoute(r, "/api/auth/github", h.StartGitHubOAuthRoute, "GET")
	// OAuth callback - handles GitHub redirect back (public for auth flow)
	addRoute(r, "/api/auth/github/callback", h.GitHubCallbackRoute, "GET")
	// OIDC start - redirects to the configured IdP, e.g. Pocket ID (public)
	addRoute(r, "/api/auth/oidc/start", h.StartOIDCRoute, "GET")
	// OIDC callback - handles IdP redirect back (public for auth flow)
	addRoute(r, "/api/auth/oidc/callback", h.CallbackOIDCRoute, "GET")
	// User login (public for authentication)
	addRoute(r, "/api/login", h.LoginRoute, "POST")
	// Password reset completion (public via email link)
	addRoute(r, "/api/reset-password", h.ResetPasswordRoute, "POST")
	// Email validation links (public via email verification)
	addRoute(r, "/api/email-validate", h.ValidateEmailRoute, "POST")
	// Request password reset email (public for password recovery)
	addRoute(r, "/api/request-reset", h.RequestPasswordResetRoute, "POST")
}
