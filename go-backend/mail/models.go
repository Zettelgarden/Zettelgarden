package mail

import (
	"database/sql"
	"fmt"
	"sync"
)

type MailClient struct {
	Host              string
	Password          string
	Testing           bool
	TestingEmailsSent int
	Queue             *EmailQueue
	mu                sync.Mutex
	isProcessing      bool
	DB                *sql.DB
	ShutdownChan      chan struct{}
	shutdownOnce      sync.Once
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
