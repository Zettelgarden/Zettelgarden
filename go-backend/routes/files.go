package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterFileRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/files", h.GetAllFilesRoute, "GET")
	addProtectedRoute(r, h, "/api/files/upload", h.UploadFileRoute, "POST")
	addProtectedRoute(r, h, "/api/files/{id}", h.GetFileMetadataRoute, "GET")
	addProtectedRoute(r, h, "/api/files/{id}", h.EditFileMetadataRoute, "PATCH")
	addProtectedRoute(r, h, "/api/files/{id}", h.DeleteFileRoute, "DELETE")
	addProtectedRoute(r, h, "/api/files/download/{id}", h.DownloadFileRoute, "GET")

	// File tag management routes
	addProtectedRoute(r, h, "/api/files/tags", h.CreateFileTagRoute, "POST")
	addProtectedRoute(r, h, "/api/files/tags", h.GetUserFileTagsRoute, "GET")
	addProtectedRoute(r, h, "/api/files/{id}/tags", h.TagFileRoute, "POST")
	addProtectedRoute(r, h, "/api/files/{id}/tags/{tag_name}", h.UntagFileRoute, "DELETE")
}