package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

// RegisterSyncRoutes registers the local-first sync API (epic
// Zettelgarden-v5b, Phase 0b — issue tsv): snapshot bootstrap, incremental
// changes feed, and transactional batch push. All authenticated; the existing
// REST surface is untouched.
func RegisterSyncRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/sync/snapshot", h.SnapshotRoute, "GET")
	addProtectedRoute(r, h, "/api/sync/changes", h.ChangesRoute, "GET")
	addProtectedRoute(r, h, "/api/sync/push", h.PushRoute, "POST")
}
