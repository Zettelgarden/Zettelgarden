package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	goMail "github.com/wneessen/go-mail"
)

func NewEmailQueue() *EmailQueue {
	return &EmailQueue{
		queue: make([]Email, 0),
	}
}

// Push adds an email to the queue
func (eq *EmailQueue) Push(email Email) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	eq.queue = append(eq.queue, email)
}

// Pop removes and returns the first email from the queue
// Returns false if queue is empty
func (eq *EmailQueue) Pop() (Email, bool) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if len(eq.queue) == 0 {
		return Email{}, false
	}

	// Get the first email
	email := eq.queue[0]
	// Remove it from the queue
	eq.queue = eq.queue[1:]

	return email, true
}

// Length returns the current size of the queue
func (eq *EmailQueue) Length() int {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return len(eq.queue)
}

// Mail timeouts: a dead SMTP server must not wedge the queue forever. The
// dial timeout bounds connecting (and the TLS handshake); the send context
// bounds the full dial+envelope+data exchange.
const (
	mailDialTimeout = 10 * time.Second
	mailSendTimeout = 30 * time.Second
)

// sendMailImpl delivers one email directly over SMTP (the separate python-mail
// Flask service was retired, bead 6er.12). In testing mode it just counts the
// email as "sent" without touching the network.
func (m *MailClient) sendMailImpl(email Email) error {
	// In testing mode, just count the email as "sent"
	if m.Testing {
		m.TestingEmailsSent += 1
		return nil
	}

	// Build the message. The library handles RFC 2047 encoding for non-ASCII
	// subjects, quoted-printable/base64 body encoding, and UTF-8 headers.
	msg := goMail.NewMsg()
	if err := msg.FromFormat("Zettelgarden", m.SMTPFrom); err != nil {
		return fmt.Errorf("invalid SMTP from address %q: %w", m.SMTPFrom, err)
	}
	if err := msg.To(email.Recipient); err != nil {
		return fmt.Errorf("invalid recipient address %q: %w", email.Recipient, err)
	}
	msg.Subject(email.Subject)
	if email.IsHTML {
		msg.SetBodyString(goMail.TypeTextHTML, email.Body)
	} else {
		msg.SetBodyString(goMail.TypeTextPlain, email.Body)
	}

	opts := []goMail.Option{
		goMail.WithPort(m.SMTPPort),
		goMail.WithTimeout(mailDialTimeout),
	}
	// Implicit TLS on 465; explicit STARTTLS (default) elsewhere; plain when
	// SMTP_STARTTLS=false for local relays without TLS.
	switch {
	case m.SMTPPort == 465:
		opts = append(opts, goMail.WithSSL())
	case m.StartTLS:
		opts = append(opts, goMail.WithTLSPolicy(goMail.TLSMandatory))
	default:
		opts = append(opts, goMail.WithTLSPolicy(goMail.NoTLS))
	}
	// Authenticate only when both username and password are present (local
	// relays without auth send unauthenticated).
	if m.SMTPUsername != "" && m.SMTPPassword != "" {
		opts = append(opts,
			goMail.WithSMTPAuth(goMail.SMTPAuthPlain),
			goMail.WithUsername(m.SMTPUsername),
			goMail.WithPassword(m.SMTPPassword),
		)
	}
	if m.insecureSkipTLSVerify {
		opts = append(opts, goMail.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	}

	client, err := goMail.NewClient(m.SMTPHost, opts...)
	if err != nil {
		return fmt.Errorf("error creating SMTP client: %w", err)
	}

	// Bound the whole exchange so a hung server can't wedge the queue.
	ctx, cancel := context.WithTimeout(context.Background(), mailSendTimeout)
	defer cancel()
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		log.Printf("error sending email via SMTP: %s", err)
		return err
	}
	return nil
}

func (m *MailClient) SendEmail(subject, recipient, body string) error {
	if m.Disabled {
		return nil
	}
	if m.Testing {
		m.TestingEmailsSent += 1
		return nil
	}
	email := Email{
		Subject:   subject,
		Recipient: recipient,
		Body:      body,
		IsHTML:    false, // default to plain text
	}
	m.Queue.Push(email)
	m.startProcessing()

	return nil
}

// SendHTMLEmail is a convenience method for sending HTML emails
func (m *MailClient) SendHTMLEmail(subject, recipient, body string) error {
	if m.Disabled {
		return nil
	}
	if m.Testing {
		m.TestingEmailsSent += 1
		return nil
	}
	email := Email{
		Subject:   subject,
		Recipient: recipient,
		Body:      body,
		IsHTML:    true,
	}
	m.Queue.Push(email)
	m.startProcessing()

	return nil
}

func (m *MailClient) startProcessing() {
	m.mu.Lock()
	if m.isProcessing {
		m.mu.Unlock()
		return
	}
	m.isProcessing = true
	m.mu.Unlock()

	go m.processQueue()
}

func (m *MailClient) processQueue() {
	for {
		// Check for shutdown signal
		select {
		case <-m.ShutdownChan:
			log.Printf("mail queue shutdown requested")
			m.mu.Lock()
			m.isProcessing = false
			m.mu.Unlock()
			return
		default:
			// Continue processing
		}

		// Get next email from queue
		email, ok := m.Queue.Pop()
		if !ok {
			// Queue is empty, stop processing
			m.mu.Lock()
			m.isProcessing = false
			m.mu.Unlock()
			return
		}

		// Send the email
		err := m.sendMailImpl(email)
		if err != nil {
			// Handle error - maybe log it or requeue the email
			log.Printf("Failed to send email: %v", err)
			// Optional: requeue the failed email
			email.Retries += 1
			if email.Retries < 4 {
				m.Queue.Push(email)
			}
		}

		// Optional: add a small delay between sends (skip in testing mode)
		if !m.Testing {
			time.Sleep(1000 * time.Millisecond)
		}
	}
}

// Shutdown gracefully shuts down the mail queue processing.
// It waits for the current queue to be drained or context cancellation.
func (m *MailClient) Shutdown(ctx context.Context) error {
	log.Printf("initiating mail queue shutdown (queue size: %d)", m.Queue.Length())

	// Signal shutdown
	m.shutdownOnce.Do(func() {
		close(m.ShutdownChan)
	})

	// Wait for queue to drain or context cancellation
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			remaining := m.Queue.Length()
			if remaining > 0 {
				log.Printf("mail queue shutdown timed out with %d emails remaining", remaining)
			}
			return ctx.Err()
		case <-ticker.C:
			m.mu.Lock()
			processing := m.isProcessing
			m.mu.Unlock()
			if !processing && m.Queue.Length() == 0 {
				log.Printf("mail queue shutdown complete")
				return nil
			}
		}
	}
}
