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
type MailClient struct {
	Host              string
	Password          string
	Testing           bool
	TestingEmailsSent int
	Queue             *EmailQueue
	mu                sync.Mutex
	isProcessing      bool
	DB                *sql.DB
	Tx                models.Database
	ShutdownChan      chan struct{}
	shutdownOnce      sync.Once
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
	return fmt.Sprintf("MailClient{Host:%q, Password:%q, Testing:%t}", m.Host, "<redacted>", m.Testing)
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
