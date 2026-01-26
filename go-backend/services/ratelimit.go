package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// JobRateLimiter manages rate limiting for job submissions
// It tracks both per-user and global concurrent job limits
type JobRateLimiter struct {
	mu               sync.RWMutex
	userJobCounts    map[int]*userJobInfo
	globalJobCount   int64
	maxUserJobs      int
	maxGlobalJobs    int
	proUserMultiplier int
	db               *sql.DB
	lastCleanup      time.Time
	cleanupInterval  time.Duration
}

type userJobInfo struct {
	activeCount int
	lastUpdate  time.Time
	isPro       bool
}

// JobRateLimitResult represents the result of a rate limit check
type JobRateLimitResult struct {
	Allowed      bool
	UserCount    int
	GlobalCount  int
	UserLimit    int
	GlobalLimit  int
	RetryAfter   time.Duration
	Reason       string
}

// NewJobRateLimiter creates a new job rate limiter
func NewJobRateLimiter(db *sql.DB) *JobRateLimiter {
	maxUserJobs := getEnvInt("MAX_JOBS_PER_USER", 10)
	maxGlobalJobs := getEnvInt("MAX_GLOBAL_JOBS", 50)

	return &JobRateLimiter{
		userJobCounts:    make(map[int]*userJobInfo),
		maxUserJobs:      maxUserJobs,
		maxGlobalJobs:    maxGlobalJobs,
		proUserMultiplier: 3, // PRO users get 3x the limit
		db:               db,
		lastCleanup:      time.Now(),
		cleanupInterval:  5 * time.Minute,
	}
}

// CheckRateLimit checks if a user is allowed to submit a new job
// It considers both per-user and global concurrent job limits
func (rl *JobRateLimiter) CheckRateLimit(ctx context.Context, userID int, isProUser bool) *JobRateLimitResult {
	rl.cleanupStaleEntries()

	// Get current active job counts
	userLimit := rl.maxUserJobs
	if isProUser {
		userLimit = rl.maxUserJobs * rl.proUserMultiplier
	}

	userCount, globalCount := rl.getJobCounts(ctx, userID)

	result := &JobRateLimitResult{
		UserCount:   userCount,
		GlobalCount: globalCount,
		UserLimit:   userLimit,
		GlobalLimit: rl.maxGlobalJobs,
		Allowed:     true,
	}

	// Check per-user limit
	if userCount >= userLimit {
		result.Allowed = false
		result.Reason = fmt.Sprintf("per-user job limit reached (%d/%d)", userCount, userLimit)
		return result
	}

	// Check global limit
	if globalCount >= rl.maxGlobalJobs {
		result.Allowed = false
		result.Reason = fmt.Sprintf("global job limit reached (%d/%d)", globalCount, rl.maxGlobalJobs)
		return result
	}

	return result
}

// RecordJobSubmission records a new job submission
func (rl *JobRateLimiter) RecordJobSubmission(userID int, isProUser bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	info, exists := rl.userJobCounts[userID]
	if !exists {
		info = &userJobInfo{
			isPro:      isProUser,
			lastUpdate: time.Now(),
		}
		rl.userJobCounts[userID] = info
	}

	info.activeCount++
	info.lastUpdate = time.Now()
	atomic.AddInt64(&rl.globalJobCount, 1)

	log.Printf("[RateLimiter] Job submitted: user=%d, user_count=%d, global_count=%d",
		userID, info.activeCount, rl.globalJobCount)
}

// RecordJobCompletion records a job completion
func (rl *JobRateLimiter) RecordJobCompletion(userID int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	info, exists := rl.userJobCounts[userID]
	if !exists {
		return
	}

	info.activeCount--
	if info.activeCount < 0 {
		info.activeCount = 0
	}
	info.lastUpdate = time.Now()

	newCount := atomic.AddInt64(&rl.globalJobCount, -1)
	if newCount < 0 {
		atomic.StoreInt64(&rl.globalJobCount, 0)
	}

	log.Printf("[RateLimiter] Job completed: user=%d, user_count=%d, global_count=%d",
		userID, info.activeCount, rl.globalJobCount)
}

// getJobCounts gets the current active job counts for a user and globally
// It combines in-memory tracking with database queries for accuracy
func (rl *JobRateLimiter) getJobCounts(ctx context.Context, userID int) (int, int) {
	rl.mu.RLock()
	info, exists := rl.userJobCounts[userID]
	userMemoryCount := 0
	if exists {
		userMemoryCount = info.activeCount
	}
	rl.mu.RUnlock()

	// If DB is not available, use in-memory counts only
	if rl.db == nil {
		return userMemoryCount, int(atomic.LoadInt64(&rl.globalJobCount))
	}

	// Query database for active jobs to get accurate count
	// This ensures we don't miss jobs that were created before server restart
	var userDBCount, globalDBCount int

	// Count pending and running jobs for user
	err := rl.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM llm_jobs WHERE user_id = $1 AND status IN ('pending', 'running')",
		userID).Scan(&userDBCount)
	if err != nil {
		log.Printf("[RateLimiter] Error querying user job count: %v", err)
		userDBCount = userMemoryCount
	}

	// Count total pending and running jobs globally
	err = rl.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM llm_jobs WHERE status IN ('pending', 'running')").
		Scan(&globalDBCount)
	if err != nil {
		log.Printf("[RateLimiter] Error querying global job count: %v", err)
		globalDBCount = int(atomic.LoadInt64(&rl.globalJobCount))
	}

	// Return the maximum of in-memory and database counts to be safe
	userCount := userMemoryCount
	if userDBCount > userCount {
		userCount = userDBCount
	}

	return userCount, globalDBCount
}

// cleanupStaleEntries removes stale entries from the in-memory tracking
func (rl *JobRateLimiter) cleanupStaleEntries() {
	now := time.Now()
	if now.Sub(rl.lastCleanup) < rl.cleanupInterval {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for userID, info := range rl.userJobCounts {
		// Remove entries that haven't been updated in 10 minutes
		if now.Sub(info.lastUpdate) > 10*time.Minute {
			delete(rl.userJobCounts, userID)
		}
	}

	rl.lastCleanup = now
}

// GetStats returns current rate limiter statistics
func (rl *JobRateLimiter) GetStats(ctx context.Context) map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	globalDBCount := 0
	if rl.db != nil {
		err := rl.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM llm_jobs WHERE status IN ('pending', 'running')").
			Scan(&globalDBCount)
		if err != nil {
			log.Printf("[RateLimiter] Error querying global job count for stats: %v", err)
			globalDBCount = int(atomic.LoadInt64(&rl.globalJobCount))
		}
	} else {
		globalDBCount = int(atomic.LoadInt64(&rl.globalJobCount))
	}

	return map[string]interface{}{
		"global_active_jobs":  globalDBCount,
		"global_limit":        rl.maxGlobalJobs,
		"user_limit_standard": rl.maxUserJobs,
		"user_limit_pro":      rl.maxUserJobs * rl.proUserMultiplier,
		"tracked_users":       len(rl.userJobCounts),
	}
}

// SetJobRateLimitHeaders sets rate limit headers on HTTP response
func SetJobRateLimitHeaders(w http.ResponseWriter, result *JobRateLimitResult) {
	if result == nil {
		return
	}

	// Standard rate limit headers (similar to Stripe, GitHub, etc.)
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.UserLimit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.UserLimit-result.UserCount))
	w.Header().Set("X-RateLimit-Used", fmt.Sprintf("%d", result.UserCount))

	// Global limit headers
	w.Header().Set("X-RateLimit-Global-Limit", fmt.Sprintf("%d", result.GlobalLimit))
	w.Header().Set("X-RateLimit-Global-Remaining", fmt.Sprintf("%d", result.GlobalLimit-result.GlobalCount))
	w.Header().Set("X-RateLimit-Global-Used", fmt.Sprintf("%d", result.GlobalCount))

	// Retry-After header if rate limited
	if !result.Allowed && result.RetryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
	}
}
