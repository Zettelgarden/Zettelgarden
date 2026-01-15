package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterAuthRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/auth", h.CheckTokenRoute, "GET")
	addRoute(r, "/api/auth/github", h.StartGitHubOAuthRoute, "GET")
	addRoute(r, "/api/auth/github/callback", h.GitHubCallbackRoute, "GET")
	addRoute(r, "/api/login", h.LoginRoute, "POST")
	addRoute(r, "/api/reset-password", h.ResetPasswordRoute, "POST")
	addRoute(r, "/api/email-validate", h.ValidateEmailRoute, "POST")
	addRoute(r, "/api/request-reset", h.RequestPasswordResetRoute, "POST")
}