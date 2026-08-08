package mail

import (
	"context"
	"testing"
	"time"
)

func TestMailClientShutdown(t *testing.T) {
	client := &MailClient{
		Queue:        NewEmailQueue(),
		ShutdownChan: make(chan struct{}),
		Testing:      true, // Bypass actual sending
	}

	// Add some emails to the queue
	client.Queue.Push(Email{Subject: "test1", Recipient: "test1@example.com", Body: "body1"})
	client.Queue.Push(Email{Subject: "test2", Recipient: "test2@example.com", Body: "body2"})

	// Start processing in background
	go client.processQueue()

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Shutdown with context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := client.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify queue is drained or processing stopped
	if client.isProcessing {
		t.Error("Client should not be processing after shutdown")
	}

	// In testing mode, queue should be empty since emails are "sent" immediately
	if client.Queue.Length() != 0 {
		t.Errorf("Queue should be empty in testing mode, got %d", client.Queue.Length())
	}
}

func TestMailClientShutdownTimeout(t *testing.T) {
	client := &MailClient{
		Queue:        NewEmailQueue(),
		ShutdownChan: make(chan struct{}),
	}

	// Add many emails to ensure queue doesn't drain quickly
	for i := 0; i < 100; i++ {
		client.Queue.Push(Email{Subject: "test", Recipient: "test@example.com", Body: "body"})
	}

	// Start processing
	go client.processQueue()

	// Shutdown with very short timeout - should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Shutdown(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}
}

// TestMailClientDisabledNoOp verifies a Disabled client (mail off, 6er.6)
// swallows sends without queueing or counting — callers degrade gracefully.
func TestMailClientDisabledNoOp(t *testing.T) {
	client := &MailClient{
		Disabled: true,
		Queue:    NewEmailQueue(),
	}

	if err := client.SendEmail("subject", "recipient@example.com", "body"); err != nil {
		t.Fatalf("disabled SendEmail returned error: %v", err)
	}
	if err := client.SendHTMLEmail("subject", "recipient@example.com", "<b>body</b>"); err != nil {
		t.Fatalf("disabled SendHTMLEmail returned error: %v", err)
	}
	if client.Queue.Length() != 0 {
		t.Errorf("disabled client must not queue emails, got %d", client.Queue.Length())
	}
	if client.TestingEmailsSent != 0 {
		t.Errorf("disabled client must not count emails, got %d", client.TestingEmailsSent)
	}
}
