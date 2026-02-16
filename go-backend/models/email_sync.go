package models

import "time"

// EmailAccount represents a configured email account for synchronization
type EmailAccount struct {
	ID                 int        `json:"id"`
	UserID             int        `json:"user_id"`
	EmailAddress       string     `json:"email_address"`
	JMAPServerURL      string     `json:"jmap_server_url"`
	ApiTokenEncrypted  *string    `json:"api_token_encrypted,omitempty"`
	IsActive           bool       `json:"is_active"`
	LastSyncAt         *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus         string     `json:"sync_status"`
	JMAPState          *string    `json:"jmap_state,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
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
	Status         string     `json:"status"`
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
	EmailAddress string  `json:"email_address"`
	ApiToken     *string `json:"api_token,omitempty"`
}

// UpdateEmailAccountParams represents parameters for updating an email account
type UpdateEmailAccountParams struct {
	IsActive   *bool   `json:"is_active,omitempty"`
	SyncStatus *string `json:"sync_status,omitempty"`
}

// EmailListFilters represents filters for listing emails
type EmailListFilters struct {
	Status *string `json:"status,omitempty"`
	Folder *string `json:"folder,omitempty"`
	Limit  *int    `json:"limit,omitempty"`
	Offset *int    `json:"offset,omitempty"`
}
