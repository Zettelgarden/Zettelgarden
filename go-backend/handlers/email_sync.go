package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	if params.AppPassword == nil || *params.AppPassword == "" {
		http.Error(w, "app_password is required", http.StatusBadRequest)
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

	// Verify credentials by connecting to Fastmail via IMAP
	log.Printf("[email-sync] verifying credentials for account %d (%s)", account.ID, account.EmailAddress)

	// Decrypt the app password
	password, err := services.DecryptAppPassword(*account.AppPasswordEncrypted, "")
	if err != nil {
		log.Printf("[email-sync] failed to decrypt password for account %d: %v", account.ID, err)
		// Delete the account since we can't use it
		_ = accountService.DeleteEmailAccount(context.Background(), userID, account.ID)
		http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
		return
	}

	// Create IMAP client and try to connect
	imapClient := services.NewIMAPClient(account.IMAPServer, account.EmailAddress, password)
	if err := imapClient.Connect(context.Background()); err != nil {
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
	imapClient.Close()

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
	password, err := services.DecryptAppPassword(*account.AppPasswordEncrypted, "")
	if err != nil {
		log.Printf("[email-sync] failed to decrypt password for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
		return
	}

	// Create IMAP client and connect
	imapClient := services.NewIMAPClient(account.IMAPServer, account.EmailAddress, password)
	if err := imapClient.Connect(context.Background()); err != nil {
		log.Printf("[email-sync] failed to connect to IMAP server for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to connect to Fastmail: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer imapClient.Close()

	// Select INBOX
	if err := imapClient.SelectInbox(context.Background()); err != nil {
		log.Printf("[email-sync] failed to select INBOX for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to select INBOX: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch recent emails with attachments (for automatic file vault creation)
	emails, maxUID, err := imapClient.FetchRecentEmailsWithAttachments(context.Background(), 50)
	if err != nil {
		log.Printf("[email-sync] failed to fetch emails for account %d: %v", accountID, err)
		accountService.UpdateSyncStatus(context.Background(), userID, accountID, "error")
		http.Error(w, "Failed to fetch emails: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[email-sync] fetched %d emails for account %d", len(emails), accountID)

	// Create S3 attachment handler for file uploads
	s3Handler := &S3AttachmentHandler{h: h}
	attachmentService := services.NewEmailAttachmentService(db, s3Handler, userID)

	// Store emails in database
	storedCount := 0
	for _, emailWithAtt := range emails {
		email := emailWithAtt.Email
		email.UserID = userID
		email.EmailAccountID = &accountID

		// Create email (upsert logic handles duplicates)
		storedEmail, err := emailService.CreateEmail(context.Background(), email)
		if err != nil {
			log.Printf("[email-sync] warning: failed to store email %s: %v", email.MessageID, err)
			continue
		}

		// Process attachments: upload to S3 and automatically save to file vault
		for _, att := range emailWithAtt.Attachments {
			// Skip inline attachments (they're embedded in HTML)
			if att.IsInline {
				continue
			}

			// Upload attachment to S3 and create email_attachment record
			createdAtt, err := attachmentService.CreateAttachmentWithData(
				context.Background(),
				storedEmail.ID,
				att.Filename,
				att.ContentType,
				att.ContentID,
				att.IsInline,
				att.Data,
			)
			if err != nil {
				log.Printf("[email-sync] warning: failed to save attachment %s: %v", att.Filename, err)
				continue
			}

			// Automatically save to file vault
			_, err = attachmentService.SaveToFileVault(context.Background(), userID, createdAtt.ID, nil)
			if err != nil {
				log.Printf("[email-sync] warning: failed to save attachment %s to vault: %v", att.Filename, err)
				// Continue anyway - attachment is stored in S3
			} else {
				log.Printf("[email-sync] automatically saved attachment %s to file vault", att.Filename)
			}
		}

		storedCount++
	}

	log.Printf("[email-sync] stored %d/%d emails for account %d", storedCount, len(emails), accountID)

	totalEmails := len(emails)

	// Discover archive folder name
	archiveFolder, err := imapClient.FindArchiveMailbox(context.Background())
	if err != nil {
		log.Printf("[email-sync] could not find archive folder for account %d: %v", accountID, err)
		archiveFolder = "Archive" // fallback to default
	}

	// Sync from Archive folder
	err = imapClient.SelectMailbox(context.Background(), archiveFolder)
	if err != nil {
		log.Printf("[email-sync] failed to select %s for account %d: %v", archiveFolder, accountID, err)
		// Continue - Archive might not exist
	} else {
		log.Printf("[email-sync] syncing from %s folder for account %d", archiveFolder, accountID)
		archiveMailEmails, _, err := imapClient.FetchRecentEmailsWithAttachments(context.Background(), 100)
		if err != nil {
			log.Printf("[email-sync] failed to fetch from %s: %v", archiveFolder, err)
		} else {
			for _, emailWithAtt := range archiveMailEmails {
				email := emailWithAtt.Email
				email.UserID = userID
				email.EmailAccountID = &accountID
				storedEmail, err := emailService.CreateEmail(context.Background(), email)
				if err != nil {
					log.Printf("[email-sync] warning: failed to store archive email %s: %v", email.MessageID, err)
				} else {
					totalEmails++

					// Process attachments for archive emails
					for _, att := range emailWithAtt.Attachments {
						if att.IsInline {
							continue
						}

						// Upload attachment to S3 and create email_attachment record
						createdAtt, err := attachmentService.CreateAttachmentWithData(
							context.Background(),
							storedEmail.ID,
							att.Filename,
							att.ContentType,
							att.ContentID,
							att.IsInline,
							att.Data,
						)
						if err != nil {
							log.Printf("[email-sync] warning: failed to save archive attachment %s: %v", att.Filename, err)
							continue
						}

						// Automatically save to file vault
						_, err = attachmentService.SaveToFileVault(context.Background(), userID, createdAtt.ID, nil)
						if err != nil {
							log.Printf("[email-sync] warning: failed to save archive attachment %s to vault: %v", att.Filename, err)
						} else {
							log.Printf("[email-sync] automatically saved archive attachment %s to file vault", att.Filename)
						}
					}
				}
			}
			log.Printf("[email-sync] synced %d emails from %s for account %d", len(archiveMailEmails), archiveFolder, accountID)
		}
	}

	// Reconcile: detect external changes (emails moved by user in email client)
	log.Printf("[email-sync] starting reconciliation for account %d", accountID)

	// Get all emails for this account from database
	dbEmails, _, err := emailService.ListEmails(context.Background(), userID, models.EmailListFilters{
		Limit: func() *int { l := 10000; return &l }(),
	})
	if err != nil {
		log.Printf("[email-sync] failed to get emails for reconciliation: %v", err)
	} else {
		// Build a map of normalized Message-ID to email in database
		dbEmailMap := make(map[string]models.Email)
		for _, email := range dbEmails {
			normalized := services.NormalizeMessageID(email.MessageID)
			dbEmailMap[normalized] = email
		}
		log.Printf("[email-sync] reconciling %d emails from database for account %d", len(dbEmailMap), accountID)

		// Check INBOX
		if err := imapClient.SelectMailbox(context.Background(), "INBOX"); err != nil {
			log.Printf("[email-sync] failed to select INBOX for reconciliation: %v", err)
		} else {
			inboxMessageIDs, err := imapClient.GetAllMessageUIDs(context.Background())
			if err != nil {
				log.Printf("[email-sync] failed to get INBOX Message-IDs: %v", err)
			} else {
				log.Printf("[email-sync] found %d messages in INBOX for reconciliation", len(inboxMessageIDs))
				reconciledCount := 0

				for _, dbEmail := range dbEmails {
					if dbEmail.EmailAccountID == nil || *dbEmail.EmailAccountID != accountID {
						continue
					}
					normalizedDBID := services.NormalizeMessageID(dbEmail.MessageID)
					inInbox := inboxMessageIDs[normalizedDBID]

					if inInbox && dbEmail.Status == "archived" {
						log.Printf("[email-sync] reconciliation: email %s found in INBOX, unarchiving", dbEmail.MessageID)
						status := "unprocessed"
						if err := emailService.UpdateEmailFolder(context.Background(), userID, dbEmail.MessageID, "INBOX", &status); err != nil {
							log.Printf("[email-sync] failed to update email: %v", err)
						} else {
							reconciledCount++
						}
					}
				}
				log.Printf("[email-sync] reconciled %d emails from INBOX", reconciledCount)
			}
		}

		// Check Archive
		if err := imapClient.SelectMailbox(context.Background(), archiveFolder); err != nil {
			log.Printf("[email-sync] failed to select %s for reconciliation: %v", archiveFolder, err)
		} else {
			archiveMessageIDs, err := imapClient.GetAllMessageUIDs(context.Background())
			if err != nil {
				log.Printf("[email-sync] failed to get %s Message-IDs: %v", archiveFolder, err)
			} else {
				log.Printf("[email-sync] found %d messages in %s for reconciliation", len(archiveMessageIDs), archiveFolder)
				reconciledCount := 0

				for _, dbEmail := range dbEmails {
					if dbEmail.EmailAccountID == nil || *dbEmail.EmailAccountID != accountID {
						continue
					}
					normalizedDBID := services.NormalizeMessageID(dbEmail.MessageID)
					inArchive := archiveMessageIDs[normalizedDBID]

					if inArchive && dbEmail.Status != "archived" {
						log.Printf("[email-sync] reconciliation: email %s found in %s, archiving", dbEmail.MessageID, archiveFolder)
						status := "archived"
						if err := emailService.UpdateEmailFolder(context.Background(), userID, dbEmail.MessageID, archiveFolder, &status); err != nil {
							log.Printf("[email-sync] failed to update email: %v", err)
						} else {
							reconciledCount++
						}
					}
				}
				log.Printf("[email-sync] reconciled %d emails from %s", reconciledCount, archiveFolder)
			}
		}
	}

	// Update last_sync_at and IMAP state
	currentUIDValidity := imapClient.GetUIDValidity()
	err = accountService.UpdateIMAPState(context.Background(), userID, accountID, maxUID, currentUIDValidity)
	if err != nil {
		log.Printf("[email-sync] warning: failed to update IMAP state for account %d: %v", accountID, err)
	}

	// Update sync status to "active"
	err = accountService.UpdateSyncStatus(context.Background(), userID, accountID, "active")
	if err != nil {
		log.Printf("[email-sync] warning: failed to update sync status for account %d: %v", accountID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Sync completed successfully",
		"account_id":     accountID,
		"emails_fetched": totalEmails,
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

	if isReadStr := r.URL.Query().Get("is_read"); isReadStr != "" {
		if isRead, err := strconv.ParseBool(isReadStr); err == nil {
			filters.IsRead = &isRead
		}
	}

	if fromAddressStr := r.URL.Query().Get("from_address"); fromAddressStr != "" {
		filters.FromAddress = &fromAddressStr
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

// GetTopSendersRoute handles GET /api/emails/top-senders
// Returns top senders by email count with optional status filter
func (h *Handler) GetTopSendersRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	var statusFilter *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		statusFilter = &statusStr
	}

	limit := 10 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	emailService := services.NewEmailService(h.GetDB())

	senders, err := emailService.GetTopSenders(context.Background(), userID, statusFilter, limit)
	if err != nil {
		log.Printf("[email-sync] failed to get top senders: %v", err)
		http.Error(w, "Failed to get top senders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"senders": senders,
	})
}

// UpdateEmailStatusParams represents parameters for updating email status
type UpdateEmailStatusParams struct {
	Status string `json:"status"`
}

// UpdateEmailStatusRoute handles PATCH /api/emails/{id}/status
// Updates the status of an email (e.g., archive, triage, delete)
// For archived status, also moves the email to the Archive folder in IMAP
func (h *Handler) UpdateEmailStatusRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var params UpdateEmailStatusParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	if params.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	db := h.GetDB()

	// Get the current email first to check if we need to move in IMAP
	currentEmail, err := emailService.GetEmailByID(context.Background(), userID, emailID)
	if err != nil {
		log.Printf("[email-sync] failed to get email: %v", err)
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Handle IMAP folder movement for archive/unarchive
	isArchiving := params.Status == "archived" && currentEmail.Status != "archived"
	isUnarchiving := params.Status != "archived" && currentEmail.Status == "archived"

	if isArchiving || isUnarchiving {
		// Only move in IMAP if we have an email account
		if currentEmail.EmailAccountID != nil {
			// Get email account credentials
			var accountID int
			var emailAddress string
			var encryptedPassword string
			var imapServer string

			err := db.QueryRowContext(context.Background(), `
				SELECT id, email_address, app_password_encrypted, imap_server
				FROM email_accounts
				WHERE id = $1 AND user_id = $2
			`, *currentEmail.EmailAccountID, userID).Scan(&accountID, &emailAddress, &encryptedPassword, &imapServer)
			if err != nil {
				log.Printf("[email-sync] failed to get email account: %v", err)
				http.Error(w, "Failed to get email account", http.StatusInternalServerError)
				return
			}

			// Decrypt password
			password, err := services.DecryptAppPassword(encryptedPassword, "")
			if err != nil {
				log.Printf("[email-sync] failed to decrypt password: %v", err)
				http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
				return
			}

			// Create a client with the specific mailbox based on operation
			var imapClient *services.IMAPClient
			var searchMailbox string
			if isUnarchiving {
				// Unarchiving: email should be in Archive, move to INBOX
				searchMailbox = "Archive"
				imapClient = services.NewIMAPClientWithMailbox(imapServer, emailAddress, password, "Archive")
			} else {
				// Archiving: email should be in INBOX, move to Archive
				searchMailbox = "INBOX"
				imapClient = services.NewIMAPClientWithMailbox(imapServer, emailAddress, password, "INBOX")
			}

			if err := imapClient.Connect(context.Background()); err != nil {
				log.Printf("[email-sync] failed to connect to IMAP: %v", err)
				// Don't fail the request - just log it
			} else if err := imapClient.SelectInbox(context.Background()); err != nil {
				log.Printf("[email-sync] failed to select mailbox: %v", err)
				// Don't fail the request - just log it
				imapClient.Close()
			} else {
				defer imapClient.Close()

				// Get UID if missing
				uidToMove := uint32(0)
				if currentEmail.IMAPUID != nil {
					uidToMove = uint32(*currentEmail.IMAPUID)
				} else {
					// Backfill: find UID by Message-ID
					log.Printf("[email-sync] backfilling IMAP UID for email %s in %s", currentEmail.MessageID, searchMailbox)
					foundUID, err := imapClient.FindUIDByMessageID(context.Background(), currentEmail.MessageID)
					if err != nil {
						// Try the other mailbox as fallback
						fallbackMailbox := "INBOX"
						if searchMailbox == "INBOX" {
							fallbackMailbox = "Archive"
						}
						log.Printf("[email-sync] not found in %s, trying %s", searchMailbox, fallbackMailbox)

						fallbackClient := services.NewIMAPClientWithMailbox(imapServer, emailAddress, password, fallbackMailbox)
						if err := fallbackClient.Connect(context.Background()); err == nil {
							if err := fallbackClient.SelectInbox(context.Background()); err == nil {
								foundUID, err = fallbackClient.FindUIDByMessageID(context.Background(), currentEmail.MessageID)
								if err == nil {
									imapClient.Close()
									defer func() { fallbackClient.Close() }()
									imapClient = fallbackClient
									log.Printf("[email-sync] found email in %s mailbox", fallbackMailbox)
								} else {
									fallbackClient.Close()
								}
							} else {
								fallbackClient.Close()
							}
						}
					}

					if foundUID > 0 {
						uidToMove = foundUID
						// Update database with the found UID and folder
						uidInt := int64(foundUID)
						currentFolder := imapClient.GetMailbox()
						db.ExecContext(context.Background(),
							"UPDATE emails SET imap_uid = $1, folder = $2 WHERE id = $3",
							uidInt, currentFolder, currentEmail.ID)
						log.Printf("[email-sync] backfilled IMAP UID %d for email %d (folder: %s)", foundUID, currentEmail.ID, currentFolder)
					} else {
						log.Printf("[email-sync] failed to find IMAP UID in any mailbox")
					}
				}

				if uidToMove > 0 {
					if isArchiving {
						// Make sure we're moving from the right mailbox
						if imapClient.GetMailbox() != "INBOX" {
							// Email is in Archive, not INBOX - no need to move, just update folder
							log.Printf("[email-sync] email already in Archive folder, updating folder in database")
							db.ExecContext(context.Background(),
								"UPDATE emails SET folder = 'Archive' WHERE id = $1",
								currentEmail.ID)
						} else {
							if err := imapClient.MoveToArchive(context.Background(), uidToMove); err != nil {
								log.Printf("[email-sync] failed to move to archive: %v", err)
							} else {
								log.Printf("[email-sync] successfully moved email UID %d to Archive", uidToMove)
								// Update folder in database to reflect the move
								db.ExecContext(context.Background(),
									"UPDATE emails SET folder = 'Archive' WHERE id = $1",
									currentEmail.ID)
							}
						}
					} else if isUnarchiving {
						// Make sure we're moving from the right mailbox
						if imapClient.GetMailbox() != "Archive" {
							// Email is in INBOX, not Archive - no need to move, just update folder
							log.Printf("[email-sync] email already in INBOX folder, updating folder in database")
							db.ExecContext(context.Background(),
								"UPDATE emails SET folder = 'INBOX' WHERE id = $1",
								currentEmail.ID)
						} else {
							if err := imapClient.MoveFromArchive(context.Background(), uidToMove); err != nil {
								log.Printf("[email-sync] failed to move from archive: %v", err)
							} else {
								log.Printf("[email-sync] successfully moved email UID %d from Archive to INBOX", uidToMove)
								// Update folder in database to reflect the move
								db.ExecContext(context.Background(),
									"UPDATE emails SET folder = 'INBOX' WHERE id = $1",
									currentEmail.ID)
							}
						}
					}
				} else {
					log.Printf("[email-sync] skipping IMAP move - no UID available for email %d", currentEmail.ID)
				}
			}
		}
	}

	email, err := emailService.UpdateEmailStatus(context.Background(), userID, emailID, params.Status)
	if err != nil {
		log.Printf("[email-sync] failed to update email status: %v", err)
		if err.Error() == "email not found" {
			http.Error(w, "Email not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "invalid status") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update email status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(email)
}

// ConvertEmailToCardRoute handles POST /api/emails/{id}/convert
// Converts an email to a card, optionally linking to an existing card
func (h *Handler) ConvertEmailToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var params models.ConvertEmailParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate title
	if params.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	db := h.GetDB()

	// Get the email to verify ownership
	_, err = emailService.GetEmailByID(context.Background(), userID, emailID)
	if err != nil {
		if err.Error() == "email not found" {
			http.Error(w, "Email not found", http.StatusNotFound)
			return
		}
		log.Printf("[email] failed to get email for conversion: %v", err)
		http.Error(w, "Failed to get email", http.StatusInternalServerError)
		return
	}

	var cardInternalID int
	var cardID string

	// If card_id provided, update existing card; otherwise create new
	if params.CardID != nil && *params.CardID != "" {
		// Get the card by string card_id to get internal ID
		partialCard, err := services.GetPartialCardByCardID(db, userID, *params.CardID)
		if err != nil {
			http.Error(w, "Card not found", http.StatusNotFound)
			return
		}

		// Update card with new content
		updateParams := models.EditCardParams{
			CardID: *params.CardID,
			Title:  params.Title,
		}
		if params.Body != nil {
			updateParams.Body = *params.Body
		}

		_, err = services.UpdateCard(db, userID, partialCard.ID, updateParams)
		if err != nil {
			log.Printf("[email] failed to update card: %v", err)
			http.Error(w, "Failed to update card", http.StatusInternalServerError)
			return
		}

		cardInternalID = partialCard.ID
		cardID = partialCard.CardID
	} else {
		// Create new card - generate next root card ID
		createParams := models.EditCardParams{
			CardID: h.getNextRootCardID(userID),
			Title:  params.Title,
		}
		if params.Body != nil {
			createParams.Body = *params.Body
		}

		card, err := services.CreateCard(db, userID, createParams)
		if err != nil {
			log.Printf("[email] failed to create card: %v", err)
			http.Error(w, "Failed to create card", http.StatusInternalServerError)
			return
		}

		cardInternalID = card.ID
		cardID = card.CardID
	}

	// Handle tags if provided
	if params.Tags != nil && *params.Tags != "" {
		// Parse comma-separated tags
		tagNames := strings.Split(*params.Tags, ",")
		// Trim whitespace from each tag
		for i := range tagNames {
			tagNames[i] = strings.TrimSpace(tagNames[i])
		}
		// Remove empty tags
		var cleanTags []string
		for _, tag := range tagNames {
			if tag != "" {
				cleanTags = append(cleanTags, tag)
			}
		}

		// Remove existing tags and add new ones
		_ = services.RemoveAllTagsFromCard(db, userID, cardInternalID)
		if len(cleanTags) > 0 {
			// Create tags and add to card
			for _, tagName := range cleanTags {
				tagParams := models.EditTagParams{
					Name:  tagName,
					Color: "black",
				}
				_, err := services.CreateTag(db, userID, tagParams)
				if err != nil {
					log.Printf("[email] failed to create tag %s: %v", tagName, err)
					continue
				}
				err = services.AddTagToCard(db, userID, tagName, cardInternalID)
				if err != nil {
					log.Printf("[email] failed to add tag %s to card: %v", tagName, err)
				}
			}
		}
	}

	// Create email_card_link record
	// First check if link already exists
	var existingLinkID int
	err = db.QueryRowContext(context.Background(),
		"SELECT id FROM email_card_links WHERE email_id = $1",
		emailID).Scan(&existingLinkID)

	if err == sql.ErrNoRows {
		// No existing link, create one
		_, err = db.ExecContext(context.Background(),
			"INSERT INTO email_card_links (email_id, card_id) VALUES ($1, $2)",
			emailID, cardInternalID)
		if err != nil {
			log.Printf("[email] failed to create email_card_link: %v", err)
			// Don't fail the request - card was created/updated successfully
		}
	} else if err != nil {
		log.Printf("[email] error checking email_card_link: %v", err)
		// Don't fail the request
	}

	// Update email's card_id
	_, err = db.ExecContext(context.Background(),
		"UPDATE emails SET card_id = $1 WHERE id = $2",
		cardInternalID, emailID)
	if err != nil {
		log.Printf("[email] failed to update email card_id: %v", err)
		// Don't fail the request
	}

	log.Printf("[email] converted email %d to card %s", emailID, cardID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": cardID,
	})
}

// BatchEmailParams represents parameters for batch email operations
type BatchEmailParams struct {
	EmailIDs []int `json:"email_ids"`
}

// BatchArchiveEmailsParams represents parameters for batch archiving emails
type BatchArchiveEmailsParams struct {
	EmailIDs []int  `json:"email_ids"`
	Status   string `json:"status"` // "archived" or "unprocessed"
}

// BatchConvertEmailsParams represents parameters for batch converting emails
type BatchConvertEmailsParams struct {
	EmailIDs []int  `json:"email_ids"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Tags     string `json:"tags"`
}

// BatchArchiveEmailsRoute handles POST /api/emails/batch-archive
// Archives or unarchives multiple emails at once
func (h *Handler) BatchArchiveEmailsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse request body
	var params BatchArchiveEmailsParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email IDs
	if len(params.EmailIDs) == 0 {
		http.Error(w, "email_ids is required", http.StatusBadRequest)
		return
	}

	// Default to archived if not specified
	status := params.Status
	if status == "" {
		status = "archived"
	}

	// Validate status
	if status != "archived" && status != "unprocessed" {
		http.Error(w, "status must be 'archived' or 'unprocessed'", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	db := h.GetDB()

	// Get emails before updating to handle IMAP folder movement
	emails, _, err := emailService.ListEmails(context.Background(), userID, models.EmailListFilters{
		Limit: func() *int { l := 10000; return &l }(),
	})
	if err != nil {
		log.Printf("[email] failed to get emails for batch archive: %v", err)
		http.Error(w, "Failed to get emails", http.StatusInternalServerError)
		return
	}

	// Build map of email ID to email
	emailMap := make(map[int]models.Email)
	for _, email := range emails {
		emailMap[email.ID] = email
	}

	// Handle IMAP folder movement for each email
	for _, emailID := range params.EmailIDs {
		email, exists := emailMap[emailID]
		if !exists {
			log.Printf("[email] email %d not found for user %d", emailID, userID)
			continue
		}

		isArchiving := status == "archived" && email.Status != "archived"
		isUnarchiving := status != "archived" && email.Status == "archived"

		if isArchiving || isUnarchiving {
			// Only move in IMAP if we have an email account
			if email.EmailAccountID != nil {
				// Get email account credentials
				var accountID int
				var emailAddress string
				var encryptedPassword string
				var imapServer string

				err := db.QueryRowContext(context.Background(), `
					SELECT id, email_address, app_password_encrypted, imap_server
					FROM email_accounts
					WHERE id = $1 AND user_id = $2
				`, *email.EmailAccountID, userID).Scan(&accountID, &emailAddress, &encryptedPassword, &imapServer)
				if err != nil {
					log.Printf("[email] failed to get email account for batch archive: %v", err)
					continue
				}

				// Decrypt password
				password, err := services.DecryptAppPassword(encryptedPassword, "")
				if err != nil {
					log.Printf("[email] failed to decrypt password for batch archive: %v", err)
					continue
				}

				// Create a client with the specific mailbox based on operation
				var imapClient *services.IMAPClient
				if isUnarchiving {
					imapClient = services.NewIMAPClientWithMailbox(imapServer, emailAddress, password, "Archive")
				} else {
					imapClient = services.NewIMAPClientWithMailbox(imapServer, emailAddress, password, "INBOX")
				}

				if err := imapClient.Connect(context.Background()); err != nil {
					log.Printf("[email] failed to connect to IMAP for batch archive: %v", err)
					continue
				} else if err := imapClient.SelectInbox(context.Background()); err != nil {
					log.Printf("[email] failed to select mailbox for batch archive: %v", err)
					imapClient.Close()
				} else {
					defer imapClient.Close()

					// Get UID if missing
					uidToMove := uint32(0)
					if email.IMAPUID != nil {
						uidToMove = uint32(*email.IMAPUID)
					} else {
						// Backfill: find UID by Message-ID
						foundUID, findErr := imapClient.FindUIDByMessageID(context.Background(), email.MessageID)
						if findErr == nil && foundUID > 0 {
							uidToMove = foundUID
							uidInt := int64(foundUID)
							currentFolder := imapClient.GetMailbox()
							db.ExecContext(context.Background(),
								"UPDATE emails SET imap_uid = $1, folder = $2 WHERE id = $3",
								uidInt, currentFolder, email.ID)
						}
					}

					if uidToMove > 0 {
						if isArchiving && imapClient.GetMailbox() == "INBOX" {
							if moveErr := imapClient.MoveToArchive(context.Background(), uidToMove); moveErr != nil {
								log.Printf("[email] failed to move to archive in batch: %v", moveErr)
							} else {
								db.ExecContext(context.Background(),
									"UPDATE emails SET folder = 'Archive' WHERE id = $1", email.ID)
							}
						} else if isUnarchiving && imapClient.GetMailbox() == "Archive" {
							if moveErr := imapClient.MoveFromArchive(context.Background(), uidToMove); moveErr != nil {
								log.Printf("[email] failed to move from archive in batch: %v", moveErr)
							} else {
								db.ExecContext(context.Background(),
									"UPDATE emails SET folder = 'INBOX' WHERE id = $1", email.ID)
							}
						}
					}
				}
			}
		}
	}

	// Batch update status
	updatedEmails, err := emailService.BatchUpdateEmailStatus(context.Background(), userID, params.EmailIDs, status)
	if err != nil {
		log.Printf("[email] failed to batch update email status: %v", err)
		http.Error(w, "Failed to batch update emails", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(updatedEmails),
		"emails":  updatedEmails,
	})
}

// BatchConvertEmailsRoute handles POST /api/emails/batch-convert
// Converts multiple emails to cards at once
func (h *Handler) BatchConvertEmailsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse request body
	var params BatchConvertEmailsParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email IDs
	if len(params.EmailIDs) == 0 {
		http.Error(w, "email_ids is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	db := h.GetDB().(*sql.DB)

	var tagsPtr *string
	if params.Tags != "" {
		tagsPtr = &params.Tags
	}

	results, err := emailService.BatchConvertEmailsToCards(context.Background(), db, userID, params.EmailIDs, params.Title, params.Body, tagsPtr)
	if err != nil {
		log.Printf("[email] failed to batch convert emails: %v", err)
		http.Error(w, "Failed to batch convert emails", http.StatusInternalServerError)
		return
	}

	// Count successes and failures
	successCount := 0
	failCount := 0
	for _, result := range results {
		if success, ok := result["success"].(bool); ok && success {
			successCount++
		} else {
			failCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"total":         len(params.EmailIDs),
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       results,
	})
}

// BatchCreateTasksRoute handles POST /api/emails/batch-create-tasks
// Creates tasks from multiple emails at once
func (h *Handler) BatchCreateTasksRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse request body
	var params BatchEmailParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email IDs
	if len(params.EmailIDs) == 0 {
		http.Error(w, "email_ids is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	db := h.GetDB().(*sql.DB)

	results, err := emailService.BatchCreateTasksFromEmails(context.Background(), db, userID, params.EmailIDs)
	if err != nil {
		log.Printf("[email] failed to batch create tasks: %v", err)
		http.Error(w, "Failed to batch create tasks", http.StatusInternalServerError)
		return
	}

	// Count successes and failures
	successCount := 0
	failCount := 0
	for _, result := range results {
		if success, ok := result["success"].(bool); ok && success {
			successCount++
		} else {
			failCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"total":         len(params.EmailIDs),
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       results,
	})
}

// EmailSearchParams represents parameters for email search
type EmailSearchParams struct {
	SearchTerm string `json:"search_term"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
}

// EmailSearchResponse represents the response from email search
type EmailSearchResponse struct {
	Results    []models.EmailSearchResult `json:"results"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"per_page"`
	Total      int                        `json:"total"`
	TotalPages int                        `json:"total_pages"`
}

// SearchEmailsRoute handles GET /api/emails/search
// Searches emails using SQL-based full-text search across subject, from_address, and body_text
func (h *Handler) SearchEmailsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Search query 'q' is required", http.StatusBadRequest)
		return
	}

	// Set default pagination values
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if p, err := strconv.Atoi(perPageStr); err == nil && p > 0 && p <= 100 {
			perPage = p
		}
	}

	offset := (page - 1) * perPage

	// Build SQL search query with ILIKE for case-insensitive search
	// Search across subject, from_address, and body_text
	query := `
		SELECT id, subject, from_address, from_name, body_text
		FROM emails
		WHERE user_id = $1
			AND (subject ILIKE $2 OR from_address ILIKE $2 OR body_text ILIKE $2)
		ORDER BY received_at DESC NULLS LAST, created_at DESC
		LIMIT $3 OFFSET $4
	`

	// Add wildcards to search term for partial matching
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	rows, err := h.GetDB().QueryContext(context.Background(), query, userID, searchPattern, perPage, offset)
	if err != nil {
		log.Printf("[email] SQL search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*)
		FROM emails
		WHERE user_id = $1
			AND (subject ILIKE $2 OR from_address ILIKE $2 OR body_text ILIKE $2)
	`
	var total int
	err = h.GetDB().QueryRowContext(context.Background(), countQuery, userID, searchPattern).Scan(&total)
	if err != nil {
		log.Printf("[email] failed to get search count: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Process results
	var results []models.EmailSearchResult
	for rows.Next() {
		var id int
		var subject sql.NullString
		var fromAddress sql.NullString
		var fromName sql.NullString
		var bodyText sql.NullString

		err := rows.Scan(&id, &subject, &fromAddress, &fromName, &bodyText)
		if err != nil {
			log.Printf("[email] failed to scan search result: %v", err)
			continue
		}

		// Build sender display name
		sender := ""
		if fromName.Valid && fromName.String != "" {
			sender = fromName.String
			if fromAddress.Valid && fromAddress.String != "" {
				sender += " <" + fromAddress.String + ">"
			}
		} else if fromAddress.Valid {
			sender = fromAddress.String
		}

		// Build preview (first 200 chars of body)
		preview := ""
		if bodyText.Valid && bodyText.String != "" {
			preview = bodyText.String
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
		}

		result := models.EmailSearchResult{
			ID:      id,
			Subject: subject.String,
			Sender:  sender,
			Preview: preview,
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[email] error iterating search results: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	totalPages := (total + perPage - 1) / perPage

	response := EmailSearchResponse{
		Results:    results,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmailThreadRoute handles GET /api/emails/threads/{thread_id}
// Retrieves all emails in a thread
func (h *Handler) GetEmailThreadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get thread_id from path
	threadID := mux.Vars(r)["thread_id"]
	if threadID == "" {
		http.Error(w, "thread_id is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())

	thread, err := emailService.GetEmailThreadByID(context.Background(), userID, threadID)
	if err != nil {
		if err.Error() == "thread not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to get email thread: %v", err)
		http.Error(w, "Failed to get email thread", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thread)
}

// MarkThreadAsReadRoute handles PATCH /api/emails/threads/{thread_id}/read
// Marks all emails in a thread as read
func (h *Handler) MarkThreadAsReadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get thread_id from path
	threadID := mux.Vars(r)["thread_id"]
	if threadID == "" {
		http.Error(w, "thread_id is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())

	err := emailService.MarkThreadAsRead(context.Background(), userID, threadID)
	if err != nil {
		if err.Error() == "thread not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to mark thread as read: %v", err)
		http.Error(w, "Failed to mark thread as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ArchiveThreadRoute handles PATCH /api/emails/threads/{thread_id}/archive
// Archives all emails in a thread
func (h *Handler) ArchiveThreadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get thread_id from path
	threadID := mux.Vars(r)["thread_id"]
	if threadID == "" {
		http.Error(w, "thread_id is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())

	err := emailService.ArchiveThread(context.Background(), userID, threadID)
	if err != nil {
		if err.Error() == "thread not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[email-sync] failed to archive thread: %v", err)
		http.Error(w, "Failed to archive thread", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ExtractFactsFromEmailRoute handles POST /api/emails/{id}/extract-facts
// Extracts factual statements from an email using AI (PRO feature)
func (h *Handler) ExtractFactsFromEmailRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Check if user is PRO (has active or trialing subscription)
	user, err := h.QueryUser(userID)
	if err != nil {
		log.Printf("[email] failed to query user for fact extraction: %v", err)
		http.Error(w, "Failed to verify subscription", http.StatusInternalServerError)
		return
	}

	isProUser := user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing"
	if !isProUser {
		log.Printf("[email] fact extraction rejected for non-PRO user %d", userID)
		http.Error(w, "Fact extraction is a PRO feature. Please upgrade your subscription.", http.StatusForbidden)
		return
	}

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())

	// Extract facts from email
	facts, err := emailService.ExtractFactsFromEmail(context.Background(), userID, emailID)
	if err != nil {
		log.Printf("[email] failed to extract facts from email %d: %v", emailID, err)
		http.Error(w, "Failed to extract facts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email_id": emailID,
		"facts":    facts,
		"count":    len(facts),
	})
}

// SaveFactsFromEmailRoute handles POST /api/emails/{id}/save-facts
// Saves extracted facts from an email (PRO feature)
func (h *Handler) SaveFactsFromEmailRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Check if user is PRO (has active or trialing subscription)
	user, err := h.QueryUser(userID)
	if err != nil {
		log.Printf("[email] failed to query user for fact saving: %v", err)
		http.Error(w, "Failed to verify subscription", http.StatusInternalServerError)
		return
	}

	isProUser := user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing"
	if !isProUser {
		log.Printf("[email] fact saving rejected for non-PRO user %d", userID)
		http.Error(w, "Fact saving is a PRO feature. Please upgrade your subscription.", http.StatusForbidden)
		return
	}

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Parse request body with facts
	var req struct {
		Facts []string `json:"facts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Facts) == 0 {
		http.Error(w, "No facts provided", http.StatusBadRequest)
		return
	}

	db := h.GetDB().(*sql.DB)

	// Save facts and link them to the email
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[email] failed to begin transaction: %v", err)
		http.Error(w, "Failed to save facts", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	savedFacts := make([]models.Fact, 0, len(req.Facts))

	for _, factText := range req.Facts {
		if strings.TrimSpace(factText) == "" {
			continue
		}

		// Create a temporary card for this fact (facts must be linked to a card)
		// Use the email subject as the card title
		var subject string
		err = tx.QueryRow(`SELECT subject FROM emails WHERE id = $1 AND user_id = $2`, emailID, userID).Scan(&subject)
		if err != nil {
			log.Printf("[email] failed to get email subject: %v", err)
			subject = "Email Facts"
		}

		// Create or find a card for this email's facts
		cardTitle := fmt.Sprintf("Facts from: %s", subject)
		cardBody := fmt.Sprintf("Source: Email ID %d\n\nFacts:\n", emailID)

		var cardID int
		err = tx.QueryRow(`
			INSERT INTO cards (user_id, card_id, title, body)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, card_id) DO UPDATE SET title = EXCLUDED.title, body = cards.body || $5, updated_at = NOW()
			RETURNING id
		`, userID, fmt.Sprintf("email-facts-%d", emailID), cardTitle, cardBody, "\n"+factText).Scan(&cardID)

		if err != nil {
			log.Printf("[email] failed to create/find card for facts: %v", err)
			// Try without ON CONFLICT for older databases
			err = tx.QueryRow(`
				INSERT INTO cards (user_id, card_id, title, body)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, userID, fmt.Sprintf("email-facts-%d", emailID), cardTitle, cardBody).Scan(&cardID)

			if err != nil {
				log.Printf("[email] failed to create card for facts: %v", err)
				continue
			}
		}

		// Create the fact
		var factID int
		err = tx.QueryRow(`
			INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			RETURNING id
		`, userID, cardID, factText).Scan(&factID)

		if err != nil {
			log.Printf("[email] failed to create fact: %v", err)
			continue
		}

		// Link fact to card
		_, err = tx.Exec(`
			INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
			VALUES ($1, $2, $3, TRUE, NOW(), NOW())
			ON CONFLICT (fact_id, card_pk) DO UPDATE SET updated_at = NOW()
		`, factID, cardID, userID)

		if err != nil {
			log.Printf("[email] failed to link fact to card: %v", err)
			continue
		}

		// Link fact to email
		_, err = tx.Exec(`
			INSERT INTO email_fact_junction (user_id, email_id, fact_id, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			ON CONFLICT (user_id, email_id, fact_id) DO UPDATE SET updated_at = NOW()
		`, userID, emailID, factID)

		if err != nil {
			log.Printf("[email] failed to link fact to email: %v", err)
			continue
		}

		savedFacts = append(savedFacts, models.Fact{
			ID:     factID,
			UserID: userID,
			CardPK: cardID,
			Fact:   factText,
		})
	}

	if err = tx.Commit(); err != nil {
		log.Printf("[email] failed to commit transaction: %v", err)
		http.Error(w, "Failed to save facts", http.StatusInternalServerError)
		return
	}

	log.Printf("[email] saved %d facts from email %d for user %d", len(savedFacts), emailID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"email_id":  emailID,
		"saved_count": len(savedFacts),
		"facts":     savedFacts,
	})
}

// GetEmailFactsRoute handles GET /api/emails/{id}/facts
// Retrieves all facts extracted from an email (PRO feature)
func (h *Handler) GetEmailFactsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Check if user is PRO (has active or trialing subscription)
	user, err := h.QueryUser(userID)
	if err != nil {
		log.Printf("[email] failed to query user for email facts: %v", err)
		http.Error(w, "Failed to verify subscription", http.StatusInternalServerError)
		return
	}

	isProUser := user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing"
	if !isProUser {
		log.Printf("[email] email facts retrieval rejected for non-PRO user %d", userID)
		http.Error(w, "Email facts are a PRO feature. Please upgrade your subscription.", http.StatusForbidden)
		return
	}

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	db := h.GetDB()

	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.card_pk, f.fact, f.created_at, f.updated_at,
		       efj.created_at as linked_at
		FROM facts f
		JOIN email_fact_junction efj ON f.id = efj.fact_id
		WHERE efj.user_id = $1 AND efj.email_id = $2
		ORDER BY efj.created_at DESC
	`, userID, emailID)

	if err != nil {
		log.Printf("[email] failed to get email facts: %v", err)
		http.Error(w, "Failed to get facts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		var linkedAt time.Time
		if err := rows.Scan(&f.ID, &f.UserID, &f.CardPK, &f.Fact, &f.CreatedAt, &f.UpdatedAt, &linkedAt); err != nil {
			log.Printf("[email] error scanning email fact: %v", err)
			continue
		}
		facts = append(facts, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email_id": emailID,
		"facts":    facts,
		"count":    len(facts),
	})
}

// GetEmailAttachmentsRoute handles GET /api/emails/{id}/attachments
// Returns all attachments for an email
func (h *Handler) GetEmailAttachmentsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get email ID from path
	idStr := mux.Vars(r)["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Verify email exists and belongs to user
	emailService := services.NewEmailService(h.GetDB())
	_, err = emailService.GetEmailByID(context.Background(), userID, emailID)
	if err != nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Get attachments
	attachmentService := services.NewEmailAttachmentService(h.GetDB(), &S3AttachmentHandler{h: h}, userID)
	attachments, err := attachmentService.GetAttachmentsByEmailID(context.Background(), userID, emailID)
	if err != nil {
		log.Printf("[email] failed to get attachments: %v", err)
		http.Error(w, "Failed to get attachments", http.StatusInternalServerError)
		return
	}

	// Add download URLs
	result := make([]models.EmailAttachmentWithDownloadURL, 0, len(attachments))
	for _, att := range attachments {
		withURL := models.EmailAttachmentWithDownloadURL{
			EmailAttachment: att,
			DownloadURL:     fmt.Sprintf("/api/emails/attachments/%d/download", att.ID),
			IsImage:         isImageContentType(att.ContentType),
			IsSavedToVault:  att.FileID != nil,
		}
		if att.ThumbnailPath != nil {
			withURL.ThumbnailURL = fmt.Sprintf("/api/emails/attachments/%d/thumbnail", att.ID)
		}
		result = append(result, withURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"attachments": result,
		"count":       len(result),
	})
}

// DownloadEmailAttachmentRoute handles GET /api/emails/attachments/{id}/download
// Downloads an attachment file
func (h *Handler) DownloadEmailAttachmentRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get attachment ID from path
	idStr := mux.Vars(r)["id"]
	attachmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}

	// Get attachment
	attachmentService := services.NewEmailAttachmentService(h.GetDB(), &S3AttachmentHandler{h: h}, userID)
	attachment, err := attachmentService.GetAttachmentByID(context.Background(), userID, attachmentID)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	if attachment.S3Key == nil {
		http.Error(w, "Attachment file not available", http.StatusNotFound)
		return
	}

	// Stream file from S3
	s3Output, err := h.downloadObject(h.Server.S3, *attachment.S3Key, "")
	if err != nil {
		log.Printf("[email] failed to download attachment from S3: %v", err)
		http.Error(w, "Failed to download attachment", http.StatusInternalServerError)
		return
	}
	defer s3Output.Body.Close()

	// Set headers
	contentType := "application/octet-stream"
	if attachment.ContentType != nil {
		contentType = *attachment.ContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))

	// Copy content to response
	if _, err := io.Copy(w, s3Output.Body); err != nil {
		log.Printf("[email] failed to stream attachment: %v", err)
	}
}

// GetEmailAttachmentThumbnailRoute handles GET /api/emails/attachments/{id}/thumbnail
// Returns the thumbnail for an image attachment
func (h *Handler) GetEmailAttachmentThumbnailRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get attachment ID from path
	idStr := mux.Vars(r)["id"]
	attachmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}

	// Get attachment
	attachmentService := services.NewEmailAttachmentService(h.GetDB(), &S3AttachmentHandler{h: h}, userID)
	attachment, err := attachmentService.GetAttachmentByID(context.Background(), userID, attachmentID)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	if attachment.ThumbnailPath == nil || *attachment.ThumbnailPath == "" {
		http.Error(w, "Thumbnail not available", http.StatusNotFound)
		return
	}

	// Stream thumbnail from S3
	s3Output, err := h.downloadObject(h.Server.S3, *attachment.ThumbnailPath, "")
	if err != nil {
		log.Printf("[email] failed to download thumbnail from S3: %v", err)
		http.Error(w, "Failed to download thumbnail", http.StatusInternalServerError)
		return
	}
	defer s3Output.Body.Close()

	// Set headers
	w.Header().Set("Content-Type", "image/jpeg")

	// Copy content to response
	if _, err := io.Copy(w, s3Output.Body); err != nil {
		log.Printf("[email] failed to stream thumbnail: %v", err)
	}
}

// SaveEmailAttachmentToVaultRoute handles POST /api/emails/attachments/{id}/save-to-vault
// Saves an attachment to the file vault
func (h *Handler) SaveEmailAttachmentToVaultRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get attachment ID from path
	idStr := mux.Vars(r)["id"]
	attachmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var params models.SaveAttachmentToVaultParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil && err != io.EOF {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Save to file vault
	attachmentService := services.NewEmailAttachmentService(h.GetDB(), &S3AttachmentHandler{h: h}, userID)
	updatedAttachment, err := attachmentService.SaveToFileVault(context.Background(), userID, attachmentID, params.CardPK)
	if err != nil {
		log.Printf("[email] failed to save attachment to vault: %v", err)
		http.Error(w, "Failed to save attachment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAttachment)
}

// DeleteEmailAttachmentRoute handles DELETE /api/emails/attachments/{id}
// Deletes an attachment
func (h *Handler) DeleteEmailAttachmentRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get attachment ID from path
	idStr := mux.Vars(r)["id"]
	attachmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}

	// Delete attachment
	attachmentService := services.NewEmailAttachmentService(h.GetDB(), &S3AttachmentHandler{h: h}, userID)
	err = attachmentService.DeleteAttachment(context.Background(), userID, attachmentID)
	if err != nil {
		log.Printf("[email] failed to delete attachment: %v", err)
		http.Error(w, "Failed to delete attachment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// S3AttachmentHandler wraps Handler to implement S3StorageService interface
type S3AttachmentHandler struct {
	h *Handler
}

// UploadAttachment implements S3StorageService
func (s *S3AttachmentHandler) UploadAttachment(key string, data []byte, contentType string) (string, error) {
	// Create temp file for upload
	tempFile, err := os.CreateTemp("/tmp", "email-att-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		return "", err
	}

	// Seek back to start
	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", err
	}

	// Upload to S3
	s.h.uploadObject(s.h.Server.S3, key, tempFile.Name())
	return key, nil
}

// GenerateThumbnail implements S3StorageService
func (s *S3AttachmentHandler) GenerateThumbnail(data []byte, contentType string) ([]byte, error) {
	// Check if this is an image
	if !isImageContentTypeString(contentType) {
		return nil, fmt.Errorf("not an image")
	}

	// Create temp file
	tempFile, err := os.CreateTemp("/tmp", "email-thumb-*.tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		return nil, err
	}

	// Generate thumbnail
	thumbnailTempPath := tempFile.Name() + ".thumb"
	defer os.Remove(thumbnailTempPath)

	err = s.h.generateThumbnail(tempFile.Name(), thumbnailTempPath)
	if err != nil {
		return nil, err
	}

	// Read thumbnail
	thumbnailData, err := os.ReadFile(thumbnailTempPath)
	if err != nil {
		return nil, err
	}

	return thumbnailData, nil
}

// isImageContentType checks if content type is an image
func isImageContentType(contentType *string) bool {
	if contentType == nil {
		return false
	}
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml"}
	for _, t := range imageTypes {
		if *contentType == t {
			return true
		}
	}
	return false
}

// isImageContentTypeString checks if content type is an image (string version)
func isImageContentTypeString(contentType string) bool {
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml"}
	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}
