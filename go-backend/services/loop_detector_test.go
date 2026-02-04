package services

import (
	"fmt"
	"testing"
)

// TestIdenticalCallDetection verifies that calling the same tool with the same params 3+ times is blocked
func TestIdenticalCallDetection(t *testing.T) {
	ld := NewLoopDetector()

	// Make 3 identical calls
	params := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 3; i++ {
		ld.RecordCall("search_cards", params, result, false)
	}

	// Should now block
	shouldBlock, reason, msg := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected identical call detection to block, but it didn't")
	}
	if reason != "identical_calls" {
		t.Errorf("Expected reason 'identical_calls', got '%s'", reason)
	}
	if msg == "" {
		t.Errorf("Expected intervention message, got empty string")
	}
}

// TestIdenticalCallNotTriggeredTooSoon verifies that 2 identical calls don't trigger the block
func TestIdenticalCallNotTriggeredTooSoon(t *testing.T) {
	ld := NewLoopDetector()

	// Make 2 identical calls
	params := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"cards": []interface{}{}}

	ld.RecordCall("search_cards", params, result, false)
	ld.RecordCall("search_cards", params, result, false)

	// Should NOT block yet
	shouldBlock, _, _ := ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected identical call detection to NOT block after 2 calls, but it did")
	}
}

// TestPingPongPatternDetection verifies that alternating between two tools is detected
func TestPingPongPatternDetection(t *testing.T) {
	ld := NewLoopDetector()

	// Create A->B->A->B->A->B->A->B pattern (8 calls)
	// Use DIFFERENT tools and DIFFERENT params each time to avoid identical call detection
	result := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 8; i++ {
		if i%2 == 0 {
			params := map[string]interface{}{"id": fmt.Sprintf("card%d", i)}
			ld.RecordCall("get_card", params, result, false)
		} else {
			params := map[string]interface{}{"query": fmt.Sprintf("search%d", i)}
			ld.RecordCall("search_cards", params, result, false)
		}
	}

	// Should block
	shouldBlock, reason, msg := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected ping-pong pattern detection to block, but it didn't")
	}
	if reason != "ping_pong" {
		t.Errorf("Expected reason 'ping_pong', got '%s'", reason)
	}
	if msg == "" {
		t.Errorf("Expected intervention message, got empty string")
	}
}

// TestPingPongNotTriggeredTooSoon verifies that ping-pong pattern needs enough calls
func TestPingPongNotTriggeredTooSoon(t *testing.T) {
	ld := NewLoopDetector()

	// Create A->B->A->B pattern (only 4 calls) - use different tools and different params
	result := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 4; i++ {
		if i%2 == 0 {
			params := map[string]interface{}{"id": fmt.Sprintf("card%d", i)}
			ld.RecordCall("get_card", params, result, false)
		} else {
			params := map[string]interface{}{"query": fmt.Sprintf("search%d", i)}
			ld.RecordCall("search_cards", params, result, false)
		}
	}

	// Should NOT block yet
	shouldBlock, _, _ := ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected ping-pong detection to NOT block after 4 calls, but it did")
	}
}

// TestConsecutiveErrorDetection verifies that 5+ consecutive errors trigger a block
func TestConsecutiveErrorDetection(t *testing.T) {
	ld := NewLoopDetector()

	// Make 5 consecutive error calls with different params to avoid identical call detection
	errorResult := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "not_found",
			"message": "card not found",
		},
	}

	for i := 0; i < 5; i++ {
		params := map[string]interface{}{"id": fmt.Sprintf("card%d", i)}
		ld.RecordCall("get_card", params, errorResult, true)
	}

	// Should block
	shouldBlock, reason, msg := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected consecutive error detection to block, but it didn't")
	}
	if reason != "consecutive_errors" {
		t.Errorf("Expected reason 'consecutive_errors', got '%s'", reason)
	}
	if msg == "" {
		t.Errorf("Expected intervention message, got empty string")
	}
}

// TestConsecutiveErrorsNotTriggeredTooSoon verifies that consecutive errors need 5 to trigger
func TestConsecutiveErrorsNotTriggeredTooSoon(t *testing.T) {
	ld := NewLoopDetector()

	// Make 4 consecutive error calls with different params
	errorResult := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "not_found",
			"message": "card not found",
		},
	}

	for i := 0; i < 4; i++ {
		params := map[string]interface{}{"id": fmt.Sprintf("card%d", i)}
		ld.RecordCall("get_card", params, errorResult, true)
	}

	// Should NOT block yet
	shouldBlock, _, _ := ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected consecutive error detection to NOT block after 4 calls, but it did")
	}
}

// TestConsecutiveErrorsBrokenBySuccess verifies that a success breaks the consecutive error streak
func TestConsecutiveErrorsBrokenBySuccess(t *testing.T) {
	ld := NewLoopDetector()

	errorResult := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "not_found",
			"message": "card not found",
		},
	}
	successResult := map[string]interface{}{"id": "123", "title": "Test Card"}

	// 3 errors, then success, then 3 more errors (all with different params)
	for i := 0; i < 3; i++ {
		params := map[string]interface{}{"id": fmt.Sprintf("card%d", i)}
		ld.RecordCall("get_card", params, errorResult, true)
	}
	params := map[string]interface{}{"id": "123"}
	ld.RecordCall("get_card", params, successResult, false)
	for i := 0; i < 3; i++ {
		params := map[string]interface{}{"id": fmt.Sprintf("card%d", i+3)}
		ld.RecordCall("get_card", params, errorResult, true)
	}

	// Should NOT block (consecutive errors broken by success)
	shouldBlock, _, _ := ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected consecutive error detection to NOT block when streak is broken, but it did")
	}
}

// TestNoProgressDetection verifies that calling the same tool with different params but empty results triggers block
func TestNoProgressDetection(t *testing.T) {
	ld := NewLoopDetector()

	// Make 6 calls to the same tool with different params, all returning empty results
	emptyResult := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 6; i++ {
		params := map[string]interface{}{"query": string(rune('a' + i))}
		ld.RecordCall("search_cards", params, emptyResult, false)
	}

	// Should block
	shouldBlock, reason, msg := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected no-progress detection to block, but it didn't")
	}
	if reason != "no_progress" {
		t.Errorf("Expected reason 'no_progress', got '%s'", reason)
	}
	if msg == "" {
		t.Errorf("Expected intervention message, got empty string")
	}
}

// TestNoProgressNotTriggeredByErrors verifies that errors don't trigger no-progress detection
func TestNoProgressNotTriggeredByErrors(t *testing.T) {
	ld := NewLoopDetector()

	// Make 6 calls that return errors (not empty results)
	errorResult := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "not_found",
			"message": "card not found",
		},
	}

	for i := 0; i < 6; i++ {
		params := map[string]interface{}{"id": string(rune('a' + i))}
		ld.RecordCall("get_card", params, errorResult, true)
	}

	// This should trigger consecutive errors, not no-progress
	shouldBlock, reason, _ := ld.ShouldBlock()
	if shouldBlock && reason == "no_progress" {
		t.Errorf("Expected no-progress detection to NOT trigger on errors, but it did")
	}
}

// TestValidWorkNotBlocked verifies that normal operations are not blocked
func TestValidWorkNotBlocked(t *testing.T) {
	ld := NewLoopDetector()

	// Simulate a normal conversation with variety
	// Use different tools and different params to avoid triggering detection
	tools := []string{"search_cards", "get_card", "search_cards", "create_card", "get_card"}
	results := []map[string]interface{}{
		{"cards": []interface{}{}},                                    // empty but we're mixing tools
		{"id": "123", "title": "Test Card"},                          // success
		{"cards": []interface{}{"card1", "card2"}},                   // success
		{"id": "456", "title": "New Card"},                           // success
		{"id": "789", "title": "Another Card"},                       // success
	}

	for i, tool := range tools {
		// Use unique params for each call to avoid identical call detection
		params := map[string]interface{}{"id": fmt.Sprintf("param%d", i)}
		ld.RecordCall(tool, params, results[i], false)
	}

	// Should NOT block
	shouldBlock, _, _ := ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected valid work to NOT be blocked, but it was")
	}
}

// TestReset verifies that resetting the detector clears the history
func TestReset(t *testing.T) {
	ld := NewLoopDetector()

	// Make some calls that would trigger a block
	params := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 3; i++ {
		ld.RecordCall("search_cards", params, result, false)
	}

	// Should block
	shouldBlock, _, _ := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected block before reset")
	}

	// Reset
	ld.Reset()

	// Should NOT block anymore
	shouldBlock, _, _ = ld.ShouldBlock()
	if shouldBlock {
		t.Errorf("Expected NO block after reset")
	}

	// Iteration count should be reset
	if ld.GetIterationCount() != 0 {
		t.Errorf("Expected iteration count to be 0 after reset, got %d", ld.GetIterationCount())
	}
}

// TestHistoryCleanup verifies that old records are removed when maxHistory is exceeded
func TestHistoryCleanup(t *testing.T) {
	ld := NewLoopDetector()

	// Make more calls than maxHistory (50)
	params := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"cards": []interface{}{}}

	for i := 0; i < 60; i++ {
		ld.RecordCall("search_cards", params, result, false)
	}

	// Iteration count should still be correct (not trimmed)
	if ld.GetIterationCount() != 60 {
		t.Errorf("Expected iteration count 60, got %d", ld.GetIterationCount())
	}

	// Make 3 identical calls and verify they're detected
	// (This works because history maintains recent calls)
	ld.RecordCall("search_cards", params, result, false)
	ld.RecordCall("search_cards", params, result, false)
	ld.RecordCall("search_cards", params, result, false)

	shouldBlock, _, _ := ld.ShouldBlock()
	if !shouldBlock {
		t.Errorf("Expected identical call detection to work after history cleanup")
	}
}

// TestGetIterationCount verifies the iteration counter works correctly
func TestGetIterationCount(t *testing.T) {
	ld := NewLoopDetector()

	if ld.GetIterationCount() != 0 {
		t.Errorf("Expected initial iteration count 0, got %d", ld.GetIterationCount())
	}

	params := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"cards": []interface{}{}}

	ld.RecordCall("search_cards", params, result, false)
	if ld.GetIterationCount() != 1 {
		t.Errorf("Expected iteration count 1, got %d", ld.GetIterationCount())
	}

	ld.RecordCall("get_card", params, result, false)
	if ld.GetIterationCount() != 2 {
		t.Errorf("Expected iteration count 2, got %d", ld.GetIterationCount())
	}
}

// TestHashParamsConsistency verifies that hashing the same params produces the same hash
func TestHashParamsConsistency(t *testing.T) {
	params1 := map[string]interface{}{
		"query": "test",
		"limit": 10,
	}
	params2 := map[string]interface{}{
		"limit": 10,
		"query": "test",
	}

	hash1 := hashParams(params1)
	hash2 := hashParams(params2)

	if hash1 != hash2 {
		t.Errorf("Expected same hash for same params regardless of order, got %s and %s", hash1, hash2)
	}
}

// TestHashParamsDifferentiates verifies that different params produce different hashes
func TestHashParamsDifferentiates(t *testing.T) {
	params1 := map[string]interface{}{"query": "test"}
	params2 := map[string]interface{}{"query": "different"}

	hash1 := hashParams(params1)
	hash2 := hashParams(params2)

	if hash1 == hash2 {
		t.Errorf("Expected different hashes for different params")
	}
}

// TestIsToolResultEmptyForDetector verifies emptiness detection
func TestIsToolResultEmptyForDetector(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected bool
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: true,
		},
		{
			name:     "empty map",
			result:   map[string]interface{}{},
			expected: true,
		},
		{
			name:     "empty array",
			result:   map[string]interface{}{"cards": []interface{}{}},
			expected: true,
		},
		{
			name:     "empty string",
			result:   map[string]interface{}{"content": ""},
			expected: true,
		},
		{
			name:     "whitespace only string",
			result:   map[string]interface{}{"content": "   "},
			expected: true,
		},
		{
			name:     "error only (should be empty for no-progress check)",
			result:   map[string]interface{}{"error": "something went wrong"},
			expected: true,
		},
		{
			name:     "non-empty array",
			result:   map[string]interface{}{"cards": []interface{}{"card1"}},
			expected: false,
		},
		{
			name:     "non-empty string",
			result:   map[string]interface{}{"content": "hello"},
			expected: false,
		},
		{
			name:     "non-empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{"key": "value"}},
			expected: false,
		},
		{
			name:     "mixed content with empty array but has string",
			result:   map[string]interface{}{"cards": []interface{}{}, "message": "found"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolResultEmptyForDetector(tt.result)
			if result != tt.expected {
				t.Errorf("isToolResultEmptyForDetector() = %v, want %v", result, tt.expected)
			}
		})
	}
}
