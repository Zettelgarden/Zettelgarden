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

	// Fetch recent emails
	emails, maxUID, err := imapClient.FetchRecentEmails(context.Background(), 50)
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
		archiveMailEmails, _, err := imapClient.FetchRecentEmails(context.Background(), 100)
		if err != nil {
			log.Printf("[email-sync] failed to fetch from %s: %v", archiveFolder, err)
		} else {
			for _, email := range archiveMailEmails {
				email.UserID = userID
				email.EmailAccountID = &accountID
				if _, err := emailService.CreateEmail(context.Background(), email); err != nil {
					log.Printf("[email-sync] warning: failed to store archive email %s: %v", email.MessageID, err)
				} else {
					totalEmails++
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
