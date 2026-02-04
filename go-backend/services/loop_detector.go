package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ToolCallRecord represents a single tool call for loop detection
type ToolCallRecord struct {
	ToolName    string
	ParamsHash  string
	ResultHash  string
	Timestamp   time.Time
	WasError    bool
	ResultEmpty bool
}

// LoopDetector detects and prevents agent loops in tool calling
type LoopDetector struct {
	mu          sync.RWMutex
	recentCalls []ToolCallRecord
	maxHistory  int
	iteration   int // Track current loop iteration
}

// NewLoopDetector creates a new loop detector with the specified history size
func NewLoopDetector() *LoopDetector {
	return &LoopDetector{
		recentCalls: make([]ToolCallRecord, 0, 50),
		maxHistory:  50,
		iteration:   0,
	}
}

// RecordCall adds a tool call record to the history
func (ld *LoopDetector) RecordCall(toolName string, params map[string]interface{}, result map[string]interface{}, wasError bool) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	record := ToolCallRecord{
		ToolName:    toolName,
		ParamsHash:  hashParams(params),
		ResultHash:  hashResult(result),
		Timestamp:   time.Now(),
		WasError:    wasError,
		ResultEmpty: isToolResultEmptyForDetector(result),
	}

	ld.recentCalls = append(ld.recentCalls, record)
	ld.iteration++

	// Trim history if it exceeds max size
	if len(ld.recentCalls) > ld.maxHistory {
		ld.recentCalls = ld.recentCalls[len(ld.recentCalls)-ld.maxHistory:]
	}

	log.Printf("[LoopDetector] Iteration %d: %s (error: %v, empty: %v)", ld.iteration, toolName, wasError, record.ResultEmpty)
}

// ShouldBlock checks if a tool call should be blocked due to looping
// Returns (shouldBlock, reason, interventionMessage)
func (ld *LoopDetector) ShouldBlock() (shouldBlock bool, reason string, interventionMessage string) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	if len(ld.recentCalls) < 2 {
		return false, "", ""
	}

	// Pattern 1: Identical calls (same tool + same params 3+ times)
	if block, msg := ld.checkIdenticalCalls(); block {
		return true, "identical_calls", msg
	}

	// Pattern 2: Ping-pong pattern (alternating between 2 tools)
	if block, msg := ld.checkPingPongPattern(); block {
		return true, "ping_pong", msg
	}

	// Pattern 3: Consecutive errors (5+ tool calls in a row with errors)
	if block, msg := ld.checkConsecutiveErrors(); block {
		return true, "consecutive_errors", msg
	}

	// Pattern 4: No progress (same tool called many times with empty results)
	if block, msg := ld.checkNoProgress(); block {
		return true, "no_progress", msg
	}

	return false, "", ""
}

// checkIdenticalCalls detects when the same tool with the same params is called 3+ times
func (ld *LoopDetector) checkIdenticalCalls() (bool, string) {
	// Look at the last 10 calls for identical patterns
	window := 10
	if len(ld.recentCalls) < window {
		window = len(ld.recentCalls)
	}

	// Count occurrences of each unique (tool, params) combination
	callCounts := make(map[string]int)
	var lastIdenticalTool string

	for i := len(ld.recentCalls) - window; i < len(ld.recentCalls); i++ {
		if i < 0 {
			continue
		}
		record := ld.recentCalls[i]
		key := record.ToolName + ":" + record.ParamsHash
		callCounts[key]++

		if callCounts[key] >= 3 {
			lastIdenticalTool = record.ToolName
			break
		}
	}

	if lastIdenticalTool != "" {
		// Build the intervention message
		var callHistory strings.Builder
		count := 0
		for i := len(ld.recentCalls) - 1; i >= 0 && count < 5; i-- {
			record := ld.recentCalls[i]
			if record.ToolName == lastIdenticalTool {
				callHistory.WriteString(fmt.Sprintf("- %s (params hash: %s...)\n", record.ToolName, record.ParamsHash[:8]))
				count++
			}
		}

		msg := fmt.Sprintf(`SYSTEM INTERVENTION: You appear to be in a loop calling the same tool.

Recent identical calls:
%s

This tool call has been made 3+ times with the same parameters. The result is not changing.

Suggestion:
1. The tool may not be able to complete this request with the current parameters
2. Try a different approach or use a different tool
3. If searching for information, the resource may not exist - try broader search terms
4. Consider asking the user for clarification or additional information
`, callHistory.String())

		return true, msg
	}

	return false, ""
}

// checkPingPongPattern detects alternating between two tools (A->B->A->B...)
func (ld *LoopDetector) checkPingPongPattern() (bool, string) {
	// Need at least 8 calls to detect ping-pong
	if len(ld.recentCalls) < 8 {
		return false, ""
	}

	// Look at the last 8 calls
	window := 8
	startIdx := len(ld.recentCalls) - window

	// Extract the sequence of tool names
	sequence := make([]string, window)
	for i := 0; i < window; i++ {
		sequence[i] = ld.recentCalls[startIdx+i].ToolName
	}

	// Check for A->B->A->B->... pattern
	// Get the two potential tools in the pattern
	toolA := sequence[0]
	toolB := sequence[1]

	// If the first two are the same, it's not a ping-pong
	if toolA == toolB {
		return false, ""
	}

	// Check if the pattern holds
	isPingPong := true
	for i := 0; i < window; i++ {
		expected := toolA
		if i%2 == 1 {
			expected = toolB
		}
		if sequence[i] != expected {
			isPingPong = false
			break
		}
	}

	if isPingPong {
		msg := fmt.Sprintf(`SYSTEM INTERVENTION: You appear to be in a ping-pong loop between two tools.

Recent pattern:
%s

You are alternating between "%s" and "%s" without making progress.

Suggestion:
1. This pattern suggests neither tool can complete the requested operation
2. The approach may be fundamentally flawed - try a different strategy
3. Consider using a different tool that combines both operations
4. Ask the user for clarification about what they're trying to accomplish
`, formatSequence(sequence), toolA, toolB)

		return true, msg
	}

	return false, ""
}

// checkConsecutiveErrors detects when 5+ tool calls in a row return errors
func (ld *LoopDetector) checkConsecutiveErrors() (bool, string) {
	consecutiveErrors := 0
	var lastErrorTool string
	var errorSummary strings.Builder

	// Count consecutive errors from the end
	for i := len(ld.recentCalls) - 1; i >= 0; i-- {
		record := ld.recentCalls[i]
		if record.WasError {
			consecutiveErrors++
			if lastErrorTool == "" {
				lastErrorTool = record.ToolName
			}
			if consecutiveErrors <= 5 {
				errorSummary.WriteString(fmt.Sprintf("- %s: error\n", record.ToolName))
			}
		} else {
			break
		}
	}

	if consecutiveErrors >= 5 {
		msg := fmt.Sprintf(`SYSTEM INTERVENTION: Multiple tool errors detected in sequence.

Recent errors:
%s

You have experienced %d consecutive tool call failures.

Suggestion:
1. The current approach is fundamentally not working
2. There may be an issue with the input parameters or data
3. Try a completely different approach or tool
4. If these are validation errors, carefully check the required parameters
5. Consider explaining the difficulty to the user and asking for guidance
`, errorSummary.String(), consecutiveErrors)

		return true, msg
	}

	return false, ""
}

// checkNoProgress detects when the same tool is called many times with different params but all results are empty
func (ld *LoopDetector) checkNoProgress() (bool, string) {
	// Need at least 6 calls to detect no progress
	if len(ld.recentCalls) < 6 {
		return false, ""
	}

	// Look at the last 12 calls
	window := 12
	if len(ld.recentCalls) < window {
		window = len(ld.recentCalls)
	}
	startIdx := len(ld.recentCalls) - window

	// Count how many times the same tool was called with empty results
	toolEmptyCount := make(map[string]int)
	var lastTool string
	var paramVariations []string

	for i := startIdx; i < len(ld.recentCalls); i++ {
		record := ld.recentCalls[i]
		if record.ResultEmpty && !record.WasError {
			toolEmptyCount[record.ToolName]++
			if toolEmptyCount[record.ToolName] >= 6 {
				lastTool = record.ToolName
				// Collect some param hashes to show variation
				for j := startIdx; j < len(ld.recentCalls); j++ {
					if ld.recentCalls[j].ToolName == lastTool && len(paramVariations) < 5 {
						paramVariations = append(paramVariations, ld.recentCalls[j].ParamsHash[:8])
					}
				}
				break
			}
		}
	}

	if lastTool != "" {
		var callHistory strings.Builder
		for i, hash := range paramVariations {
			if i >= 5 {
				break
			}
			callHistory.WriteString(fmt.Sprintf("- %s (params: %s...)\n", lastTool, hash))
		}

		msg := fmt.Sprintf(`SYSTEM INTERVENTION: No progress detected with repeated tool calls.

Recent calls with empty results:
%s

The tool "%s" has been called %d times with different parameters, but all results are empty.

Suggestion:
1. The resources you're looking for may not exist
2. Try using broader search terms or different criteria
3. Browse the available data using a different tool (e.g., list or browse functions)
4. Ask the user for more specific information or clarification
`, callHistory.String(), lastTool, toolEmptyCount[lastTool])

		return true, msg
	}

	return false, ""
}

// hashParams creates a hash of the parameters for comparison
func hashParams(params map[string]interface{}) string {
	if params == nil {
		return "nil"
	}

	// Sort keys for consistent hashing
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	// Actually sort the keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Create a stable JSON representation
	var sortedParams strings.Builder
	sortedParams.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			sortedParams.WriteString(",")
		}
		sortedParams.WriteString(fmt.Sprintf("%q:", k))
		v := params[k]
		switch val := v.(type) {
		case string:
			sortedParams.WriteString(fmt.Sprintf("%q", val))
		case int, int64, float64, bool:
			sortedParams.WriteString(fmt.Sprintf("%v", val))
		case []interface{}:
			sortedParams.WriteString("[]")
		case map[string]interface{}:
			sortedParams.WriteString("{}")
		default:
			sortedParams.WriteString(fmt.Sprintf("%q", fmt.Sprintf("%v", val)))
		}
	}
	sortedParams.WriteString("}")

	h := sha256.New()
	h.Write([]byte(sortedParams.String()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// hashResult creates a hash of the result for comparison
func hashResult(result map[string]interface{}) string {
	if result == nil {
		return "nil"
	}

	// For results, we care about meaningful content, not exact structure
	// Create a simplified representation
	h := sha256.New()
	jsonBytes, _ := json.Marshal(result)
	h.Write(jsonBytes)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// isToolResultEmptyForDetector checks if a tool result is effectively empty
func isToolResultEmptyForDetector(result map[string]interface{}) bool {
	if result == nil || len(result) == 0 {
		return true
	}

	// Check if result only contains empty values
	for key, value := range result {
		// Skip error field for emptiness check - errors are handled separately
		if key == "error" {
			continue
		}

		// Check various types of emptiness
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return false
			}
		case []interface{}:
			if len(v) > 0 {
				return false
			}
		case map[string]interface{}:
			if len(v) > 0 {
				return false
			}
		default:
			// If there's any other non-nil value, consider it non-empty
			if v != nil {
				return false
			}
		}
	}

	return true
}

// formatSequence creates a formatted string representation of a tool call sequence
func formatSequence(sequence []string) string {
	var result strings.Builder
	for i, tool := range sequence {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, tool))
	}
	return result.String()
}

// Reset clears the loop detector history
func (ld *LoopDetector) Reset() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.recentCalls = make([]ToolCallRecord, 0, ld.maxHistory)
	ld.iteration = 0
	log.Printf("[LoopDetector] Reset")
}

// GetIterationCount returns the current iteration count
func (ld *LoopDetector) GetIterationCount() int {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	return ld.iteration
}
