package chat_agent

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
)

// updateAssistantMessage updates an existing assistant message with the generated content (thread-safe)
func (s *ChatService) updateAssistantMessage(userID int, conversationID, messageID string, generatedMessage *models.ChatMessage) error {
	// Acquire per-message mutex to prevent race conditions on message updates
	mu := s.getMessageMutex(messageID)
	mu.Lock()
	defer mu.Unlock()
	defer s.cleanupMessageMutex(messageID)

	var query string
	var args []interface{}

	// Convert tool calls to JSON if present
	// IMPORTANT: Only update tool_calls if we have tool_calls to set.
	// If toolCallsJSON is nil, we don't want to overwrite existing tool_calls in the DB.
	var toolCallsJSON *string
	if generatedMessage.ToolCalls != nil && len(generatedMessage.ToolCalls) > 0 {
		toolCallsData, err := json.Marshal(generatedMessage.ToolCalls)
		if err != nil {
			return err
		}
		toolCallsStr := string(toolCallsData)
		toolCallsJSON = &toolCallsStr

		// Update both content and tool_calls
		query = `
			UPDATE chat_messages
			SET content = $1, tool_calls = $2, status = 'completed'
			WHERE id = $3
		`
		args = []interface{}{generatedMessage.Content, toolCallsJSON, messageID}
	} else {
		// Only update content, preserve existing tool_calls
		query = `
			UPDATE chat_messages
			SET content = $1, status = 'completed'
			WHERE id = $2
		`
		args = []interface{}{generatedMessage.Content, messageID}
	}

	_, err := s.DB.Exec(query, args...)

	// Update user memory based on chat exchange (async) - if there's actual content
	// Note: This will be moved to a separate memory service in future iterations
	if err == nil && generatedMessage.Content != nil && *generatedMessage.Content != "" {
		log.Printf("Would update user memory here (to be implemented)")
	}

	return err
}

// updateAssistantMessageWithToolCalls updates an existing assistant message with tool calls (but keeps status as processing)
func (s *ChatService) updateAssistantMessageWithToolCalls(messageID string, content *string, toolCalls []models.ChatToolCall) error {
	// Convert tool calls to JSON if present
	var toolCallsJSON *string
	if toolCalls != nil && len(toolCalls) > 0 {
		toolCallsData, err := json.Marshal(toolCalls)
		if err != nil {
			return err
		}
		toolCallsStr := string(toolCallsData)
		toolCallsJSON = &toolCallsStr
	}

	query := `
		UPDATE chat_messages
		SET content = $1, tool_calls = $2
		WHERE id = $3
	`
	_, err := s.DB.Exec(query, content, toolCallsJSON, messageID)
	return err
}

// finalizeChatMessage validates and prepares the final chat message content
func (s *ChatService) finalizeChatMessage(content string) string {
	if content == "" {
		return "I apologize, but I wasn't able to generate a proper response. Could you please try rephrasing your question?"
	}
	return content
}

// SaveToolResponse saves a tool response message to the database
func (s *ChatService) SaveToolResponse(conversationID, toolCallID string, result map[string]interface{}) error {
	// Get next sequence number for this conversation
	var sequenceNumber int
	err := s.DB.QueryRow("SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM chat_messages WHERE conversation_id = $1", conversationID).Scan(&sequenceNumber)
	if err != nil {
		return fmt.Errorf("failed to get sequence number: %w", err)
	}

	// Convert result to JSON string for tool response
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal tool result: %w", err)
	}
	resultStr := string(resultJSON)

	query := `
		INSERT INTO chat_messages (conversation_id, role, content, tool_call_id, sequence_number, status)
		VALUES ($1, 'tool', $2, $3, $4, 'completed')
	`
	_, err = s.DB.Exec(query, conversationID, resultStr, toolCallID, sequenceNumber)
	return err
}
