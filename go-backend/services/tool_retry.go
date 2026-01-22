package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

// RetryConfig defines the retry behavior for tool calls
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	BaseDelay       time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Multiplier for exponential backoff
	TimeoutPerTry   time.Duration // Timeout for each individual tool call
	OverallTimeout  time.Duration // Overall timeout for all retries combined
}

// DefaultRetryConfig returns sensible defaults for tool retry behavior
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		BaseDelay:       500 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		BackoffFactor:   2.0,
		TimeoutPerTry:   30 * time.Second,
		OverallTimeout:  2 * time.Minute,
	}
}

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota // Normal operation
	CircuitOpen                                // Failing, reject requests
	CircuitHalfOpen                            // Testing if service has recovered
)

// CircuitBreaker implements the circuit breaker pattern for tool calls
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	lastStateChange   time.Time

	// Configuration
	FailureThreshold  int           // Number of failures before opening
	SuccessThreshold  int           // Number of successes to close half-open
	OpenTimeout       time.Duration // How long to stay open before half-open
}

// ToolCircuitBreaker manages circuit breakers for individual tools
type ToolCircuitBreaker struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewToolCircuitBreaker creates a new circuit breaker manager
func NewToolCircuitBreaker() *ToolCircuitBreaker {
	return &ToolCircuitBreaker{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// getOrCreateBreaker gets or creates a circuit breaker for a tool
func (tcb *ToolCircuitBreaker) getOrCreateBreaker(toolName string) *CircuitBreaker {
	tcb.mu.Lock()
	defer tcb.mu.Unlock()

	if cb, exists := tcb.breakers[toolName]; exists {
		return cb
	}

	cb := &CircuitBreaker{
		state:            CircuitClosed,
		FailureThreshold: 5,                // Open after 5 consecutive failures
		SuccessThreshold: 2,                // Close after 2 consecutive successes
		OpenTimeout:      60 * time.Second, // Stay open for 60 seconds
	}

	tcb.breakers[toolName] = cb
	return cb
}

// CanExecute checks if a tool call should be allowed based on circuit breaker state
func (tcb *ToolCircuitBreaker) CanExecute(toolName string) bool {
	cb := tcb.getOrCreateBreaker(toolName)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If open and timeout has passed, transition to half-open
	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.OpenTimeout {
		log.Printf("Circuit breaker for %s: OPEN -> HALF_OPEN (timeout elapsed)", toolName)
		cb.state = CircuitHalfOpen
		cb.lastStateChange = time.Now()
		return true
	}

	return cb.state != CircuitOpen
}

// RecordSuccess records a successful tool call
func (tcb *ToolCircuitBreaker) RecordSuccess(toolName string) {
	cb := tcb.getOrCreateBreaker(toolName)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0 // Reset failure count on success

	if cb.state == CircuitHalfOpen {
		cb.successCount++
		log.Printf("Circuit breaker for %s: success count %d/%d", toolName, cb.successCount, cb.SuccessThreshold)

		// If we've had enough successes in half-open, close the circuit
		if cb.successCount >= cb.SuccessThreshold {
			log.Printf("Circuit breaker for %s: HALF_OPEN -> CLOSED", toolName)
			cb.state = CircuitClosed
			cb.lastStateChange = time.Now()
			cb.successCount = 0
		}
	} else if cb.state == CircuitClosed {
		// In closed state, just track for monitoring
		cb.successCount++
	}
}

// RecordFailure records a failed tool call
func (tcb *ToolCircuitBreaker) RecordFailure(toolName string, err error) {
	cb := tcb.getOrCreateBreaker(toolName)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	log.Printf("Circuit breaker for %s: failure count %d/%d (state: %v)", toolName, cb.failureCount, cb.FailureThreshold, cb.state)

	// If we're in closed state and hit threshold, open the circuit
	if cb.state == CircuitClosed && cb.failureCount >= cb.FailureThreshold {
		log.Printf("Circuit breaker for %s: CLOSED -> OPEN (threshold reached)", toolName)
		cb.state = CircuitOpen
		cb.lastStateChange = time.Now()
		return
	}

	// If we're in half-open state, any failure opens the circuit immediately
	if cb.state == CircuitHalfOpen {
		log.Printf("Circuit breaker for %s: HALF_OPEN -> OPEN (failed during test)", toolName)
		cb.state = CircuitOpen
		cb.lastStateChange = time.Now()
		cb.failureCount = cb.FailureThreshold // Max out failures
		cb.successCount = 0
	}
}

// GetState returns the current state and stats for a tool's circuit breaker
func (tcb *ToolCircuitBreaker) GetState(toolName string) (CircuitBreakerState, int, int) {
	cb := tcb.getOrCreateBreaker(toolName)
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state, cb.failureCount, cb.successCount
}

// calculateBackoff calculates the delay for a given retry attempt using exponential backoff with jitter
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	if attempt == 0 {
		return 0
	}

	// Calculate exponential backoff
	delay := float64(config.BaseDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter (±25%)
	jitter := delay * 0.25 * (2*randomFloat() - 1)
	delay += jitter

	return time.Duration(delay)
}

// randomFloat returns a random float between 0 and 1
func randomFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// executeToolWithContext executes a tool with context support
func executeToolWithContext(ctx context.Context, toolRegistry *ToolRegistry, toolName string, args map[string]interface{}, toolCtx *ToolContext) (map[string]interface{}, error) {
	// Create a channel to receive the result
	resultChan := make(chan struct {
		result map[string]interface{}
		err    error
	}, 1)

	// Execute the tool in a goroutine
	go func() {
		result, err := toolRegistry.ExecuteTool(toolName, args, toolCtx)
		resultChan <- struct {
			result map[string]interface{}
			err    error
		}{result, err}
	}()

	// Wait for result or context cancellation
	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool execution timed out or was cancelled")
	}
}

// ToolExecutionResult represents the result of a tool execution attempt
type ToolExecutionResult struct {
	Result       map[string]interface{}
	Error        error
	Attempts     int
	TotalTime    time.Duration
	CircuitOpen  bool
	LastError    string
}

// ExecuteToolWithRetry executes a tool with intelligent retry logic
func ExecuteToolWithRetry(
	ctx context.Context,
	toolName string,
	args map[string]interface{},
	toolRegistry *ToolRegistry,
	toolCtx *ToolContext,
	circuitBreaker *ToolCircuitBreaker,
	config RetryConfig,
) ToolExecutionResult {
	startTime := time.Now()
	result := ToolExecutionResult{
		Attempts: 0,
	}

	// Check circuit breaker first
	if !circuitBreaker.CanExecute(toolName) {
		state, failures, _ := circuitBreaker.GetState(toolName)
		result.CircuitOpen = true
		result.Error = fmt.Errorf("circuit breaker is %v for tool %s (failures: %d)", state, toolName, failures)
		result.LastError = result.Error.Error()
		return result
	}

	// Create overall timeout context
	overallCtx, cancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer cancel()

	var lastError error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Calculate backoff delay (skip on first attempt)
		if attempt > 0 {
			delay := calculateBackoff(attempt, config)
			log.Printf("Tool %s: retry attempt %d after %v delay", toolName, attempt, delay)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-overallCtx.Done():
				result.Error = fmt.Errorf("overall timeout exceeded during backoff")
				result.LastError = result.Error.Error()
				return result
			}
		}

		// Create per-attempt timeout context
		attemptCtx, attemptCancel := context.WithTimeout(overallCtx, config.TimeoutPerTry)

		// Execute the tool
		toolResult, err := executeToolWithContext(attemptCtx, toolRegistry, toolName, args, toolCtx)
		attemptCancel()

		if err == nil {
			// Success!
			result.Result = toolResult
			result.TotalTime = time.Since(startTime)
			circuitBreaker.RecordSuccess(toolName)

			log.Printf("Tool %s: succeeded on attempt %d (total time: %v)", toolName, attempt+1, result.TotalTime)
			return result
		}

		// Record the error
		lastError = err
		result.LastError = err.Error()

		// Check if error is retryable
		if !isRetryableError(err) {
			log.Printf("Tool %s: non-retryable error on attempt %d: %v", toolName, attempt+1, err)
			circuitBreaker.RecordFailure(toolName, err)
			result.Error = err
			result.TotalTime = time.Since(startTime)
			return result
		}

		log.Printf("Tool %s: attempt %d failed with retryable error: %v", toolName, attempt+1, err)
	}

	// All retries exhausted
	circuitBreaker.RecordFailure(toolName, lastError)
	result.Error = fmt.Errorf("tool %s failed after %d attempts: %w", toolName, result.Attempts, lastError)
	result.TotalTime = time.Since(startTime)
	return result
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
