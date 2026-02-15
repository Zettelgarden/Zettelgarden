package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

// RegisterExternalEventRoutes registers all external calendar event routes
func RegisterExternalEventRoutes(r *mux.Router, h *handlers.Handler) {
	// External calendar subscriptions
	addProtectedRoute(r, h, "/api/user/external-calendars", h.ListExternalCalendarsRoute, "GET")
	addProtectedRoute(r, h, "/api/user/external-calendars", h.CreateExternalCalendarRoute, "POST")
	addProtectedRoute(r, h, "/api/user/external-calendars/{id}", h.UpdateExternalCalendarRoute, "PUT")
	addProtectedRoute(r, h, "/api/user/external-calendars/{id}", h.DeleteExternalCalendarRoute, "DELETE")
	addProtectedRoute(r, h, "/api/user/external-calendars/{id}/sync", h.SyncExternalCalendarRoute, "POST")

	// External events (calendar CRUD)
	addProtectedRoute(r, h, "/api/user/external-calendars/{id}/events", h.CreateEventOnCalendarRoute, "POST")

	// External events
	addProtectedRoute(r, h, "/api/user/external-events", h.GetExternalEventsRoute, "GET")

	// Event-card linking
	addProtectedRoute(r, h, "/api/user/external-events/{id}/link", h.LinkEventToCardRoute, "PUT")
	addProtectedRoute(r, h, "/api/user/external-events/{id}/link", h.UnlinkEventFromCardRoute, "DELETE")
}
