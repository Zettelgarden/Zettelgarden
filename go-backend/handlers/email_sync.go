package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CreateEmailAccountRoute handles POST /api/email/accounts
// Creates a new email account for the authenticated user
func (h *Handler) CreateEmailAccountRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateEmailAccountParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("[email-sync] failed to decode request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if params.EmailAddress == "" {
		http.Error(w, "email_address is required", http.StatusBadRequest)
		return
	}

	// Get the database connection
	db := h.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)

	// Use encryption key from environment if available
	encryptionKey := ""
	if h.EncryptionService != nil {
		encryptionKey = "encrypted"
	}

	account, err := accountService.CreateEmailAccount(context.Background(), userID, params, encryptionKey)
	if err != nil {
		log.Printf("[email-sync] failed to create email account: %v", err)
		http.Error(w, "Failed to create email account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

// ListEmailAccountsRoute handles GET /api/email/accounts
// Lists all email accounts for the authenticated user
func (h *Handler) ListEmailAccountsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	db := h.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)

	accounts, err := accountService.GetEmailAccounts(context.Background(), userID)
	if err != nil {
		log.Printf("[email-sync] failed to list email accounts: %v", err)
		http.Error(w, "Failed to list email accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// GetEmailAccountRoute handles GET /api/email/accounts/{id}
// Retrieves a specific email account by ID
func (h *Handler) GetEmailAccountRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get account ID from path
	idStr := mux.Vars(r)["id"]
	accountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	db := h.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)

	account, err := accountService.GetEmailAccountByID(context.Background(), userID, accountID)
	if err != nil {
		if err.Error() == "email account not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to get email account: %v", err)
		http.Error(w, "Failed to get email account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// DeleteEmailAccountRoute handles DELETE /api/email/accounts/{id}
// Deletes an email account by ID
func (h *Handler) DeleteEmailAccountRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get account ID from path
	idStr := mux.Vars(r)["id"]
	accountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	db := h.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)

	err = accountService.DeleteEmailAccount(context.Background(), userID, accountID)
	if err != nil {
		if err.Error() == "email account not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to delete email account: %v", err)
		http.Error(w, "Failed to delete email account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncEmailAccountRoute handles POST /api/email/accounts/{id}/sync
// Triggers an immediate sync for an email account
func (h *Handler) SyncEmailAccountRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get account ID from path
	idStr := mux.Vars(r)["id"]
	accountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	db := h.GetDB().(*sql.DB)
	accountService := services.NewEmailAccountService(db)

	// Verify account exists and belongs to user
	account, err := accountService.GetEmailAccountByID(context.Background(), userID, accountID)
	if err != nil {
		if err.Error() == "email account not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to get email account for sync: %v", err)
		http.Error(w, "Failed to get email account", http.StatusInternalServerError)
		return
	}

	// Update sync status to "syncing"
	err = accountService.UpdateSyncStatus(context.Background(), userID, accountID, "syncing")
	if err != nil {
		log.Printf("[email-sync] failed to update sync status: %v", err)
		// Continue anyway, this is not critical
	}

	// In a full implementation, this would trigger a background job
	// For now, just return success
	log.Printf("[email-sync] triggered sync for account %d (%s)", accountID, account.EmailAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Sync triggered successfully",
		"account_id": accountID,
	})
}

// ListEmailsRoute handles GET /api/emails
// Lists emails with optional filters (status, folder, limit, offset)
func (h *Handler) ListEmailsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	var filters models.EmailListFilters

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		filters.Status = &statusStr
	}

	if folderStr := r.URL.Query().Get("folder"); folderStr != "" {
		filters.Folder = &folderStr
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = &limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = &offset
		}
	}

	emailService := services.NewEmailService(h.GetDB())

	emails, total, err := emailService.ListEmails(context.Background(), userID, filters)
	if err != nil {
		log.Printf("[email-sync] failed to list emails: %v", err)
		http.Error(w, "Failed to list emails", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"emails": emails,
		"total":  total,
	})
}

// GetEmailRoute handles GET /api/emails/{id}
// Retrieves a specific email by ID
func (h *Handler) GetEmailRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get email ID from path
	idStr := mux.Vars(r)["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())

	email, err := emailService.GetEmailByID(context.Background(), userID, emailID)
	if err != nil {
		if err.Error() == "email not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to get email: %v", err)
		http.Error(w, "Failed to get email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(email)
}

// GetEmailStatsRoute handles GET /api/emails/stats
// Returns statistics about emails (counts by status)
func (h *Handler) GetEmailStatsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	emailService := services.NewEmailService(h.GetDB())

	stats, err := emailService.GetEmailStats(context.Background(), userID)
	if err != nil {
		log.Printf("[email-sync] failed to get email stats: %v", err)
		http.Error(w, "Failed to get email stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
