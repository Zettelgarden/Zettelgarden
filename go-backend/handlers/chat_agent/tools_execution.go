package chat_agent

import (
	"context"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"log"

	openai "github.com/sashabaranov/go-openai"
)

// executeToolCall executes a single tool call with simple retry logic
func (s *ChatService) executeToolCall(tc openai.ToolCall, userID int, conversationID, assistantMessageID, model string, loopDetector *services.LoopDetector) map[string]interface{} {
	var args map[string]interface{}
	json.Unmarshal([]byte(tc.Function.Arguments), &args)

	ctx := &services.ToolContext{
		UserID:          userID,
		DB:              s.DB,
		TypesenseClient: s.Server.TypesenseClient,
		ConversationID:  &conversationID,
		MessageID:       &assistantMessageID,
		Model:           model,
		Context:         context.Background(),
	}

	// Use simplified retry logic (max 1 retry)
	result, err := services.ExecuteToolWithRetry(
		context.Background(),
		tc.Function.Name,
		args,
		s.getToolRegistry(),
		ctx,
	)

	// Increment loop detector counter for each tool call
	if loopDetector != nil {
		loopDetector.Increment()
	}

	// Handle execution error
	if err != nil {
		log.Printf("Tool %s: failed - %v", tc.Function.Name, err)
		return services.WrapToolError(tc.Function.Name, args, err)
	}

	return result
}

// executeAndSaveToolCalls executes all tool calls and saves their responses
func (s *ChatService) executeAndSaveToolCalls(toolCalls []openai.ToolCall, userID int, conversationID, assistantMessageID, model string, loopDetector *services.LoopDetector) error {
	for _, tc := range toolCalls {
		result := s.executeToolCall(tc, userID, conversationID, assistantMessageID, model, loopDetector)

		// Save tool response message
		if err := s.SaveToolResponse(conversationID, tc.ID, result); err != nil {
			log.Printf("Error saving tool response: %v", err)
		}
	}

	return nil
}

// executeAndBroadcastToolCalls executes tool calls and sends result events via callback
func (s *ChatService) executeAndBroadcastToolCalls(
	toolCalls []openai.ToolCall,
	userID int,
	conversationID, assistantMessageID, model string,
	sendEvent func(string, interface{}) error,
	loopDetector *services.LoopDetector,
) error {
	for _, tc := range toolCalls {
		result := s.executeToolCall(tc, userID, conversationID, assistantMessageID, model, loopDetector)

		// Check if result contains an error
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		hasError := models.HasError(result)

		// Send enhanced tool result event
		eventData := map[string]interface{}{
			"tool_call_id": tc.ID,
			"name":         tc.Function.Name,
			"result":       result,
			"has_error":    hasError,
			"arguments":    args,
		}
		sendEvent("tool_result", eventData)

		// Save tool response
		if err := s.SaveToolResponse(conversationID, tc.ID, result); err != nil {
			log.Printf("Error saving tool response: %v", err)
		}
	}

	return nil
}
