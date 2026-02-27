package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// EmailService handles email storage operations
type EmailService struct {
	db models.Database
}

// NewEmailService creates a new EmailService
func NewEmailService(db models.Database) *EmailService {
	return &EmailService{
		db: db,
	}
}

// CreateEmail creates or updates an email with upsert logic
// Uses ON CONFLICT (user_id, message_id) to handle duplicates
func (s *EmailService) CreateEmail(ctx context.Context, email models.Email) (*models.Email, error) {
	// Build the upsert query
	query := `
		INSERT INTO emails (
			user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (user_id, message_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			from_address = EXCLUDED.from_address,
			from_name = EXCLUDED.from_name,
			body_text = EXCLUDED.body_text,
			body_html = EXCLUDED.body_html,
			folder = EXCLUDED.folder,
			imap_uid = EXCLUDED.imap_uid,
			is_read = EXCLUDED.is_read,
			updated_at = NOW()
		RETURNING id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, created_at, updated_at
	`

	// Handle nullable fields
	var accountID interface{} = nil
	if email.EmailAccountID != nil {
		accountID = *email.EmailAccountID
	}
	var threadID interface{} = nil
	if email.ThreadID != nil {
		threadID = *email.ThreadID
	}
	var subject interface{} = nil
	if email.Subject != nil {
		subject = *email.Subject
	}
	var fromAddress interface{} = nil
	if email.FromAddress != nil {
		fromAddress = *email.FromAddress
	}
	var fromName interface{} = nil
	if email.FromName != nil {
		fromName = *email.FromName
	}
	var toAddresses interface{} = nil
	if email.ToAddresses != nil {
		toAddresses = *email.ToAddresses
	}
	var bodyText interface{} = nil
	if email.BodyText != nil {
		bodyText = *email.BodyText
	}
	var bodyHTML interface{} = nil
	if email.BodyHTML != nil {
		bodyHTML = *email.BodyHTML
	}
	var receivedAt interface{} = nil
	if email.ReceivedAt != nil {
		receivedAt = *email.ReceivedAt
	}
	var folder interface{} = nil
	if email.Folder != nil {
		folder = *email.Folder
	}
	var imapUID interface{} = nil
	if email.IMAPUID != nil {
		imapUID = *email.IMAPUID
	}

	// Default status if not set
	status := email.Status
	if status == "" {
		status = "unprocessed"
	}

	var result models.Email
	err := s.db.QueryRowContext(ctx, query,
		email.UserID, accountID, email.MessageID, threadID, subject,
		fromAddress, fromName, toAddresses, bodyText, bodyHTML,
		receivedAt, folder, imapUID, status, email.IsRead,
	).Scan(
		&result.ID,
		&result.UserID,
		&result.EmailAccountID,
		&result.MessageID,
		&result.ThreadID,
		&result.Subject,
		&result.FromAddress,
		&result.FromName,
		&result.ToAddresses,
		&result.BodyText,
		&result.BodyHTML,
		&result.ReceivedAt,
		&result.Folder,
		&result.IMAPUID,
		&result.Status,
		&result.IsRead,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update email: %w", err)
	}

	log.Printf("[email] created/updated email %s for user %d", email.MessageID, email.UserID)

	// Index email in Typesense for search
	go UpsertEmailToTypesense(s.db, result)

	return &result, nil
}

// ListEmails lists emails with pagination and optional filters
func (s *EmailService) ListEmails(ctx context.Context, userID int, filters models.EmailListFilters) ([]models.Email, int, error) {
	// Build WHERE clause dynamically
	whereConditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argPos := 2

	if filters.Status != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.Folder != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("folder = $%d", argPos))
		args = append(args, *filters.Folder)
		argPos++
	}

	if filters.IsRead != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("is_read = $%d", argPos))
		args = append(args, *filters.IsRead)
		argPos++
	}

	if filters.FromAddress != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("from_address = $%d", argPos))
		args = append(args, *filters.FromAddress)
		argPos++
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM emails WHERE %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count emails: %w", err)
	}

	// Apply pagination defaults
	limit := 50
	if filters.Limit != nil {
		limit = *filters.Limit
	}
	offset := 0
	if filters.Offset != nil {
		offset = *filters.Offset
	}

	// Query emails with pagination
	query := fmt.Sprintf(`
		SELECT id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id, created_at, updated_at
		FROM emails
		WHERE %s
		ORDER BY received_at DESC NULLS LAST, created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var email models.Email
		var accountID sql.NullInt32
		var threadID sql.NullString
		var subject sql.NullString
		var fromAddress sql.NullString
		var fromName sql.NullString
		var toAddresses sql.NullString
		var bodyText sql.NullString
		var bodyHTML sql.NullString
		var receivedAt sql.NullTime
		var folder sql.NullString
		var imapUID sql.NullInt64
		var cardID sql.NullInt32

		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&accountID,
			&email.MessageID,
			&threadID,
			&subject,
			&fromAddress,
			&fromName,
			&toAddresses,
			&bodyText,
			&bodyHTML,
			&receivedAt,
			&folder,
			&imapUID,
			&email.Status,
			&email.IsRead,
			&cardID,
			&email.CreatedAt,
			&email.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan email: %w", err)
		}

		// Convert nullable fields to pointers
		if accountID.Valid {
			id := int(accountID.Int32)
			email.EmailAccountID = &id
		}
		if threadID.Valid {
			email.ThreadID = &threadID.String
		}
		if subject.Valid {
			email.Subject = &subject.String
		}
		if fromAddress.Valid {
			email.FromAddress = &fromAddress.String
		}
		if fromName.Valid {
			email.FromName = &fromName.String
		}
		if toAddresses.Valid {
			email.ToAddresses = &toAddresses.String
		}
		if bodyText.Valid {
			email.BodyText = &bodyText.String
		}
		if bodyHTML.Valid {
			email.BodyHTML = &bodyHTML.String
		}
		if receivedAt.Valid {
			email.ReceivedAt = &receivedAt.Time
		}
		if folder.Valid {
			email.Folder = &folder.String
		}
		if imapUID.Valid {
			uid := imapUID.Int64
			email.IMAPUID = &uid
		}
		if cardID.Valid {
			id := int(cardID.Int32)
			email.CardID = &id
		}

		emails = append(emails, email)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating emails: %w", err)
	}

	return emails, total, nil
}

// GetEmailByID retrieves a single email by ID
func (s *EmailService) GetEmailByID(ctx context.Context, userID, emailID int) (*models.Email, error) {
	var email models.Email
	var accountID sql.NullInt32
	var threadID sql.NullString
	var subject sql.NullString
	var fromAddress sql.NullString
	var fromName sql.NullString
	var toAddresses sql.NullString
	var bodyText sql.NullString
	var bodyHTML sql.NullString
	var receivedAt sql.NullTime
	var folder sql.NullString
	var imapUID sql.NullInt64

	var cardID sql.NullInt32

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id,
			created_at, updated_at
		FROM emails
		WHERE id = $1 AND user_id = $2
	`, emailID, userID).Scan(
		&email.ID,
		&email.UserID,
		&accountID,
		&email.MessageID,
		&threadID,
		&subject,
		&fromAddress,
		&fromName,
		&toAddresses,
		&bodyText,
		&bodyHTML,
		&receivedAt,
		&folder,
		&imapUID,
		&email.Status,
		&email.IsRead,
		&cardID,
		&email.CreatedAt,
		&email.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("email not found")
		}
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Convert nullable fields to pointers
	if accountID.Valid {
		id := int(accountID.Int32)
		email.EmailAccountID = &id
	}
	if threadID.Valid {
		email.ThreadID = &threadID.String
	}
	if subject.Valid {
		email.Subject = &subject.String
	}
	if fromAddress.Valid {
		email.FromAddress = &fromAddress.String
	}
	if fromName.Valid {
		email.FromName = &fromName.String
	}
	if toAddresses.Valid {
		email.ToAddresses = &toAddresses.String
	}
	if bodyText.Valid {
		email.BodyText = &bodyText.String
	}
	if bodyHTML.Valid {
		email.BodyHTML = &bodyHTML.String
	}
	if receivedAt.Valid {
		email.ReceivedAt = &receivedAt.Time
	}
	if folder.Valid {
		email.Folder = &folder.String
	}
	if imapUID.Valid {
		uid := imapUID.Int64
		email.IMAPUID = &uid
	}
	if cardID.Valid {
		id := int(cardID.Int32)
		email.CardID = &id
	}

	return &email, nil
}

// GetEmailStats returns counts of emails grouped by status
func (s *EmailService) GetEmailStats(ctx context.Context, userID int) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM emails
		WHERE user_id = $1
		GROUP BY status
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email stats: %w", err)
		}
		stats[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating email stats: %w", err)
	}

	return stats, nil
}

// GetTopSenders returns top senders by email count with optional status filter
func (s *EmailService) GetTopSenders(ctx context.Context, userID int, statusFilter *string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT from_address, from_name, COUNT(*) as count
		FROM emails
		WHERE user_id = $1 AND from_address IS NOT NULL AND from_address != ''
	`
	args := []interface{}{userID}
	argPos := 2

	if statusFilter != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *statusFilter)
		argPos++
	}

	query += `
		GROUP BY from_address, from_name
		ORDER BY count DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get top senders: %w", err)
	}
	defer rows.Close()

	var senders []map[string]interface{}
	for rows.Next() {
		var fromAddress string
		var fromName sql.NullString
		var count int

		err := rows.Scan(&fromAddress, &fromName, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sender stats: %w", err)
		}

		sender := map[string]interface{}{
			"from_address": fromAddress,
			"count":        count,
		}
		if fromName.Valid && fromName.String != "" {
			sender["from_name"] = fromName.String
		}
		senders = append(senders, sender)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sender stats: %w", err)
	}

	return senders, nil
}

// UpdateEmailStatus updates the status of an email
func (s *EmailService) UpdateEmailStatus(ctx context.Context, userID, emailID int, status string) (*models.Email, error) {
	// Validate status
	validStatuses := map[string]bool{
		"unprocessed": true,
		"triaged":     true,
		"reviewed":    true,
		"archived":    true,
		"deleted":     true,
		"converted":   true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	var email models.Email
	var accountID sql.NullInt32
	var threadID sql.NullString
	var subject sql.NullString
	var fromAddress sql.NullString
	var fromName sql.NullString
	var toAddresses sql.NullString
	var bodyText sql.NullString
	var bodyHTML sql.NullString
	var receivedAt sql.NullTime
	var folder sql.NullString
	var imapUID sql.NullInt64
	var cardID sql.NullInt32

	err := s.db.QueryRowContext(ctx, `
		UPDATE emails
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id, created_at, updated_at
	`, status, emailID, userID).Scan(
		&email.ID,
		&email.UserID,
		&accountID,
		&email.MessageID,
		&threadID,
		&subject,
		&fromAddress,
		&fromName,
		&toAddresses,
		&bodyText,
		&bodyHTML,
		&receivedAt,
		&folder,
		&imapUID,
		&email.Status,
		&email.IsRead,
		&cardID,
		&email.CreatedAt,
		&email.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("email not found")
		}
		return nil, fmt.Errorf("failed to update email status: %w", err)
	}

	// Convert nullable fields to pointers
	if accountID.Valid {
		id := int(accountID.Int32)
		email.EmailAccountID = &id
	}
	if threadID.Valid {
		email.ThreadID = &threadID.String
	}
	if subject.Valid {
		email.Subject = &subject.String
	}
	if fromAddress.Valid {
		email.FromAddress = &fromAddress.String
	}
	if fromName.Valid {
		email.FromName = &fromName.String
	}
	if toAddresses.Valid {
		email.ToAddresses = &toAddresses.String
	}
	if bodyText.Valid {
		email.BodyText = &bodyText.String
	}
	if bodyHTML.Valid {
		email.BodyHTML = &bodyHTML.String
	}
	if receivedAt.Valid {
		email.ReceivedAt = &receivedAt.Time
	}
	if folder.Valid {
		email.Folder = &folder.String
	}
	if imapUID.Valid {
		uid := imapUID.Int64
		email.IMAPUID = &uid
	}
	if cardID.Valid {
		id := int(cardID.Int32)
		email.CardID = &id
	}

	log.Printf("[email] updated email %d status to %s for user %d", emailID, status, userID)

	return &email, nil
}

// UpdateEmailFolder updates the folder and optionally the status of an email
func (s *EmailService) UpdateEmailFolder(ctx context.Context, userID int, messageID string, folder string, status *string) error {
	var query string
	var args []interface{}

	if status != nil {
		// Update both folder and status
		query = `UPDATE emails SET folder = $1, status = $2, updated_at = NOW() WHERE message_id = $3 AND user_id = $4`
		args = []interface{}{folder, *status, messageID, userID}
	} else {
		// Update only folder
		query = `UPDATE emails SET folder = $1, updated_at = NOW() WHERE message_id = $2 AND user_id = $3`
		args = []interface{}{folder, messageID, userID}
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update email folder: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email not found")
	}

	log.Printf("[email] updated email %s folder to %s for user %d", messageID, folder, userID)

	return nil
}

// GetEmailsByAccountAndFolder retrieves emails for a specific account and folder
func (s *EmailService) GetEmailsByAccountAndFolder(ctx context.Context, userID, accountID int, folder string) ([]models.Email, error) {
	query := `
		SELECT id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id, created_at, updated_at
		FROM emails
		WHERE user_id = $1 AND email_account_id = $2 AND folder = $3
		ORDER BY received_at DESC NULLS LAST, created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, accountID, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails by account and folder: %w", err)
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var email models.Email
		var accountID sql.NullInt32
		var threadID sql.NullString
		var subject sql.NullString
		var fromAddress sql.NullString
		var fromName sql.NullString
		var toAddresses sql.NullString
		var bodyText sql.NullString
		var bodyHTML sql.NullString
		var receivedAt sql.NullTime
		var folder sql.NullString
		var imapUID sql.NullInt64
		var cardID sql.NullInt32

		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&accountID,
			&email.MessageID,
			&threadID,
			&subject,
			&fromAddress,
			&fromName,
			&toAddresses,
			&bodyText,
			&bodyHTML,
			&receivedAt,
			&folder,
			&imapUID,
			&email.Status,
			&email.IsRead,
			&cardID,
			&email.CreatedAt,
			&email.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		// Convert nullable fields to pointers
		if accountID.Valid {
			id := int(accountID.Int32)
			email.EmailAccountID = &id
		}
		if threadID.Valid {
			email.ThreadID = &threadID.String
		}
		if subject.Valid {
			email.Subject = &subject.String
		}
		if fromAddress.Valid {
			email.FromAddress = &fromAddress.String
		}
		if fromName.Valid {
			email.FromName = &fromName.String
		}
		if toAddresses.Valid {
			email.ToAddresses = &toAddresses.String
		}
		if bodyText.Valid {
			email.BodyText = &bodyText.String
		}
		if bodyHTML.Valid {
			email.BodyHTML = &bodyHTML.String
		}
		if receivedAt.Valid {
			email.ReceivedAt = &receivedAt.Time
		}
		if folder.Valid {
			email.Folder = &folder.String
		}
		if imapUID.Valid {
			uid := imapUID.Int64
			email.IMAPUID = &uid
		}
		if cardID.Valid {
			id := int(cardID.Int32)
			email.CardID = &id
		}

		emails = append(emails, email)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating emails: %w", err)
	}

	return emails, nil
}

// BatchUpdateEmailStatus updates the status of multiple emails
func (s *EmailService) BatchUpdateEmailStatus(ctx context.Context, userID int, emailIDs []int, status string) ([]models.Email, error) {
	// Validate status
	validStatuses := map[string]bool{
		"unprocessed": true,
		"triaged":     true,
		"reviewed":    true,
		"archived":    true,
		"deleted":     true,
		"converted":   true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	if len(emailIDs) == 0 {
		return []models.Email{}, nil
	}

	// Build query with IN clause
	placeholder := make([]string, len(emailIDs))
	args := []interface{}{status, userID}
	for i, id := range emailIDs {
		placeholder[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE emails
		SET status = $1, updated_at = NOW()
		WHERE id IN (%s) AND user_id = $2
		RETURNING id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id, created_at, updated_at
	`, strings.Join(placeholder, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch update email status: %w", err)
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var email models.Email
		var accountID sql.NullInt32
		var threadID sql.NullString
		var subject sql.NullString
		var fromAddress sql.NullString
		var fromName sql.NullString
		var toAddresses sql.NullString
		var bodyText sql.NullString
		var bodyHTML sql.NullString
		var receivedAt sql.NullTime
		var folder sql.NullString
		var imapUID sql.NullInt64
		var cardID sql.NullInt32

		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&accountID,
			&email.MessageID,
			&threadID,
			&subject,
			&fromAddress,
			&fromName,
			&toAddresses,
			&bodyText,
			&bodyHTML,
			&receivedAt,
			&folder,
			&imapUID,
			&email.Status,
			&email.IsRead,
			&cardID,
			&email.CreatedAt,
			&email.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		// Convert nullable fields to pointers
		if accountID.Valid {
			id := int(accountID.Int32)
			email.EmailAccountID = &id
		}
		if threadID.Valid {
			email.ThreadID = &threadID.String
		}
		if subject.Valid {
			email.Subject = &subject.String
		}
		if fromAddress.Valid {
			email.FromAddress = &fromAddress.String
		}
		if fromName.Valid {
			email.FromName = &fromName.String
		}
		if toAddresses.Valid {
			email.ToAddresses = &toAddresses.String
		}
		if bodyText.Valid {
			email.BodyText = &bodyText.String
		}
		if bodyHTML.Valid {
			email.BodyHTML = &bodyHTML.String
		}
		if receivedAt.Valid {
			email.ReceivedAt = &receivedAt.Time
		}
		if folder.Valid {
			email.Folder = &folder.String
		}
		if imapUID.Valid {
			uid := imapUID.Int64
			email.IMAPUID = &uid
		}
		if cardID.Valid {
			id := int(cardID.Int32)
			email.CardID = &id
		}

		emails = append(emails, email)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating emails: %w", err)
	}

	log.Printf("[email] batch updated %d emails to status %s for user %d", len(emails), status, userID)

	return emails, nil
}

// BatchConvertEmailsToCards converts multiple emails to cards
func (s *EmailService) BatchConvertEmailsToCards(ctx context.Context, db *sql.DB, userID int, emailIDs []int, title, body string, tags *string) ([]map[string]interface{}, error) {
	if len(emailIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}

	for _, emailID := range emailIDs {
		result, err := s.convertEmailToCard(ctx, db, userID, emailID, title, body, tags)
		if err != nil {
			log.Printf("[email] failed to convert email %d: %v", emailID, err)
			results = append(results, map[string]interface{}{
				"email_id": emailID,
				"success":  false,
				"error":    err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

// convertEmailToCard converts a single email to a card (internal helper)
func (s *EmailService) convertEmailToCard(ctx context.Context, db *sql.DB, userID, emailID int, title, body string, tags *string) (map[string]interface{}, error) {
	// Get the email to verify ownership and get content
	email, err := s.GetEmailByID(ctx, userID, emailID)
	if err != nil {
		return nil, err
	}

	// Use provided title and body, or fall back to email content
	cardTitle := title
	if cardTitle == "" {
		cardTitle = "Email"
		if email.Subject != nil && *email.Subject != "" {
			cardTitle = *email.Subject
		}
	}

	cardBody := body
	if cardBody == "" {
		// Build email body
		var parts []string
		if email.FromName != nil || email.FromAddress != nil {
			from := email.FromName
			if from == nil {
				from = email.FromAddress
			}
			parts = append(parts, fmt.Sprintf("From: %s", *from))
		}
		if email.Subject != nil {
			parts = append(parts, fmt.Sprintf("Subject: %s", *email.Subject))
		}
		parts = append(parts, "")
		if email.BodyText != nil && *email.BodyText != "" {
			parts = append(parts, *email.BodyText)
		} else if email.BodyHTML != nil && *email.BodyHTML != "" {
			parts = append(parts, *email.BodyHTML)
		}
		cardBody = strings.Join(parts, "\n")
	}

	// Create new card with auto-generated card_id
	// We need to use the services.CreateCard function which requires getNextRootCardID
	// For batch operations, we'll generate a simple card_id
	cardID := fmt.Sprintf("email-%d", emailID)

	var cardInternalID int

	// Create the card
	query := `
		INSERT INTO cards (user_id, card_id, title, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err = db.QueryRowContext(ctx, query, userID, cardID, cardTitle, cardBody).Scan(&cardInternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	// Handle tags if provided
	if tags != nil && *tags != "" {
		// Parse comma-separated tags
		tagNames := strings.Split(*tags, ",")
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

		// Create tags and add to card
		for _, tagName := range cleanTags {
			// Insert tag if not exists
			var tagID int
			err = db.QueryRowContext(ctx,
				"INSERT INTO tags (user_id, name, color) VALUES ($1, $2, 'black') ON CONFLICT (user_id, name) DO NOTHING RETURNING id",
				userID, tagName).Scan(&tagID)
			if err == sql.ErrNoRows {
				// Tag already exists, get its ID
				err = db.QueryRowContext(ctx,
					"SELECT id FROM tags WHERE user_id = $1 AND name = $2",
					userID, tagName).Scan(&tagID)
			}
			if err != nil {
				log.Printf("[email] failed to create/get tag %s: %v", tagName, err)
				continue
			}

			// Add tag to card
			_, err = db.ExecContext(ctx,
				"INSERT INTO card_tags (card_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				cardInternalID, tagID)
			if err != nil {
				log.Printf("[email] failed to add tag %s to card: %v", tagName, err)
			}
		}
	}

	// Create email_card_link record
	_, err = db.ExecContext(ctx,
		"INSERT INTO email_card_links (email_id, card_id) VALUES ($1, $2) ON CONFLICT (email_id) DO UPDATE SET card_id = $2",
		emailID, cardInternalID)
	if err != nil {
		log.Printf("[email] failed to create email_card_link: %v", err)
	}

	// Update email's card_id
	_, err = db.ExecContext(ctx,
		"UPDATE emails SET card_id = $1, status = 'converted', updated_at = NOW() WHERE id = $2",
		cardInternalID, emailID)
	if err != nil {
		log.Printf("[email] failed to update email card_id: %v", err)
	}

	log.Printf("[email] converted email %d to card %s", emailID, cardID)

	return map[string]interface{}{
		"email_id": emailID,
		"success":  true,
		"card_id":  cardID,
	}, nil
}

// BatchCreateTasksFromEmails creates tasks from multiple emails
func (s *EmailService) BatchCreateTasksFromEmails(ctx context.Context, db *sql.DB, userID int, emailIDs []int) ([]map[string]interface{}, error) {
	if len(emailIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}

	for _, emailID := range emailIDs {
		result, err := s.createTaskFromEmail(ctx, db, userID, emailID)
		if err != nil {
			log.Printf("[email] failed to create task from email %d: %v", emailID, err)
			results = append(results, map[string]interface{}{
				"email_id": emailID,
				"success":  false,
				"error":    err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

// createTaskFromEmail creates a task from a single email (internal helper)
func (s *EmailService) createTaskFromEmail(ctx context.Context, db *sql.DB, userID, emailID int) (map[string]interface{}, error) {
	// Get the email to verify ownership and get content
	email, err := s.GetEmailByID(ctx, userID, emailID)
	if err != nil {
		return nil, err
	}

	// Create task title from email subject
	taskTitle := "Follow up on email"
	if email.Subject != nil && *email.Subject != "" {
		taskTitle = *email.Subject
	}

	// Create the task
	query := `
		INSERT INTO tasks (user_id, title, status, priority)
		VALUES ($1, $2, 'todo', 'medium')
		RETURNING id
	`
	var taskID int
	err = db.QueryRowContext(ctx, query, userID, taskTitle).Scan(&taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	log.Printf("[email] created task %d from email %d", taskID, emailID)

	return map[string]interface{}{
		"email_id": emailID,
		"success":  true,
		"task_id":  taskID,
	}, nil
}

// GetEmailThreadByID retrieves all emails in a thread by thread_id
func (s *EmailService) GetEmailThreadByID(ctx context.Context, userID int, threadID string) (*models.EmailThread, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}

	// Query all emails in the thread
	query := `
		SELECT id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, card_id, created_at, updated_at
		FROM emails
		WHERE user_id = $1 AND thread_id = $2
		ORDER BY received_at ASC NULLS LAST, created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email thread: %w", err)
	}
	defer rows.Close()

	var emails []models.Email
	var subjects []string
	participants := make(map[string]bool)
	var oldestTime, newestTime *time.Time
	unreadCount := 0

	for rows.Next() {
		var email models.Email
		var accountID sql.NullInt32
		var threadID sql.NullString
		var subject sql.NullString
		var fromAddress sql.NullString
		var fromName sql.NullString
		var toAddresses sql.NullString
		var bodyText sql.NullString
		var bodyHTML sql.NullString
		var receivedAt sql.NullTime
		var folder sql.NullString
		var imapUID sql.NullInt64
		var cardID sql.NullInt32

		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&accountID,
			&email.MessageID,
			&threadID,
			&subject,
			&fromAddress,
			&fromName,
			&toAddresses,
			&bodyText,
			&bodyHTML,
			&receivedAt,
			&folder,
			&imapUID,
			&email.Status,
			&email.IsRead,
			&cardID,
			&email.CreatedAt,
			&email.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		// Convert nullable fields to pointers
		if accountID.Valid {
			id := int(accountID.Int32)
			email.EmailAccountID = &id
		}
		if threadID.Valid {
			email.ThreadID = &threadID.String
		}
		if subject.Valid && subject.String != "" {
			email.Subject = &subject.String
			subjects = append(subjects, subject.String)
		}
		if fromAddress.Valid {
			email.FromAddress = &fromAddress.String
			participants[fromAddress.String] = true
		}
		if fromName.Valid {
			email.FromName = &fromName.String
		}
		if toAddresses.Valid {
			email.ToAddresses = &toAddresses.String
		}
		if bodyText.Valid {
			email.BodyText = &bodyText.String
		}
		if bodyHTML.Valid {
			email.BodyHTML = &bodyHTML.String
		}
		if receivedAt.Valid {
			email.ReceivedAt = &receivedAt.Time
			if oldestTime == nil || receivedAt.Time.Before(*oldestTime) {
				oldestTime = &receivedAt.Time
			}
			if newestTime == nil || receivedAt.Time.After(*newestTime) {
				newestTime = &receivedAt.Time
			}
		}
		if folder.Valid {
			email.Folder = &folder.String
		}
		if imapUID.Valid {
			uid := imapUID.Int64
			email.IMAPUID = &uid
		}
		if cardID.Valid {
			id := int(cardID.Int32)
			email.CardID = &id
		}

		if !email.IsRead {
			unreadCount++
		}

		emails = append(emails, email)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating emails: %w", err)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("thread not found")
	}

	// Determine thread subject (use most common or first non-empty)
	threadSubject := ""
	if len(subjects) > 0 {
		threadSubject = subjects[0]
		// Remove common reply prefixes
		for _, prefix := range []string{"Re: ", "RE: ", "re: ", "Fwd: ", "FWD: ", "fwd: ", "FW: "} {
			threadSubject = strings.TrimPrefix(threadSubject, prefix)
		}
	}

	thread := &models.EmailThread{
		ThreadID:        threadID,
		Subject:         threadSubject,
		ParticipantCount: len(participants),
		MessageCount:    len(emails),
		UnreadCount:     unreadCount,
		OldestDate:      oldestTime,
		NewestDate:      newestTime,
		Messages:        emails,
	}

	return thread, nil
}

// ListEmailThreads lists email threads with pagination and optional filters
func (s *EmailService) ListEmailThreads(ctx context.Context, userID int, filters models.EmailThreadListFilters) ([]models.EmailThread, int, error) {
	// Build WHERE clause dynamically
	whereConditions := []string{"user_id = $1", "thread_id IS NOT NULL"}
	args := []interface{}{userID}
	argPos := 2

	if filters.Status != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.Folder != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("folder = $%d", argPos))
		args = append(args, *filters.Folder)
		argPos++
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Get total count of distinct threads
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT thread_id) FROM emails WHERE %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count threads: %w", err)
	}

	// Apply pagination defaults
	limit := 50
	if filters.Limit != nil {
		limit = *filters.Limit
	}
	offset := 0
	if filters.Offset != nil {
		offset = *filters.Offset
	}

	// Query threads with pagination
	// We need to get thread metadata and then fetch emails for each thread
	query := fmt.Sprintf(`
		SELECT thread_id,
		       COUNT(*) as message_count,
		       SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) as unread_count,
		       MIN(received_at) as oldest_date,
		       MAX(received_at) as newest_date
		FROM emails
		WHERE %s
		GROUP BY thread_id
		ORDER BY MAX(received_at) DESC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threadIDs []string
	var threads []models.EmailThread

	for rows.Next() {
		var threadID string
		var messageCount int
		var unreadCount int
		var oldestDate sql.NullTime
		var newestDate sql.NullTime

		err := rows.Scan(&threadID, &messageCount, &unreadCount, &oldestDate, &newestDate)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan thread metadata: %w", err)
		}

		threadIDs = append(threadIDs, threadID)

		var oldestPtr, newestPtr *time.Time
		if oldestDate.Valid {
			oldestPtr = &oldestDate.Time
		}
		if newestDate.Valid {
			newestPtr = &newestDate.Time
		}

		threads = append(threads, models.EmailThread{
			ThreadID:      threadID,
			MessageCount:  messageCount,
			UnreadCount:   unreadCount,
			OldestDate:    oldestPtr,
			NewestDate:    newestPtr,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating threads: %w", err)
	}

	// Fetch participants and subject for each thread
	for i := range threads {
		thread := &threads[i]

		// Get first email in thread for subject
		var subject sql.NullString
		err := s.db.QueryRowContext(ctx, `
			SELECT subject
			FROM emails
			WHERE user_id = $1 AND thread_id = $2
			ORDER BY created_at ASC
			LIMIT 1
		`, userID, thread.ThreadID).Scan(&subject)

		if err == nil && subject.Valid {
			thread.Subject = subject.String
			// Remove reply prefixes
			for _, prefix := range []string{"Re: ", "RE: ", "re: ", "Fwd: ", "FWD: ", "fwd: ", "FW: "} {
				thread.Subject = strings.TrimPrefix(thread.Subject, prefix)
			}
		}

		// Get unique participants (from addresses)
		participantRows, err := s.db.QueryContext(ctx, `
			SELECT DISTINCT from_address
			FROM emails
			WHERE user_id = $1 AND thread_id = $2 AND from_address IS NOT NULL
		`, userID, thread.ThreadID)
		if err == nil {
			participants := make(map[string]bool)
			for participantRows.Next() {
				var addr string
				if participantRows.Scan(&addr) == nil {
					participants[addr] = true
				}
			}
			participantRows.Close()
			thread.ParticipantCount = len(participants)
		}
	}

	return threads, total, nil
}

// MarkThreadAsRead marks all emails in a thread as read
func (s *EmailService) MarkThreadAsRead(ctx context.Context, userID int, threadID string) error {
	if threadID == "" {
		return fmt.Errorf("thread_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE emails
		SET is_read = true, updated_at = NOW()
		WHERE user_id = $1 AND thread_id = $2
	`, userID, threadID)
	if err != nil {
		return fmt.Errorf("failed to mark thread as read: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("thread not found")
	}

	log.Printf("[email] marked thread %s as read for user %d (%d emails)", threadID, userID, rows)

	return nil
}

// ArchiveThread archives all emails in a thread
func (s *EmailService) ArchiveThread(ctx context.Context, userID int, threadID string) error {
	if threadID == "" {
		return fmt.Errorf("thread_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE emails
		SET status = 'archived', updated_at = NOW()
		WHERE user_id = $1 AND thread_id = $2
	`, userID, threadID)
	if err != nil {
		return fmt.Errorf("failed to archive thread: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("thread not found")
	}

	log.Printf("[email] archived thread %s for user %d (%d emails)", threadID, userID, rows)

	return nil
}

// ExtractFactsFromEmail extracts factual statements from an email using LLM
func (s *EmailService) ExtractFactsFromEmail(ctx context.Context, userID int, emailID int) ([]string, error) {
	// Get the email
	email, err := s.GetEmailByID(ctx, userID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Build email content text for LLM processing
	var emailText string
	if email.Subject != nil && *email.Subject != "" {
		emailText = fmt.Sprintf("Subject: %s\n\n", *email.Subject)
	}
	if email.FromName != nil && *email.FromName != "" {
		emailText += fmt.Sprintf("From: %s <%s>\n\n", *email.FromName, *email.FromAddress)
	} else if email.FromAddress != nil && *email.FromAddress != "" {
		emailText += fmt.Sprintf("From: %s\n\n", *email.FromAddress)
	}

	// Use body_text if available, otherwise fall back to body_html
	bodyContent := ""
	if email.BodyText != nil && *email.BodyText != "" {
		bodyContent = *email.BodyText
	} else if email.BodyHTML != nil && *email.BodyHTML != "" {
		// Strip HTML tags for processing (simple approach)
		// In production, you might want to use a proper HTML-to-text converter
		bodyContent = *email.BodyHTML
	}

	emailText += bodyContent

	if emailText == "" {
		return nil, fmt.Errorf("email has no extractable content")
	}

	// Use LLM to extract facts
	// Import services.llm for LLM processing
	// Create a simple LLM client for fact extraction
	isTesting := false // Don't use testing mode for real email extraction
	client := NewDefaultClient(s.db.(*sql.DB), userID, isTesting)
	client.RequestType = "analysis"

	// Build messages for fact extraction
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an assistant that extracts factual statements from emails.
Extract discrete, verifiable facts from the email content. Focus on:
- Specific dates, times, and deadlines
- Quantifiable information (numbers, prices, quantities)
- Action items and commitments
- Important decisions made
- Names of people, organizations, or locations mentioned

Guidelines:
- Return ONLY a JSON array of fact strings
- Each fact should be a standalone, understandable statement
- Avoid trivial facts like "email was sent on..."
- Do not include opinions or subjective statements
- Maximum 10 facts, prioritize the most important ones

Response format:
["fact 1", "fact 2", "fact 3", ...]`,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Extract factual statements from this email:\n\n%s", emailText),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
	resp, err := ExecuteLLMRequest(ctx, client, messages)
	cancel()

	if err != nil {
		return nil, fmt.Errorf("failed to extract facts from email: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse the response as JSON array
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var facts []string
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		log.Printf("[email] failed to parse facts JSON: %v, content: %s", err, content)
		return nil, fmt.Errorf("failed to parse facts from LLM response: %w", err)
	}

	log.Printf("[email] extracted %d facts from email %d", len(facts), emailID)

	return facts, nil
}
