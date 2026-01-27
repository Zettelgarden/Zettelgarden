package handlers

import (
	"database/sql"
	"go-backend/server"
	"go-backend/services"
	"sync"
	"time"
)

const (
	// MaxSummarizationRequestsPerMinute is the maximum number of summarization requests per user per minute
	MaxSummarizationRequestsPerMinute = 10

	// MaxConcurrentSummarizationsPerUser is the maximum number of concurrent summarization jobs per user
	MaxConcurrentSummarizationsPerUser = 3

	// SummarizationRateLimitWindow is the time window for rate limiting (1 minute)
	SummarizationRateLimitWindow = time.Minute
)

type Handler struct {
	DB             *sql.DB
	Server         *server.Server
	ToolRetry      *services.ToolCircuitBreaker
	messageMutexes sync.Map // map[string]*sync.Mutex - per-message mutexes

	// Rate limiting and concurrency control for summarization
	summarizationRateLimits   sync.Map // map[int][]time.Time - request timestamps per user
	summarizationRateLimitMu  sync.Map // map[int]*sync.Mutex - per-user rate limit mutex
	summarizationActiveJobs   sync.Map // map[int]int - active job count per user
	summarizationActiveJobsMu sync.Map // map[int]*sync.Mutex - per-user active jobs mutex

	// Job queue rate limiting
	JobRateLimiter *services.JobRateLimiter

	// LLM worker pool for job processing
	LLMWorkerPool *services.WorkerPool
}

// getMessageMutex gets or creates a mutex for a specific message
func (s *Handler) getMessageMutex(messageID string) *sync.Mutex {
	mu, _ := s.messageMutexes.LoadOrStore(messageID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// cleanupMessageMutex removes a mutex after a message is completed/failed
func (s *Handler) cleanupMessageMutex(messageID string) {
	s.messageMutexes.Delete(messageID)
}

// checkSummarizationRateLimit checks if a user has exceeded their rate limit for summarization requests
func (h *Handler) checkSummarizationRateLimit(userID int) bool {
	// Get or create mutex for this user
	muInt, _ := h.summarizationRateLimitMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-SummarizationRateLimitWindow)

	// Load existing timestamps
	timestampsInt, _ := h.summarizationRateLimits.LoadOrStore(userID, []time.Time{})
	timestamps := timestampsInt.([]time.Time)

	// Filter out timestamps outside the window
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit is exceeded
	if len(validTimestamps) >= MaxSummarizationRequestsPerMinute {
		return false // Rate limit exceeded
	}

	// Add current timestamp and update map
	validTimestamps = append(validTimestamps, now)
	h.summarizationRateLimits.Store(userID, validTimestamps)

	return true // Within rate limit
}

// acquireSummarizationJobSlot attempts to acquire a slot for a new summarization job
// Returns true if successful, false if the user has reached their concurrent job limit
func (h *Handler) acquireSummarizationJobSlot(userID int) bool {
	// Get or create mutex for this user
	muInt, _ := h.summarizationActiveJobsMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// Load current job count
	countInt, _ := h.summarizationActiveJobs.LoadOrStore(userID, 0)
	count := countInt.(int)

	// Check if limit is reached
	if count >= MaxConcurrentSummarizationsPerUser {
		return false
	}

	// Increment count
	h.summarizationActiveJobs.Store(userID, count+1)
	return true
}

// releaseSummarizationJobSlot releases a slot when a summarization job completes
func (h *Handler) releaseSummarizationJobSlot(userID int) {
	// Get or create mutex for this user
	muInt, _ := h.summarizationActiveJobsMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// Load current job count
	countInt, _ := h.summarizationActiveJobs.LoadOrStore(userID, 0)
	count := countInt.(int)

	// Decrement count (but not below 0)
	if count > 0 {
		h.summarizationActiveJobs.Store(userID, count-1)
	}
}
