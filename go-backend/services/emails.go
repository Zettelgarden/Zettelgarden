package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"go-backend/models"
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
			received_at, folder, imap_uid, status, is_read, created_at, updated_at
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

	err := s.db.QueryRowContext(ctx, `
		UPDATE emails
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, email_account_id, message_id, thread_id, subject,
			from_address, from_name, to_addresses, body_text, body_html,
			received_at, folder, imap_uid, status, is_read, created_at, updated_at
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

	log.Printf("[email] updated email %d status to %s for user %d", emailID, status, userID)

	return &email, nil
}
