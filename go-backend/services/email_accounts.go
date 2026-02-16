package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"go-backend/models"
)

// EmailAccountService handles email account CRUD operations
type EmailAccountService struct {
	db               *sql.DB
	encryptionService *EncryptionService
}

// NewEmailAccountService creates a new EmailAccountService
func NewEmailAccountService(db *sql.DB) *EmailAccountService {
	// Attempt to initialize encryption service, but allow it to be nil for testing
	encryptionService, _ := NewEncryptionService()
	return &EmailAccountService{
		db:               db,
		encryptionService: encryptionService,
	}
}

// CreateEmailAccount creates a new email account for a user
func (s *EmailAccountService) CreateEmailAccount(ctx context.Context, userID int, params models.CreateEmailAccountParams, encryptionKey string) (*models.EmailAccount, error) {
	// Encrypt the app password using EncryptionService if available
	var encryptedPassword *string
	if params.ApiToken != nil && *params.ApiToken != "" {
		var err error
		var encrypted string

		// Use EncryptionService if available, otherwise fall back to placeholder
		if s.encryptionService != nil {
			encrypted, err = s.encryptionService.Encrypt(*params.ApiToken)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt app password: %w", err)
			}
		} else {
			// Fallback to base64 placeholder if encryption service not available
			encrypted, err = encryptApiToken(*params.ApiToken, encryptionKey)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt app password: %w", err)
			}
		}
		encryptedPassword = &encrypted
	}

	// Set default JMAP server URL
	jmapServerURL := "https://api.fastmail.com/jmap/session"

	// Insert into database
	var accountID int
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO email_accounts (user_id, email_address, jmap_server_url, api_token_encrypted, is_active, sync_status)
		VALUES ($1, $2, $3, $4, true, 'active')
		RETURNING id
	`, userID, params.EmailAddress, jmapServerURL, encryptedPassword).Scan(&accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create email account: %w", err)
	}

	log.Printf("[email-account] created account %d for user %d", accountID, userID)

	return s.GetEmailAccountByID(ctx, userID, accountID)
}

// GetEmailAccounts retrieves all email accounts for a user
func (s *EmailAccountService) GetEmailAccounts(ctx context.Context, userID int) ([]models.EmailAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, email_address, jmap_server_url, api_token_encrypted,
		       is_active, last_sync_at, sync_status, jmap_state, created_at, updated_at
		FROM email_accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list email accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.EmailAccount
	for rows.Next() {
		var account models.EmailAccount
		var encryptedPassword sql.NullString
		var jmapState sql.NullString
		var lastSyncTime sql.NullTime

		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.EmailAddress,
			&account.JMAPServerURL,
			&encryptedPassword,
			&account.IsActive,
			&lastSyncTime,
			&account.SyncStatus,
			&jmapState,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email account: %w", err)
		}

		if encryptedPassword.Valid {
			account.ApiTokenEncrypted = &encryptedPassword.String
		}
		if lastSyncTime.Valid {
			account.LastSyncAt = &lastSyncTime.Time
		}
		if jmapState.Valid {
			account.JMAPState = &jmapState.String
		}

		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating email accounts: %w", err)
	}

	return accounts, nil
}

// GetEmailAccountByID retrieves a single email account by ID
func (s *EmailAccountService) GetEmailAccountByID(ctx context.Context, userID, accountID int) (*models.EmailAccount, error) {
	var account models.EmailAccount
	var encryptedPassword sql.NullString
	var jmapState sql.NullString
	var lastSyncTime sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, email_address, jmap_server_url, api_token_encrypted,
		       is_active, last_sync_at, sync_status, jmap_state, created_at, updated_at
		FROM email_accounts
		WHERE id = $1 AND user_id = $2
	`, accountID, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.EmailAddress,
		&account.JMAPServerURL,
		&encryptedPassword,
		&account.IsActive,
		&lastSyncTime,
		&account.SyncStatus,
		&jmapState,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("email account not found")
		}
		return nil, fmt.Errorf("failed to get email account: %w", err)
	}

	if encryptedPassword.Valid {
		account.ApiTokenEncrypted = &encryptedPassword.String
	}
	if lastSyncTime.Valid {
		account.LastSyncAt = &lastSyncTime.Time
	}
	if jmapState.Valid {
		account.JMAPState = &jmapState.String
	}

	return &account, nil
}

// UpdateEmailAccount updates an existing email account
func (s *EmailAccountService) UpdateEmailAccount(ctx context.Context, userID, accountID int, params models.UpdateEmailAccountParams) (*models.EmailAccount, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if params.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *params.IsActive)
		argPos++
	}
	if params.SyncStatus != nil {
		updates = append(updates, fmt.Sprintf("sync_status = $%d", argPos))
		args = append(args, *params.SyncStatus)
		argPos++
	}

	if len(updates) == 0 {
		return s.GetEmailAccountByID(ctx, userID, accountID)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	query := fmt.Sprintf("UPDATE email_accounts SET %s WHERE id = $%d AND user_id = $%d",
		strings.Join(updates, ", "), argPos, argPos+1)
	args = append(args, accountID, userID)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update email account: %w", err)
	}

	return s.GetEmailAccountByID(ctx, userID, accountID)
}

// DeleteEmailAccount deletes an email account
func (s *EmailAccountService) DeleteEmailAccount(ctx context.Context, userID, accountID int) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM email_accounts WHERE id = $1 AND user_id = $2", accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete email account: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email account not found")
	}

	log.Printf("[email-account] deleted account %d for user %d", accountID, userID)
	return nil
}

// UpdateLastSync updates the last_sync_at timestamp for an account
func (s *EmailAccountService) UpdateLastSync(ctx context.Context, userID, accountID int, syncTime time.Time) error {
	result, err := s.db.ExecContext(ctx, "UPDATE email_accounts SET last_sync_at = $1 WHERE id = $2 AND user_id = $3", syncTime, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to update last sync: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email account not found")
	}

	return nil
}

// UpdateJMAPState updates the jmap_state for an account
func (s *EmailAccountService) UpdateJMAPState(ctx context.Context, userID, accountID int, state string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE email_accounts SET jmap_state = $1 WHERE id = $2 AND user_id = $3", state, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to update JMAP state: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email account not found")
	}

	return nil
}

// UpdateSyncStatus updates the sync_status for an account
func (s *EmailAccountService) UpdateSyncStatus(ctx context.Context, userID, accountID int, status string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE email_accounts SET sync_status = $1 WHERE id = $2 AND user_id = $3", status, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email account not found")
	}

	return nil
}

// encryptApiToken encrypts an app password using base64 as fallback
// This is only used when EncryptionService is not available (e.g., in tests)
func encryptApiToken(password, key string) (string, error) {
	// Placeholder: base64 encode
	// In production, EncryptionService should always be available
	return encryptionServiceFallback(password)
}

// decryptApiToken decrypts an encrypted app password using base64 as fallback
// This is only used when EncryptionService is not available (e.g., in tests)
func decryptApiToken(encrypted, key string) (string, error) {
	// Placeholder: base64 decode
	// In production, EncryptionService should always be available
	return decryptionServiceFallback(encrypted)
}

// encryptionServiceFallback provides base64 encoding as fallback
func encryptionServiceFallback(password string) (string, error) {
	return base64FallbackEncrypt(password)
}

// decryptionServiceFallback provides base64 decoding as fallback
func decryptionServiceFallback(encrypted string) (string, error) {
	return base64FallbackDecrypt(encrypted)
}

// base64FallbackEncrypt is a simple base64 encoder for fallback
func base64FallbackEncrypt(password string) (string, error) {
	return simpleBase64Encode(password), nil
}

// base64FallbackDecrypt is a simple base64 decoder for fallback
func base64FallbackDecrypt(encrypted string) (string, error) {
	return simpleBase64Decode(encrypted)
}

// simpleBase64Encode encodes a string to base64
func simpleBase64Encode(s string) string {
	return encodeBase64(s)
}

// simpleBase64Decode decodes a base64 string
func simpleBase64Decode(s string) (string, error) {
	return decodeBase64(s)
}

// encodeBase64 performs base64 encoding
func encodeBase64(s string) string {
	return doBase64Encode(s)
}

// decodeBase64 performs base64 decoding
func decodeBase64(s string) (string, error) {
	return doBase64Decode(s)
}

// doBase64Encode is the actual base64 encoding implementation
func doBase64Encode(s string) string {
	return base64Encode(s)
}

// doBase64Decode is the actual base64 decoding implementation
func doBase64Decode(s string) (string, error) {
	return base64Decode(s)
}

// base64Encode encodes a string to base64
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// base64Decode decodes a base64 string
func base64Decode(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	return string(decoded), nil
}

// DecryptApiToken is exported for use by sync jobs
// This function uses EncryptionService if available, otherwise falls back to base64
func DecryptApiToken(encrypted, key string) (string, error) {
	// Try to use EncryptionService if available
	encryptionService, err := NewEncryptionService()
	if err == nil && encryptionService != nil {
		return encryptionService.Decrypt(encrypted)
	}

	// Fallback to base64 decoding
	return decryptApiToken(encrypted, key)
}
