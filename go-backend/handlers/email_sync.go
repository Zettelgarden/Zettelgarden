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
	"strings"

	"github.com/gorilla/mux"
)

// CreateEmailAccountRoute handles POST /api/email/accounts
// Creates a new email account for the authenticated user and verifies credentials
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

	// Validate app password
	if params.ApiToken == nil || *params.ApiToken == "" {
		http.Error(w, "api_token is required", http.StatusBadRequest)
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

	// Verify credentials by connecting to Fastmail
	log.Printf("[email-sync] verifying credentials for account %d (%s)", account.ID, account.EmailAddress)

	// Decrypt the app password
	password, err := services.DecryptApiToken(*account.ApiTokenEncrypted, "")
	if err != nil {
		log.Printf("[email-sync] failed to decrypt password for account %d: %v", account.ID, err)
		// Delete the account since we can't use it
		_ = accountService.DeleteEmailAccount(context.Background(), userID, account.ID)
		http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
		return
	}

	// Create JMAP client and try to connect
	jmapClient := services.NewJMAPClient(account.JMAPServerURL, password)
	if err := jmapClient.Connect(context.Background()); err != nil {
		log.Printf("[email-sync] credential verification failed for account %d: %v", account.ID, err)
		// Delete the account since credentials are invalid
		_ = accountService.DeleteEmailAccount(context.Background(), userID, account.ID)

		// Return a more specific error message for authentication failures
		if strings.Contains(err.Error(), "authentication failed") || strings.Contains(err.Error(), "Unauthorized") {
			http.Error(w, "Authentication failed - please check your email address and app password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to connect to Fastmail: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[email-sync] credential verification successful for account %d (%s)", account.ID, account.EmailAddress)

	// Update the sync status to "active"
	err = accountService.UpdateSyncStatus(context.Background(), userID, account.ID, "active")
	if err != nil {
		log.Printf("[email-sync] warning: failed to update sync status for account %d: %v", account.ID, err)
		// Continue anyway, this is not critical
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
	emailService := services.NewEmailService(db)

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

	log.Printf("[email-sync] starting sync for account %d (%s)", accountID, account.EmailAddress)

	// Decrypt the app password
	password, err := services.DecryptApiToken(*account.ApiTokenEncrypted, "")
	if err != nil {
		log.Printf("[email-sync] failed to decrypt password for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
		return
	}

	// Create JMAP client and connect
	jmapClient := services.NewJMAPClient(account.JMAPServerURL, password)
	if err := jmapClient.Connect(context.Background()); err != nil {
		log.Printf("[email-sync] failed to connect to JMAP server for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to connect to Fastmail: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch emails from Fastmail
	emails, queryState, err := jmapClient.FetchEmails(context.Background(), 50)
	if err != nil {
		log.Printf("[email-sync] failed to fetch emails for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to fetch emails: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[email-sync] fetched %d emails for account %d", len(emails), accountID)

	// Store emails in database
	storedCount := 0
	for _, email := range emails {
		email.UserID = userID
		email.EmailAccountID = &accountID

		// Create email (upsert logic handles duplicates)
		_, err := emailService.CreateEmail(context.Background(), email)
		if err != nil {
			log.Printf("[email-sync] warning: failed to store email %s: %v", email.MessageID, err)
			continue
		}
		storedCount++
	}

	log.Printf("[email-sync] stored %d/%d emails for account %d", storedCount, len(emails), accountID)

	// Update last_sync_at and jmap_state
	err = accountService.UpdateJMAPState(context.Background(), userID, accountID, queryState)
	if err != nil {
		log.Printf("[email-sync] warning: failed to update JMAP state for account %d: %v", accountID, err)
	}

	// Update sync status to "active"
	err = accountService.UpdateSyncStatus(context.Background(), userID, accountID, "active")
	if err != nil {
		log.Printf("[email-sync] warning: failed to update sync status for account %d: %v", accountID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Sync completed successfully",
		"account_id":   accountID,
		"emails_fetched": len(emails),
		"emails_stored":  storedCount,
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
