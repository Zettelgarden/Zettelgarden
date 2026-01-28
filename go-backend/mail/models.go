package mail

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"sync"
	"time"
)

// JobQueue interface is defined here to avoid import cycles
// It's a subset of the services.JobQueue interface needed for email jobs
type JobQueue interface {
	Enqueue(ctx context.Context, params models.CreateJobParams) (*models.LLMJob, error)
}

// WorkerPool interface is defined here to avoid import cycles
type WorkerPool interface {
	Start() error
	Stop() error
	IsRunning() bool
}

type MailClient struct {
	Host              string
	Password          string
	Testing           bool
	TestingEmailsSent int
	Queue             *EmailQueue
	mu                sync.Mutex
	isProcessing      bool
	DB                *sql.DB
	Tx                models.DBTX
	ShutdownChan      chan struct{}
	shutdownOnce      sync.Once

	// Job queue integration
	JobQueue    JobQueue
	WorkerPool  WorkerPool
	RateLimiter *EmailRateLimiter
}

func (m *MailClient) db() models.DBTX {
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

// EmailProcessor implements the JobProcessor interface for email jobs
type EmailProcessor struct {
	MailClient *MailClient
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(mailClient *MailClient) *EmailProcessor {
	return &EmailProcessor{
		MailClient: mailClient,
	}
}

// ProcessJob implements the JobProcessor interface
// It processes email jobs by sending them via the mail service
func (p *EmailProcessor) ProcessJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract email details from payload
	subject, ok := job.Payload["subject"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid subject in payload")
	}

	recipient, ok := job.Payload["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid recipient in payload")
	}

	body, ok := job.Payload["body"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid body in payload")
	}

	isHTML, _ := job.Payload["is_html"].(bool)
	if !isHTML {
		isHTML = false
	}

	// Check rate limit for this user
	if p.MailClient.RateLimiter != nil {
		allowed, retryAfter := p.MailClient.RateLimiter.Allow(job.UserID)
		if !allowed {
			return nil, fmt.Errorf("rate limit exceeded for user %d, retry after %v", job.UserID, retryAfter)
		}
	}

	// Send the email
	email := Email{
		Subject:   subject,
		Recipient: recipient,
		Body:      body,
		IsHTML:    isHTML,
	}

	err := p.MailClient.sendMailImpl(email)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	// Return success result
	result := map[string]interface{}{
		"recipient": recipient,
		"subject":   subject,
		"sent_at":   time.Now().Format(time.RFC3339),
	}

	log.Printf("[EmailProcessor] Successfully sent email to %s (job %d)", recipient, job.ID)
	return result, nil
}

// EmailRateLimiter implements rate limiting for email sending per user
type EmailRateLimiter struct {
	mu              sync.RWMutex
	userSendCounts  map[int]*userSendInfo
	maxEmailsPerMin int
	maxEmailsPerDay int
}

type userSendInfo struct {
	minuteCount int
	minuteReset time.Time
	dayCount    int
	dayReset    time.Time
}

// NewEmailRateLimiter creates a new email rate limiter
func NewEmailRateLimiter(maxPerMin, maxPerDay int) *EmailRateLimiter {
	return &EmailRateLimiter{
		userSendCounts:  make(map[int]*userSendInfo),
		maxEmailsPerMin: maxPerMin,
		maxEmailsPerDay: maxPerDay,
	}
}

// Allow checks if the user is allowed to send an email
// Returns (allowed, retryAfterDuration)
func (r *EmailRateLimiter) Allow(userID int) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	info, exists := r.userSendCounts[userID]

	if !exists {
		r.userSendCounts[userID] = &userSendInfo{
			minuteCount: 1,
			minuteReset: now.Add(time.Minute),
			dayCount:    1,
			dayReset:    now.Add(24 * time.Hour),
		}
		return true, 0
	}

	// Check and reset minute counter
	if now.After(info.minuteReset) {
		info.minuteCount = 0
		info.minuteReset = now.Add(time.Minute)
	}

	// Check and reset day counter
	if now.After(info.dayReset) {
		info.dayCount = 0
		info.dayReset = now.Add(24 * time.Hour)
	}

	// Check minute limit
	if info.minuteCount >= r.maxEmailsPerMin {
		retryAfter := info.minuteReset.Sub(now)
		return false, retryAfter
	}

	// Check day limit
	if info.dayCount >= r.maxEmailsPerDay {
		retryAfter := info.dayReset.Sub(now)
		return false, retryAfter
	}

	// Increment counters
	info.minuteCount++
	info.dayCount++

	return true, 0
}

// CleanupUserInfo removes user info to free memory
func (r *EmailRateLimiter) CleanupUserInfo(userID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.userSendCounts, userID)
}

// SendEmailAsync sends an email asynchronously using the job queue
func (m *MailClient) SendEmailAsync(ctx context.Context, userID int, subject, recipient, body string, isHTML bool) (*models.LLMJob, error) {
	if m.JobQueue == nil {
		return nil, fmt.Errorf("job queue not initialized")
	}

	// In testing mode, just count the email
	if m.Testing {
		m.TestingEmailsSent++
		return nil, nil
	}

	payload := map[string]interface{}{
		"subject":   subject,
		"recipient": recipient,
		"body":      body,
		"is_html":   isHTML,
	}

	params := models.CreateJobParams{
		UserID:      userID,
		JobType:     models.JobTypeEmail,
		Priority:    5, // Default priority
		Payload:     payload,
		MaxRetries:  3,
		TimeoutSecs: 60, // 1 minute timeout for email sending
	}

	job, err := m.JobQueue.Enqueue(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue email job: %w", err)
	}

	log.Printf("[MailClient] Enqueued email job %d for user %d", job.ID, userID)
	return job, nil
}
