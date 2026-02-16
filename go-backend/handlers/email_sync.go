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
		"message":       "Sync completed successfully",
		"account_id":    accountID,
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

	if isReadStr := r.URL.Query().Get("is_read"); isReadStr != "" {
		if isRead, err := strconv.ParseBool(isReadStr); err == nil {
			filters.IsRead = &isRead
		}
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
