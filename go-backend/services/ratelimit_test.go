package services

import (
	"context"
	"testing"
	"time"
)

func TestJobRateLimiter_CheckRateLimit_Basic(t *testing.T) {
	// Create a mock DB for testing
	// For this test, we'll use a nil DB and rely on in-memory tracking
	limiter := NewJobRateLimiter(nil)

	ctx := context.Background()

	// Standard user (not PRO)
	result := limiter.CheckRateLimit(ctx, 1, false)
	if !result.Allowed {
		t.Error("First request should be allowed")
	}
	if result.UserLimit != 10 {
		t.Errorf("Expected user limit 10, got %d", result.UserLimit)
	}
	if result.GlobalLimit != 50 {
		t.Errorf("Expected global limit 50, got %d", result.GlobalLimit)
	}
}

func TestJobRateLimiter_CheckRateLimit_ProUser(t *testing.T) {
	limiter := NewJobRateLimiter(nil)

	ctx := context.Background()

	// PRO user should get 3x the limit
	result := limiter.CheckRateLimit(ctx, 1, true)
	if !result.Allowed {
		t.Error("First request should be allowed for PRO user")
	}
	if result.UserLimit != 30 {
		t.Errorf("Expected user limit 30 for PRO user, got %d", result.UserLimit)
	}
}

func TestJobRateLimiter_RecordSubmission(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	// Record a submission
	limiter.RecordJobSubmission(1, false)

	// Check stats
	stats := limiter.GetStats(ctx)
	if stats["tracked_users"] != 1 {
		t.Errorf("Expected 1 tracked user, got %v", stats["tracked_users"])
	}
}

func TestJobRateLimiter_RecordCompletion(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	// Record submission and completion
	limiter.RecordJobSubmission(1, false)
	limiter.RecordJobCompletion(1)

	result := limiter.CheckRateLimit(ctx, 1, false)
	// After recording completion, user should still have slot available
	// (actual count comes from DB which is nil, so it will be 0)
	if !result.Allowed {
		t.Error("Request should be allowed after completion")
	}
}

func TestJobRateLimiter_GetStats(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	stats := limiter.GetStats(ctx)

	if stats["user_limit_standard"] != 10 {
		t.Errorf("Expected standard user limit 10, got %v", stats["user_limit_standard"])
	}
	if stats["user_limit_pro"] != 30 {
		t.Errorf("Expected PRO user limit 30, got %v", stats["user_limit_pro"])
	}
	if stats["global_limit"] != 50 {
		t.Errorf("Expected global limit 50, got %v", stats["global_limit"])
	}
}

func TestJobRateLimiter_CleanupStaleEntries(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	// Record a submission for user 1
	limiter.RecordJobSubmission(1, false)

	// Manually set the lastUpdate to a very old time to simulate stale entry
	limiter.mu.Lock()
	if info, exists := limiter.userJobCounts[1]; exists {
		info.lastUpdate = time.Now().Add(-15 * time.Minute)
	}
	limiter.mu.Unlock()

	// Trigger cleanup by calling CheckRateLimit (which calls cleanupStaleEntries internally)
	// But we need to wait for the cleanup interval to pass
	limiter.lastCleanup = time.Now().Add(-10 * time.Minute)
	limiter.CheckRateLimit(ctx, 2, false)

	// Check if user 1 was cleaned up
	limiter.mu.RLock()
	_, exists := limiter.userJobCounts[1]
	limiter.mu.RUnlock()

	// The stale entry should be cleaned up
	// Note: cleanup only removes entries older than 10 minutes, so this should work
	if exists {
		// If still exists, check if lastUpdate was updated
		limiter.mu.RLock()
		info := limiter.userJobCounts[1]
		limiter.mu.RUnlock()
		if info != nil && time.Since(info.lastUpdate) > 10*time.Minute {
			t.Error("Stale entry should have been cleaned up")
		}
	}
}

func TestJobRateLimiter_MultipleUsers(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	// User 1 (standard)
	result1 := limiter.CheckRateLimit(ctx, 1, false)
	if !result1.Allowed {
		t.Error("User 1 first request should be allowed")
	}

	// User 2 (PRO)
	result2 := limiter.CheckRateLimit(ctx, 2, true)
	if !result2.Allowed {
		t.Error("User 2 first request should be allowed")
	}

	// User 1 and User 2 should have independent limits
	if result1.UserLimit != 10 {
		t.Errorf("User 1 should have limit 10, got %d", result1.UserLimit)
	}
	if result2.UserLimit != 30 {
		t.Errorf("User 2 should have limit 30, got %d", result2.UserLimit)
	}
}

func TestJobRateLimiter_ConcurrentTracking(t *testing.T) {
	limiter := NewJobRateLimiter(nil)
	ctx := context.Background()

	// Simulate concurrent job submissions
	for i := 0; i < 5; i++ {
		limiter.RecordJobSubmission(1, false)
	}

	stats := limiter.GetStats(ctx)
	if stats["tracked_users"] != 1 {
		t.Errorf("Expected 1 tracked user, got %v", stats["tracked_users"])
	}

	// Record completions
	for i := 0; i < 5; i++ {
		limiter.RecordJobCompletion(1)
	}

	// User should still be tracked even with 0 active jobs
	stats = limiter.GetStats(ctx)
	if stats["tracked_users"] != 1 {
		t.Errorf("Expected 1 tracked user after completions, got %v", stats["tracked_users"])
	}
}
