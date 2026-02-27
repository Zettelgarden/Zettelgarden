package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/models"
	"go-backend/services"
)

// EmailSyncJob fetches emails from active email accounts via IMAP
type EmailSyncJob struct {
	db                *sql.DB
	schedule          string
	s3Uploader        S3Uploader
}

// S3Uploader interface for uploading files to S3
type S3Uploader interface {
	UploadObject(key string, filePath string) error
}

// NewEmailSyncJob creates a new email sync job
func NewEmailSyncJob(db *sql.DB) *EmailSyncJob {
	return &EmailSyncJob{
		db:       db,
		schedule: "0 */5 * * * *", // Every 5 minutes
	}
}

// SetS3Uploader sets the S3 uploader for attachment processing
func (j *EmailSyncJob) SetS3Uploader(uploader S3Uploader) {
	j.s3Uploader = uploader
}

// Name returns the unique identifier for this job
func (j *EmailSyncJob) Name() string {
	return "email-sync"
}

// Schedule returns the cron expression for when this job should run
// Note: Using 6-field cron expression (with seconds) as required by the scheduler
func (j *EmailSyncJob) Schedule() string {
	return j.schedule
}

// MaxRetries returns the number of times to retry on failure
func (j *EmailSyncJob) MaxRetries() int {
	return 3
}

// NextRun returns the next scheduled run time for this job
func (j *EmailSyncJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// emailAccount represents an email account to sync
type emailAccount struct {
	ID              int
	UserID          int
	EmailAddress    string
	IMAPServer      string
	AppPasswordEncrypted string
	IMAPUID         *int
	IMAPUIDValidity *int
}

// Handler executes the email sync job logic
func (j *EmailSyncJob) Handler(ctx context.Context) error {
	log.Println("[email-sync] starting email sync job")

	if j.db == nil {
		log.Println("[email-sync] no database configured, skipping")
		return nil
	}

	// Query active email accounts
	rows, err := j.db.QueryContext(ctx, `
		SELECT id, user_id, email_address, imap_server, app_password_encrypted, imap_uid, imap_uid_validity
		FROM email_accounts
		WHERE is_active = true AND sync_status = 'active'
	`)
	if err != nil {
		log.Printf("[email-sync] failed to fetch email accounts: %v", err)
		return err
	}
	defer rows.Close()

	var accountsToSync []emailAccount
	for rows.Next() {
		var account emailAccount
		var imapUID sql.NullInt64
		var imapUIDValidity sql.NullInt64

		err := rows.Scan(&account.ID, &account.UserID, &account.EmailAddress, &account.IMAPServer, &account.AppPasswordEncrypted, &imapUID, &imapUIDValidity)
		if err != nil {
			log.Printf("[email-sync] failed to scan account row: %v", err)
			continue
		}

		if imapUID.Valid {
			uid := int(imapUID.Int64)
			account.IMAPUID = &uid
		}
		if imapUIDValidity.Valid {
			validity := int(imapUIDValidity.Int64)
			account.IMAPUIDValidity = &validity
		}

		accountsToSync = append(accountsToSync, account)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[email-sync] error iterating accounts: %v", err)
		return err
	}

	// Sync each account
	totalEmails := 0
	totalAccounts := len(accountsToSync)
	for _, account := range accountsToSync {
		emailCount, err := j.syncAccount(ctx, account)
		if err != nil {
			log.Printf("[email-sync] failed to sync account %d: %v", account.ID, err)
			continue
		}
		totalEmails += emailCount
	}

	log.Printf("[email-sync] completed, synced %d emails from %d accounts", totalEmails, totalAccounts)
	return nil
}

// syncAccount handles syncing a single account
func (j *EmailSyncJob) syncAccount(ctx context.Context, account emailAccount) (int, error) {
	// Decrypt the app password
	password, err := services.DecryptAppPassword(account.AppPasswordEncrypted, "")
	if err != nil {
		log.Printf("[email-sync] failed to decrypt password for account %d: %v", account.ID, err)
		return 0, err
	}

	// Create IMAP client
	client := services.NewIMAPClient(account.IMAPServer, account.EmailAddress, password)

	// Connect to IMAP
	if err := client.Connect(ctx); err != nil {
		log.Printf("[email-sync] failed to connect to IMAP for account %d: %v", account.ID, err)
		// Update sync status to error
		j.updateSyncStatus(ctx, account.ID, account.UserID, "error")
		return 0, err
	}
	defer client.Close()

	// Create EmailService
	emailService := services.NewEmailService(j.db)
	accountService := services.NewEmailAccountService(j.db)

	totalEmails := 0

	// Discover archive folder name
	archiveFolder, err := client.FindArchiveMailbox(ctx)
	if err != nil {
		log.Printf("[email-sync] could not find archive folder for account %d: %v", account.ID, err)
		archiveFolder = "Archive" // fallback to default
	}

	// Sync from INBOX
	inboxEmails, err := j.syncFolder(ctx, account, client, emailService, "INBOX")
	if err != nil {
		log.Printf("[email-sync] failed to sync INBOX for account %d: %v", account.ID, err)
		// Don't fail entire sync if one folder fails
	} else {
		totalEmails += inboxEmails
	}

	// Sync from Archive folder
	archiveEmails, err := j.syncFolder(ctx, account, client, emailService, archiveFolder)
	if err != nil {
		log.Printf("[email-sync] failed to sync %s for account %d: %v", archiveFolder, account.ID, err)
		// Don't fail entire sync if one folder fails
	} else {
		totalEmails += archiveEmails
	}

	// Reconcile: detect external changes (emails moved by user in email client)
	if err := j.reconcileFolders(ctx, account, client, emailService, archiveFolder); err != nil {
		log.Printf("[email-sync] failed to reconcile folders for account %d: %v", account.ID, err)
		// Don't fail entire sync if reconciliation fails
	}

	// Update last_sync_at
	now := time.Now()
	if err := accountService.UpdateLastSync(ctx, account.UserID, account.ID, now); err != nil {
		log.Printf("[email-sync] failed to update last sync for account %d: %v", account.ID, err)
	}

	// Update sync status to active
	if err := j.updateSyncStatus(ctx, account.ID, account.UserID, "active"); err != nil {
		log.Printf("[email-sync] failed to update sync status for account %d: %v", account.ID, err)
	}

	return totalEmails, nil
}

// syncFolder syncs emails from a specific folder (INBOX or Archive)
func (j *EmailSyncJob) syncFolder(ctx context.Context, account emailAccount, client *services.IMAPClient, emailService *services.EmailService, folder string) (int, error) {
	// Select the folder
	if err := client.SelectMailbox(ctx, folder); err != nil {
		return 0, fmt.Errorf("failed to select %s: %w", folder, err)
	}

	// Check for UIDVALIDITY changes
	currentUIDValidity := client.GetUIDValidity()
	if account.IMAPUIDValidity != nil && *account.IMAPUIDValidity != int(currentUIDValidity) {
		log.Printf("[email-sync] UIDVALIDITY changed for account %d in %s (old: %d, new: %d), doing full sync",
			account.ID, folder, *account.IMAPUIDValidity, currentUIDValidity)
		// Reset IMAP UID to force full sync for this folder
	}

	// Fetch emails with attachments (using UID if available, otherwise initial fetch)
	var emails []services.EmailWithAttachments
	var err error

	if account.IMAPUID != nil && *account.IMAPUID > 0 && folder == "INBOX" {
		// Incremental sync only for INBOX (Archive doesn't need incremental sync for new emails)
		emails, _, err = client.FetchEmailsSinceUIDWithAttachments(ctx, uint32(*account.IMAPUID))
	} else {
		// Initial fetch - get recent emails from this folder
		emails, _, err = client.FetchRecentEmailsWithAttachments(ctx, 100)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to fetch emails: %w", err)
	}

	// Create S3 attachment handler if uploader is available
	var attachmentService *services.EmailAttachmentService
	if j.s3Uploader != nil {
		s3Handler := &jobS3Handler{uploader: j.s3Uploader}
		attachmentService = services.NewEmailAttachmentService(j.db, s3Handler, account.UserID)
	}

	// Store emails and process attachments
	for _, emailWithAtt := range emails {
		email := emailWithAtt.Email
		email.UserID = account.UserID
		email.EmailAccountID = &account.ID

		storedEmail, err := emailService.CreateEmail(ctx, email)
		if err != nil {
			log.Printf("[email-sync] failed to store email %s for account %d: %v", email.MessageID, account.ID, err)
			continue
		}

		// Process attachments if S3 uploader is available
		if attachmentService != nil && len(emailWithAtt.Attachments) > 0 {
			for _, att := range emailWithAtt.Attachments {
				// Skip inline attachments (embedded images in HTML)
				if att.IsInline {
					continue
				}

				// Upload attachment to S3 and create email_attachment record
				createdAtt, err := attachmentService.CreateAttachmentWithData(
					ctx,
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
				_, err = attachmentService.SaveToFileVault(ctx, account.UserID, createdAtt.ID, nil)
				if err != nil {
					log.Printf("[email-sync] warning: failed to save attachment %s to vault: %v", att.Filename, err)
				} else {
					log.Printf("[email-sync] automatically saved attachment %s to file vault", att.Filename)
				}
			}
		}
	}

	log.Printf("[email-sync] synced %d emails from %s for account %d", len(emails), folder, account.ID)

	return len(emails), nil
}

// jobS3Handler implements S3StorageService for the job using S3Uploader
type jobS3Handler struct {
	uploader S3Uploader
}

func (j *jobS3Handler) UploadAttachment(key string, data []byte, contentType string) (string, error) {
	// Create temp file
	tempFile, err := os.CreateTemp("/tmp", "email-att-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		return "", err
	}
	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", err
	}

	// Upload using the uploader
	err = j.uploader.UploadObject(key, tempFile.Name())
	if err != nil {
		return "", err
	}
	return key, nil
}

func (j *jobS3Handler) GenerateThumbnail(data []byte, contentType string) ([]byte, error) {
	return nil, fmt.Errorf("thumbnail generation not supported in job")
}

// reconcileFolders detects and reconciles external changes (emails moved by user in email client)
func (j *EmailSyncJob) reconcileFolders(ctx context.Context, account emailAccount, client *services.IMAPClient, emailService *services.EmailService, archiveFolder string) error {
	// Get all emails for this account from database
	dbEmails, _, err := emailService.ListEmails(ctx, account.UserID, models.EmailListFilters{
		Limit: func() *int { l := 10000; return &l }(),
	})
	if err != nil {
		return fmt.Errorf("failed to get emails from database: %w", err)
	}

	// Build a map of normalized Message-ID to email in database
	dbEmailMap := make(map[string]models.Email)
	for _, email := range dbEmails {
		// Normalize Message-ID for consistent comparison
		normalized := services.NormalizeMessageID(email.MessageID)
		dbEmailMap[normalized] = email
	}

	log.Printf("[email-sync] reconciling %d emails from database for account %d (archive folder: %s)", len(dbEmailMap), account.ID, archiveFolder)

	// Check INBOX
	if err := client.SelectMailbox(ctx, "INBOX"); err != nil {
		log.Printf("[email-sync] failed to select INBOX for reconciliation: %v", err)
		// Continue to Archive
	} else {
		inboxMessageIDs, err := client.GetAllMessageUIDs(ctx)
		if err != nil {
			log.Printf("[email-sync] failed to get INBOX Message-IDs: %v", err)
		} else {
			log.Printf("[email-sync] found %d messages in INBOX for reconciliation", len(inboxMessageIDs))

			// Reconcile INBOX
			for normalizedID, dbEmail := range dbEmailMap {
				if dbEmail.EmailAccountID == nil || *dbEmail.EmailAccountID != account.ID {
					continue
				}

				inInbox := inboxMessageIDs[normalizedID]

				// Email is in INBOX on IMAP but marked as archived in DB
				if inInbox && dbEmail.Status == "archived" {
					log.Printf("[email-sync] reconciliation: email %s (normalized: %s) found in INBOX on IMAP, updating status from 'archived' to 'unprocessed'", dbEmail.MessageID, normalizedID)
					status := "unprocessed"
					if err := emailService.UpdateEmailFolder(ctx, account.UserID, dbEmail.MessageID, "INBOX", &status); err != nil {
						log.Printf("[email-sync] failed to update email %s: %v", dbEmail.MessageID, err)
					}
				}

				// Email is in INBOX on IMAP but folder in DB is not INBOX
				if inInbox && (dbEmail.Folder == nil || *dbEmail.Folder != "INBOX") {
					log.Printf("[email-sync] reconciliation: email %s found in INBOX on IMAP, updating folder", dbEmail.MessageID)
					if err := emailService.UpdateEmailFolder(ctx, account.UserID, dbEmail.MessageID, "INBOX", nil); err != nil {
						log.Printf("[email-sync] failed to update email folder %s: %v", dbEmail.MessageID, err)
					}
				}
			}
		}
	}

	// Check Archive
	if err := client.SelectMailbox(ctx, archiveFolder); err != nil {
		log.Printf("[email-sync] failed to select %s for reconciliation: %v", archiveFolder, err)
		return nil
	}

	archiveMessageIDs, err := client.GetAllMessageUIDs(ctx)
	if err != nil {
		log.Printf("[email-sync] failed to get %s Message-IDs: %v", archiveFolder, err)
		return nil
	}

	log.Printf("[email-sync] found %d messages in %s for reconciliation", len(archiveMessageIDs), archiveFolder)

	// Reconcile Archive
	for normalizedID, dbEmail := range dbEmailMap {
		if dbEmail.EmailAccountID == nil || *dbEmail.EmailAccountID != account.ID {
			continue
		}

		inArchive := archiveMessageIDs[normalizedID]

		// Email is in Archive on IMAP but not marked as archived in DB
		if inArchive && dbEmail.Status != "archived" {
			log.Printf("[email-sync] reconciliation: email %s (normalized: %s) found in %s on IMAP, updating status to 'archived'", dbEmail.MessageID, normalizedID, archiveFolder)
			status := "archived"
			if err := emailService.UpdateEmailFolder(ctx, account.UserID, dbEmail.MessageID, archiveFolder, &status); err != nil {
				log.Printf("[email-sync] failed to update email %s: %v", dbEmail.MessageID, err)
			}
		}

		// Email is in Archive on IMAP but folder in DB is not Archive
		if inArchive && (dbEmail.Folder == nil || *dbEmail.Folder != archiveFolder) {
			log.Printf("[email-sync] reconciliation: email %s found in %s on IMAP, updating folder", dbEmail.MessageID, archiveFolder)
			if err := emailService.UpdateEmailFolder(ctx, account.UserID, dbEmail.MessageID, archiveFolder, nil); err != nil {
				log.Printf("[email-sync] failed to update email folder %s: %v", dbEmail.MessageID, err)
			}
		}
	}

	log.Printf("[email-sync] reconciliation completed for account %d", account.ID)

	return nil
}

// updateSyncStatus updates the sync status for an account
func (j *EmailSyncJob) updateSyncStatus(ctx context.Context, accountID, userID int, status string) error {
	accountService := services.NewEmailAccountService(j.db)
	return accountService.UpdateSyncStatus(ctx, userID, accountID, status)
}

// Verify EmailSyncJob implements ScheduledJob interface
var _ services.ScheduledJob = (*EmailSyncJob)(nil)
