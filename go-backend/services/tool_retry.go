package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// ExecuteToolWithRetry executes a tool with simple retry logic
// Max 1 retry with 500ms delay for retryable errors only
// Each attempt has a per-attempt timeout to prevent hanging tools
func ExecuteToolWithRetry(
	ctx context.Context,
	toolName string,
	args map[string]interface{},
	toolRegistry *ToolRegistry,
	toolCtx *ToolContext,
) (map[string]interface{}, error) {
	const maxRetries = 1
	const retryDelay = 500 * time.Millisecond
	const attemptTimeout = 30 * time.Second

	var lastError error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Tool %s: retry attempt %d after %v delay", toolName, attempt, retryDelay)
			select {
			case <-time.After(retryDelay):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("tool execution cancelled during retry delay")
			}
		}

		// Create per-attempt timeout context to prevent hanging tools
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		defer cancel()

		// Update toolCtx with the timeout context for this attempt
		originalCtx := toolCtx.Context
		toolCtx.Context = attemptCtx

		// Execute the tool with timeout
		result, err := toolRegistry.ExecuteTool(toolName, args, toolCtx)

		// Restore original context
		toolCtx.Context = originalCtx

		// Check for context timeout first
		if attemptCtx.Err() == context.DeadlineExceeded {
			log.Printf("Tool %s: attempt %d timed out after %v", toolName, attempt+1, attemptTimeout)
			lastError = fmt.Errorf("tool execution timed out after %v", attemptTimeout)
			// Timeouts are retryable - may be transient
			if attempt < maxRetries {
				continue
			}
			// Already at max retries, return error
			return nil, lastError
		}

		if err == nil {
			// Success!
			if attempt > 0 {
				log.Printf("Tool %s: succeeded on attempt %d", toolName, attempt+1)
			}
			return result, nil
		}

		// Record the error
		lastError = err

		// Check if error is retryable
		if !isRetryableError(lastError) {
			log.Printf("Tool %s: non-retryable error on attempt %d: %v", toolName, attempt+1, lastError)
			return nil, lastError
		}

		log.Printf("Tool %s: attempt %d failed with retryable error: %v", toolName, attempt+1, lastError)
	}

	// All retries exhausted
	return nil, fmt.Errorf("tool %s failed after %d attempts: %w", toolName, maxRetries+1, lastError)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// Non-retryable errors - these indicate client/usage errors
	nonRetryablePatterns := []string{
		"invalid",
		"not found",
		"unauthorized",
		"forbidden",
		"validation",
		"required",
		"parameter",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsgLower, pattern) {
			return false
		}
	}

	// Assume retryable for network issues, timeouts, etc.
	return true
}
