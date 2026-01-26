package mail

import (
	"context"
	"go-backend/models"
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

func TestEmailRateLimiter_Allow(t *testing.T) {
	limiter := NewEmailRateLimiter(2, 10) // 2 per minute, 10 per day

	// First request should be allowed
	allowed, _ := limiter.Allow(1)
	if !allowed {
		t.Error("First request should be allowed")
	}

	// Second request should be allowed
	allowed, _ = limiter.Allow(1)
	if !allowed {
		t.Error("Second request should be allowed")
	}

	// Third request should exceed minute limit
	allowed, retryAfter := limiter.Allow(1)
	if allowed {
		t.Error("Third request should exceed minute limit")
	}
	if retryAfter <= 0 {
		t.Error("RetryAfter should be positive when rate limited")
	}
}

func TestEmailRateLimiter_DifferentUsers(t *testing.T) {
	limiter := NewEmailRateLimiter(1, 10) // 1 per minute

	// User 1 can send
	allowed, _ := limiter.Allow(1)
	if !allowed {
		t.Error("User 1 first request should be allowed")
	}

	// User 1 cannot send again immediately
	allowed, _ = limiter.Allow(1)
	if allowed {
		t.Error("User 1 second request should be rate limited")
	}

	// User 2 can still send (independent limits)
	allowed, _ = limiter.Allow(2)
	if !allowed {
		t.Error("User 2 first request should be allowed")
	}
}

func TestEmailRateLimiter_CleanupAndReset(t *testing.T) {
	limiter := NewEmailRateLimiter(1, 10) // 1 per minute, 10 per day

	// Send first email - user 1
	allowed, _ := limiter.Allow(1)
	if !allowed {
		t.Error("First request for user 1 should be allowed")
	}

	// User 1 tries again - should be rate limited
	allowed, _ = limiter.Allow(1)
	if allowed {
		t.Error("Second request for user 1 should be rate limited")
	}

	// Cleanup user 1 info (simulating time passing / cache expiration)
	limiter.CleanupUserInfo(1)

	// User 1 should be able to send again after cleanup
	allowed, _ = limiter.Allow(1)
	if !allowed {
		t.Error("Request should be allowed after cleanup")
	}

	// User 2 should have independent limit
	allowed, _ = limiter.Allow(2)
	if !allowed {
		t.Error("First request for user 2 should be allowed")
	}
}

func TestEmailRateLimiter_CleanupUserInfo(t *testing.T) {
	limiter := NewEmailRateLimiter(1, 10)

	// Use the rate limiter
	limiter.Allow(1)

	// Cleanup user info
	limiter.CleanupUserInfo(1)

	// User should be able to send again (fresh state)
	allowed, _ := limiter.Allow(1)
	if !allowed {
		t.Error("Request should be allowed after cleanup")
	}
}

func TestEmailProcessor_ProcessJob_Success(t *testing.T) {
	client := &MailClient{
		Host:     "http://localhost:8080",
		Password: "test",
		Testing:  true,
	}
	processor := NewEmailProcessor(client)

	job := &models.LLMJob{
		ID:     1,
		UserID: 1,
		Payload: map[string]interface{}{
			"subject":   "Test Subject",
			"recipient": "test@example.com",
			"body":      "Test Body",
			"is_html":   false,
		},
	}

	result, err := processor.ProcessJob(context.Background(), job)
	if err != nil {
		t.Errorf("ProcessJob failed: %v", err)
	}

	if result["recipient"] != "test@example.com" {
		t.Errorf("Expected recipient test@example.com, got %v", result["recipient"])
	}

	if result["subject"] != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got %v", result["subject"])
	}

	if _, ok := result["sent_at"]; !ok {
		t.Error("Expected sent_at in result")
	}
}

func TestEmailProcessor_ProcessJob_MissingPayload(t *testing.T) {
	client := &MailClient{
		Host:     "http://localhost:8080",
		Password: "test",
		Testing:  true,
	}
	processor := NewEmailProcessor(client)

	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name: "missing subject",
			payload: map[string]interface{}{
				"recipient": "test@example.com",
				"body":      "Test Body",
			},
		},
		{
			name: "missing recipient",
			payload: map[string]interface{}{
				"subject": "Test Subject",
				"body":    "Test Body",
			},
		},
		{
			name: "missing body",
			payload: map[string]interface{}{
				"subject":   "Test Subject",
				"recipient": "test@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &models.LLMJob{
				ID:       1,
				UserID:   1,
				Payload:  tt.payload,
				JobType:  models.JobTypeEmail,
			}

			_, err := processor.ProcessJob(context.Background(), job)
			if err == nil {
				t.Error("Expected error for missing payload field")
			}
		})
	}
}

func TestEmailProcessor_ProcessJob_RateLimit(t *testing.T) {
	client := &MailClient{
		Host:        "http://localhost:8080",
		Password:    "test",
		Testing:     true,
		RateLimiter: NewEmailRateLimiter(1, 10), // 1 per minute
	}
	processor := NewEmailProcessor(client)

	job := &models.LLMJob{
		ID:     1,
		UserID: 1,
		Payload: map[string]interface{}{
			"subject":   "Test Subject",
			"recipient": "test@example.com",
			"body":      "Test Body",
			"is_html":   false,
		},
	}

	// First request should succeed
	_, err := processor.ProcessJob(context.Background(), job)
	if err != nil {
		t.Errorf("First ProcessJob failed: %v", err)
	}

	// Second request should be rate limited
	_, err = processor.ProcessJob(context.Background(), job)
	if err == nil {
		t.Error("Second request should be rate limited")
	}
	if err != nil && err.Error() == "" {
		t.Error("Expected rate limit error message")
	}
}

// MockJobQueue is a mock implementation of JobQueue for testing
type MockJobQueue struct {
	EnqueueFunc func(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error)
}

func (m *MockJobQueue) Enqueue(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(ctx, params)
	}
	return &models.LLMJob{ID: 1, UserID: params.UserID}, nil
}

func TestMailClient_SendEmailAsync(t *testing.T) {
	mockQueue := &MockJobQueue{}

	client := &MailClient{
		Host:     "http://localhost:8080",
		Password: "test",
		Testing:  false,
		JobQueue: mockQueue,
	}

	job, err := client.SendEmailAsync(context.Background(), 1, "Test Subject", "test@example.com", "Test Body", false)
	if err != nil {
		t.Errorf("SendEmailAsync failed: %v", err)
	}

	if job == nil {
		t.Error("Expected job to be returned")
	}

	if job != nil && job.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", job.UserID)
	}
}

func TestMailClient_SendEmailAsync_TestingMode(t *testing.T) {
	mockQueue := &MockJobQueue{}
	enqueueCalled := false
	mockQueue.EnqueueFunc = func(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error) {
		enqueueCalled = true
		return &models.LLMJob{ID: 1}, nil
	}

	client := &MailClient{
		Host:     "http://localhost:8080",
		Password: "test",
		Testing:  true, // Testing mode
		JobQueue: mockQueue,
	}

	job, err := client.SendEmailAsync(context.Background(), 1, "Test Subject", "test@example.com", "Test Body", false)
	if err != nil {
		t.Errorf("SendEmailAsync failed: %v", err)
	}

	if job != nil {
		t.Error("Expected nil job in testing mode")
	}

	if enqueueCalled {
		t.Error("Enqueue should not be called in testing mode")
	}

	if client.TestingEmailsSent != 1 {
		t.Errorf("Expected TestingEmailsSent to be 1, got %d", client.TestingEmailsSent)
	}
}

func TestMailClient_SendEmailAsync_NoQueue(t *testing.T) {
	client := &MailClient{
		Host:     "http://localhost:8080",
		Password: "test",
		Testing:  false,
		JobQueue: nil, // No job queue
	}

	_, err := client.SendEmailAsync(context.Background(), 1, "Test Subject", "test@example.com", "Test Body", false)
	if err == nil {
		t.Error("Expected error when job queue is not initialized")
	}
}
