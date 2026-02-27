package routes

import (
	"github.com/gorilla/mux"
	"go-backend/handlers"
)

func RegisterEmailRoutes(r *mux.Router, h *handlers.Handler) {
	// Email account management routes
	addProtectedRoute(r, h, "/api/email/accounts", h.ListEmailAccountsRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts", h.CreateEmailAccountRoute, "POST")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.GetEmailAccountRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.DeleteEmailAccountRoute, "DELETE")
	addProtectedRoute(r, h, "/api/email/accounts/{id}/sync", h.SyncEmailAccountRoute, "POST")

	// Email message routes
	addProtectedRoute(r, h, "/api/emails", h.ListEmailsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/search", h.SearchEmailsRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/{id}", h.GetEmailRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/{id}/status", h.UpdateEmailStatusRoute, "PATCH")
	addProtectedRoute(r, h, "/api/emails/{id}/convert", h.ConvertEmailToCardRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/{id}/extract-facts", h.ExtractFactsFromEmailRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/{id}/save-facts", h.SaveFactsFromEmailRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/{id}/facts", h.GetEmailFactsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/stats", h.GetEmailStatsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/top-senders", h.GetTopSendersRoute, "GET")

	// Email attachment routes
	addProtectedRoute(r, h, "/api/emails/{id}/attachments", h.GetEmailAttachmentsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/attachments/{id}/download", h.DownloadEmailAttachmentRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/attachments/{id}/thumbnail", h.GetEmailAttachmentThumbnailRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/attachments/{id}/save-to-vault", h.SaveEmailAttachmentToVaultRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/attachments/{id}", h.DeleteEmailAttachmentRoute, "DELETE")

	// Batch operation routes
	addProtectedRoute(r, h, "/api/emails/batch-archive", h.BatchArchiveEmailsRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/batch-convert", h.BatchConvertEmailsRoute, "POST")
	addProtectedRoute(r, h, "/api/emails/batch-create-tasks", h.BatchCreateTasksRoute, "POST")

	// Thread routes
	addProtectedRoute(r, h, "/api/emails/threads/{thread_id}", h.GetEmailThreadRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/threads/{thread_id}/read", h.MarkThreadAsReadRoute, "PATCH")
	addProtectedRoute(r, h, "/api/emails/threads/{thread_id}/archive", h.ArchiveThreadRoute, "PATCH")
}
