package models

import "time"

// EmailAccount represents a configured email account for synchronization
type EmailAccount struct {
	ID                     int        `json:"id"`
	UserID                 int        `json:"user_id"`
	EmailAddress           string     `json:"email_address"`
	IMAPServer             string     `json:"imap_server,omitempty"`             // e.g., "imap.fastmail.com:993"
	IMAPServerType         string     `json:"imap_server_type,omitempty"`      // "imap"
	AppPasswordEncrypted   *string    `json:"app_password_encrypted,omitempty"` // Encrypted app password
	IsActive               bool       `json:"is_active"`
	LastSyncAt             *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus             string     `json:"sync_status"`
	IMAPUID                *int       `json:"imap_uid,omitempty"`    // Last UID synced
	IMAPUIDValidity        *int       `json:"imap_uid_validity,omitempty"` // UIDVALIDITY for mailbox
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// Email represents a synchronized email message
type Email struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	EmailAccountID *int       `json:"email_account_id,omitempty"`
	MessageID      string     `json:"message_id"`
	ThreadID       *string    `json:"thread_id,omitempty"`
	Subject        *string    `json:"subject,omitempty"`
	FromAddress    *string    `json:"from_address,omitempty"`
	FromName       *string    `json:"from_name,omitempty"`
	ToAddresses    *string    `json:"to_addresses,omitempty"`
	BodyText       *string    `json:"body_text,omitempty"`
	BodyHTML       *string    `json:"body_html,omitempty"`
	ReceivedAt     *time.Time `json:"received_at,omitempty"`
	Folder         *string    `json:"folder,omitempty"`
	IMAPUID        *int64     `json:"imap_uid,omitempty"` // IMAP message UID for folder operations
	Status         string     `json:"status"`
	IsRead         bool       `json:"is_read"`             // Whether the email has been read (synced with IMAP \Seen flag)
	CardID         *int       `json:"card_id,omitempty"`   // ID of card created from this email
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// EmailTriageDecision represents an AI-powered triage recommendation for an email
type EmailTriageDecision struct {
	ID             int       `json:"id"`
	EmailID        int       `json:"email_id"`
	Decision       string    `json:"decision"`
	Confidence     float64   `json:"confidence"`
	Reasoning      *string   `json:"reasoning,omitempty"`
	IsAutoExecuted bool      `json:"is_auto_executed"`
	CreatedAt      time.Time `json:"created_at"`
}

// EmailCardLink represents a link between an email and a card created from it
type EmailCardLink struct {
	ID        int       `json:"id"`
	EmailID   int       `json:"email_id"`
	CardID    int       `json:"card_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateEmailAccountParams represents parameters for creating an email account
type CreateEmailAccountParams struct {
	EmailAddress   string  `json:"email_address"`
	AppPassword    *string `json:"app_password,omitempty"` // App password (will be encrypted)
}

// UpdateEmailAccountParams represents parameters for updating an email account
type UpdateEmailAccountParams struct {
	IsActive   *bool   `json:"is_active,omitempty"`
	SyncStatus *string `json:"sync_status,omitempty"`
}

// EmailListFilters represents filters for listing emails
type EmailListFilters struct {
	Status      *string `json:"status,omitempty"`
	Folder      *string `json:"folder,omitempty"`
	IsRead      *bool   `json:"is_read,omitempty"`   // Filter by read status
	FromAddress *string `json:"from_address,omitempty"` // Filter by sender email address
	Limit       *int    `json:"limit,omitempty"`
	Offset      *int    `json:"offset,omitempty"`
}

// ConvertEmailParams represents parameters for converting an email to a card
type ConvertEmailParams struct {
	Title   string  `json:"title"`
	Body    *string `json:"body,omitempty"`
	Tags    *string `json:"tags,omitempty"`
	CardID  *string `json:"card_id,omitempty"` // For linking to existing card
}
