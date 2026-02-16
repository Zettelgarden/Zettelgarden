package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/models"
	"go-backend/services"
)

// EmailSyncJob fetches emails from active email accounts via IMAP
type EmailSyncJob struct {
	db       *sql.DB
	schedule string
}

// NewEmailSyncJob creates a new email sync job
func NewEmailSyncJob(db *sql.DB) *EmailSyncJob {
	return &EmailSyncJob{
		db:       db,
		schedule: "0 */5 * * * *", // Every 5 minutes
	}
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

	// Select INBOX
	if err := client.SelectInbox(ctx); err != nil {
		log.Printf("[email-sync] failed to select INBOX for account %d: %v", account.ID, err)
		j.updateSyncStatus(ctx, account.ID, account.UserID, "error")
		return 0, err
	}

	// Check for UIDVALIDITY changes
	currentUIDValidity := client.GetUIDValidity()
	if account.IMAPUIDValidity != nil && *account.IMAPUIDValidity != int(currentUIDValidity) {
		log.Printf("[email-sync] UIDVALIDITY changed for account %d (old: %d, new: %d), doing full sync",
			account.ID, *account.IMAPUIDValidity, currentUIDValidity)
		account.IMAPUID = nil // Reset to force full sync
	}

	// Fetch emails (using UID if available, otherwise initial fetch)
	var emails []models.Email
	var maxUID uint32

	if account.IMAPUID != nil && *account.IMAPUID > 0 {
		// Incremental sync
		emails, maxUID, err = client.FetchEmailsSinceUID(ctx, uint32(*account.IMAPUID))
	} else {
		// Initial fetch - get recent emails
		emails, maxUID, err = client.FetchRecentEmails(ctx, 100)
	}

	if err != nil {
		log.Printf("[email-sync] failed to fetch emails for account %d: %v", account.ID, err)
		// Update sync status to error
		j.updateSyncStatus(ctx, account.ID, account.UserID, "error")
		return 0, err
	}

	// Create EmailService
	emailService := services.NewEmailService(j.db)

	// Store emails
	for _, email := range emails {
		email.UserID = account.UserID
		email.EmailAccountID = &account.ID

		_, err := emailService.CreateEmail(ctx, email)
		if err != nil {
			log.Printf("[email-sync] failed to store email %s for account %d: %v", email.MessageID, account.ID, err)
			// Continue with next email
		}
	}

	// Update last_sync_at and IMAP state
	now := time.Now()
	accountService := services.NewEmailAccountService(j.db)
	if err := accountService.UpdateLastSync(ctx, account.UserID, account.ID, now); err != nil {
		log.Printf("[email-sync] failed to update last sync for account %d: %v", account.ID, err)
	}
	if err := accountService.UpdateIMAPState(ctx, account.UserID, account.ID, maxUID, currentUIDValidity); err != nil {
		log.Printf("[email-sync] failed to update IMAP state for account %d: %v", account.ID, err)
	}

	// Update sync status to active
	if err := j.updateSyncStatus(ctx, account.ID, account.UserID, "active"); err != nil {
		log.Printf("[email-sync] failed to update sync status for account %d: %v", account.ID, err)
	}

	return len(emails), nil
}

// updateSyncStatus updates the sync status for an account
func (j *EmailSyncJob) updateSyncStatus(ctx context.Context, accountID, userID int, status string) error {
	accountService := services.NewEmailAccountService(j.db)
	return accountService.UpdateSyncStatus(ctx, userID, accountID, status)
}

// Verify EmailSyncJob implements ScheduledJob interface
var _ services.ScheduledJob = (*EmailSyncJob)(nil)
