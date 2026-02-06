// Package chat provides tool execution orchestration for the chat service.
package chat

import (
	"context"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// ToolExecutionMetrics tracks execution statistics for tool calls.
type ToolExecutionMetrics struct {
	TotalCalls      int
	SuccessfulCalls int
	FailedCalls     int
	TotalDuration   time.Duration
	CallDurations   map[string]time.Duration
}

// ExecuteToolCalls executes a list of tool calls and saves results to the database.
func (s *Service) ExecuteToolCalls(
	toolCalls []openai.ToolCall,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	model string,
	loopDetector *services.LoopDetector,
) error {
	metrics := &ToolExecutionMetrics{
		CallDurations: make(map[string]time.Duration),
	}

	ctx := &services.ToolContext{
		UserID:          userID,
		DB:              s.db,
		TypesenseClient: s.typesenseClient,
		ConversationID:  &conversation.ID,
		MessageID:       &assistantMessageID,
		Model:           model,
		Context:         context.Background(),
	}

	for _, tc := range toolCalls {
		startTime := time.Now()

		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result, err := services.ExecuteToolWithRetry(
			context.Background(),
			tc.Function.Name,
			args,
			s.registry,
			ctx,
		)

		duration := time.Since(startTime)
		metrics.TotalCalls++
		metrics.TotalDuration += duration
		metrics.CallDurations[tc.Function.Name] = duration

		if err != nil {
			s.logError("Tool %s: failed - %v", tc.Function.Name, err)
			metrics.FailedCalls++
		} else {
			metrics.SuccessfulCalls++
		}

		if loopDetector != nil {
			loopDetector.Increment()
		}

		// Save tool response to database
		resultJSON, _ := json.Marshal(result)
		resultStr := string(resultJSON)
		query := `
			INSERT INTO chat_messages (id, conversation_id, role, content, tool_call_id, status, created_at)
			VALUES (gen_random_uuid(), $1, 'tool', $2, $3, 'completed', NOW())
		`
		_, err = s.db.Exec(query, conversation.ID, resultStr, tc.ID)
		if err != nil {
			s.logError("Error saving tool response: %v", err)
		}
	}

	s.logToolExecutionMetrics(metrics)
	return nil
}

// ExecuteToolCallsWithBroadcast executes tool calls and broadcasts results via SSE.
func (s *Service) ExecuteToolCallsWithBroadcast(
	toolCalls []openai.ToolCall,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	model string,
	sendEvent func(string, interface{}) error,
	loopDetector *services.LoopDetector,
) error {
	metrics := &ToolExecutionMetrics{
		CallDurations: make(map[string]time.Duration),
	}

	ctx := &services.ToolContext{
		UserID:          userID,
		DB:              s.db,
		TypesenseClient: s.typesenseClient,
		ConversationID:  &conversation.ID,
		MessageID:       &assistantMessageID,
		Model:           model,
		Context:         context.Background(),
	}

	for _, tc := range toolCalls {
		startTime := time.Now()

		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result, err := services.ExecuteToolWithRetry(
			context.Background(),
			tc.Function.Name,
			args,
			s.registry,
			ctx,
		)

		duration := time.Since(startTime)
		metrics.TotalCalls++
		metrics.TotalDuration += duration
		metrics.CallDurations[tc.Function.Name] = duration

		if err != nil {
			s.logError("Tool %s: failed - %v", tc.Function.Name, err)
			metrics.FailedCalls++
		} else {
			metrics.SuccessfulCalls++
		}

		if loopDetector != nil {
			loopDetector.Increment()
		}

		// Check for error in result
		hasError := models.HasError(result)

		// Send enhanced tool result event
		eventData := map[string]interface{}{
			"tool_call_id": tc.ID,
			"name":         tc.Function.Name,
			"result":       result,
			"has_error":    hasError,
			"arguments":    args,
			"timestamp":    time.Now().Format(time.RFC3339),
			"duration_ms":  duration.Milliseconds(),
		}
		sendEvent("tool_result", eventData)

		// Save tool response to database
		resultJSON, _ := json.Marshal(result)
		resultStr := string(resultJSON)
		query := `
			INSERT INTO chat_messages (id, conversation_id, role, content, tool_call_id, status, created_at)
			VALUES (gen_random_uuid(), $1, 'tool', $2, $3, 'completed', NOW())
		`
		_, err = s.db.Exec(query, conversation.ID, resultStr, tc.ID)
		if err != nil {
			s.logError("Error saving tool response: %v", err)
		}
	}

	s.logToolExecutionMetrics(metrics)
	return nil
}

// logToolExecutionMetrics logs execution metrics for tool calls.
func (s *Service) logToolExecutionMetrics(metrics *ToolExecutionMetrics) {
	if metrics.TotalCalls == 0 {
		return
	}

	avgDuration := metrics.TotalDuration / time.Duration(metrics.TotalCalls)
	s.logInfo("[ToolMetrics] Calls: %d, Success: %d, Failed: %d, Avg: %v",
		metrics.TotalCalls,
		metrics.SuccessfulCalls,
		metrics.FailedCalls,
		avgDuration)

	for toolName, duration := range metrics.CallDurations {
		s.logInfo("[ToolMetrics] %s: %v", toolName, duration)
	}
}

// executeToolCalls is an internal helper (kept for backward compatibility).
func (s *Service) executeToolCalls(toolCalls []openai.ToolCall, userID int, conversation *models.ChatConversation, assistantMessageID string, model string, loopDetector *services.LoopDetector) error {
	return s.ExecuteToolCalls(toolCalls, userID, conversation, assistantMessageID, model, loopDetector)
}

// executeToolCallsWithBroadcast is an internal helper (kept for backward compatibility).
func (s *Service) executeToolCallsWithBroadcast(toolCalls []openai.ToolCall, userID int, conversation *models.ChatConversation, assistantMessageID string, model string, sendEvent func(string, interface{}) error, loopDetector *services.LoopDetector) error {
	return s.ExecuteToolCallsWithBroadcast(toolCalls, userID, conversation, assistantMessageID, model, sendEvent, loopDetector)
}
