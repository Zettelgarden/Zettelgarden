package mail

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"sync"
)

// MailClient sends transactional email via an in-process queue. The previous
// integration with the LLM job queue (JobQueue/WorkerPool/EmailProcessor) has
// been removed; email is delivered synchronously off the EmailQueue by
// processQueue in mail.go.
//
// Delivery is direct SMTP (the separate python-mail Flask service was removed,
// bead 6er.12): SMTPHost/SMTPPort/SMTPUsername/SMTPPassword/SMTPFrom/StartTLS
// describe the relay, and sendMailImpl builds an RFC 5322 message with the
// wneessen/go-mail library. STARTTLS (SMTP_STARTTLS, default true) matches the
// retired service's MAIL_USE_TLS=True; port 465 uses implicit TLS.
//
// Disabled marks a no-op client for self-hosters without SMTP (6er.6):
// SendEmail/SendHTMLEmail return nil immediately and nothing is queued, so
// callers (validation mails, reminders, notifications) degrade gracefully
// without nil checks.
type MailClient struct {
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	SMTPFrom          string
	StartTLS          bool
	Disabled          bool
	Testing           bool
	TestingEmailsSent int
	Queue             *EmailQueue
	mu                sync.Mutex
	isProcessing      bool
	DB                *sql.DB
	Tx                models.Database
	ShutdownChan      chan struct{}
	shutdownOnce      sync.Once
	// insecureSkipTLSVerify is a test-only hook: the fake-SMTP tests present a
	// self-signed certificate and must skip verification. Never set in prod.
	insecureSkipTLSVerify bool
}

func (m *MailClient) db() models.Database {
	if m.Tx != nil {
		return m.Tx
	}
	return m.DB
}

func (m *MailClient) String() string {
	if m == nil {
		return "<nil>"
	}
	return fmt.Sprintf("MailClient{SMTPHost:%q, SMTPPort:%d, SMTPFrom:%q, StartTLS:%t, Testing:%t}",
		m.SMTPHost, m.SMTPPort, m.SMTPFrom, m.StartTLS, m.Testing)
}

type Email struct {
	Subject   string `json:"subject"`
	Recipient string `json:"recipient"`
	Body      string `json:"body"`
	IsHTML    bool   `json:"is_html"`
	Retries   int
}

type EmailQueue struct {
	queue []Email
	mu    sync.Mutex
}
