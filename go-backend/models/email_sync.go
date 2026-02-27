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
	Attachments    []EmailAttachment `json:"attachments,omitempty"` // Attachments parsed during sync
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

// EmailSearchResult represents a single email search result
type EmailSearchResult struct {
	ID       int                    `json:"id"`
	Subject  string                 `json:"subject"`
	Sender   string                 `json:"sender"`
	Preview  string                 `json:"preview"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EmailThread represents a conversation thread with multiple emails
type EmailThread struct {
	ThreadID      string        `json:"thread_id"`
	Subject       string        `json:"subject"`
	ParticipantCount int        `json:"participant_count"`
	MessageCount  int           `json:"message_count"`
	UnreadCount   int           `json:"unread_count"`
	OldestDate    *time.Time    `json:"oldest_date,omitempty"`
	NewestDate    *time.Time    `json:"newest_date,omitempty"`
	Messages      []Email       `json:"messages,omitempty"`
}

// EmailThreadListFilters represents filters for listing email threads
type EmailThreadListFilters struct {
	Status      *string `json:"status,omitempty"`
	Folder      *string `json:"folder,omitempty"`
	Limit       *int    `json:"limit,omitempty"`
	Offset      *int    `json:"offset,omitempty"`
}

// EmailAttachment represents a file attached to an email
type EmailAttachment struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	EmailID      int        `json:"email_id"`
	FileID       *int       `json:"file_id,omitempty"`
	Filename     string     `json:"filename"`
	ContentType  *string    `json:"content_type,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	S3Key        *string    `json:"s3_key,omitempty"`
	ThumbnailPath *string   `json:"thumbnail_path,omitempty"`
	ContentID    *string    `json:"content_id,omitempty"` // Content-ID for inline images
	IsInline     bool       `json:"is_inline"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EmailAttachmentWithDownloadURL extends EmailAttachment with download info
type EmailAttachmentWithDownloadURL struct {
	EmailAttachment
	DownloadURL     string `json:"download_url"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	IsImage         bool   `json:"is_image"`
	IsSavedToVault  bool   `json:"is_saved_to_vault"`
}

// SaveAttachmentToVaultParams represents parameters for saving an attachment to file vault
type SaveAttachmentToVaultParams struct {
	CardPK *int `json:"card_pk,omitempty"` // Optional card to link the file to
}
