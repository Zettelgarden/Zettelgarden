package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
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

func (m *MailClient) sendMailImpl(email Email) error {
	// In testing mode, just count the email as "sent"
	if m.Testing {
		m.TestingEmailsSent += 1
		return nil
	}

	emailJSON, err := json.Marshal(email)
	if err != nil {
		return err
	}
	// Create a new request
	req, err := http.NewRequest("POST", m.Host+"/api/send", bytes.NewBuffer(emailJSON))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", m.Password)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error with email client: %s", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("failed to send email: %s", resp.Status)
		return fmt.Errorf("failed to send email: %s", resp.Status)
	}
	return nil
}

func (m *MailClient) SendEmail(subject, recipient, body string) error {
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
			if !m.isProcessing && m.Queue.Length() == 0 {
				log.Printf("mail queue shutdown complete")
				return nil
			}
		}
	}
}
