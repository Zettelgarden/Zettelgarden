package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterRSSRoutes(r *mux.Router, h *handlers.Handler) {
	// Feed discovery route
	addProtectedRoute(r, h, "/api/rss/discover", h.DiscoverFeedRoute, "POST")

	addProtectedRoute(r, h, "/api/rss/feeds", h.ListRSSFeedsRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/feeds", h.CreateRSSFeedRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.GetRSSFeedRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.UpdateRSSFeedRoute, "PUT")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.DeleteRSSFeedRoute, "DELETE")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}/read", h.MarkRSSFeedAsReadRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/feeds/fetch", h.RefreshRSSFeedsRoute, "POST")

	addProtectedRoute(r, h, "/api/rss/articles", h.ListRSSArticlesRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/articles/{id}", h.GetRSSArticleRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/articles/{id}/read", h.MarkRSSArticleAsReadRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/articles/{id}/convert", h.ConvertRSSArticleToCardRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/smart", h.ListSmartRSSArticlesRoute, "GET")

	addProtectedRoute(r, h, "/api/rss/folders", h.ListRSSFoldersRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/folders", h.CreateRSSFolderRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/folders/{id}", h.UpdateRSSFolderRoute, "PUT")
	addProtectedRoute(r, h, "/api/rss/folders/{id}", h.DeleteRSSFolderRoute, "DELETE")
	addProtectedRoute(r, h, "/api/rss/folders/{id}/read", h.MarkRSSFolderAsReadRoute, "POST")

	addProtectedRoute(r, h, "/api/rss/unread-counts", h.GetUnreadCountsRoute, "GET")

	// OPML import/export routes
	addProtectedRoute(r, h, "/api/rss/opml/export", h.ExportOPMLRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/opml/import", h.ImportOPMLRoute, "POST")
}
