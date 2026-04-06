package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

// RegisterAgentRoutes registers agent management API endpoints
// All endpoints require JWT authentication
func RegisterAgentRoutes(r *mux.Router, h *handlers.Handler) {
	// Agent CRUD endpoints
	r.HandleFunc("/agents", h.JwtMiddleware(h.CreateAgentHandler)).Methods("POST")
	r.HandleFunc("/agents", h.JwtMiddleware(h.ListAgentsHandler)).Methods("GET")
	r.HandleFunc("/agents/{id}", h.JwtMiddleware(h.RevokeAgentHandler)).Methods("DELETE")
	r.HandleFunc("/agents/{id}/activity", h.JwtMiddleware(h.GetAgentActivityHandler)).Methods("GET")
}
