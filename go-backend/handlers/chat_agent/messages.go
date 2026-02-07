package chat_agent

import (
	"encoding/json"
	"go-backend/models"
	"log"
)

// updateAssistantMessage updates an existing assistant message with the generated content (thread-safe)
func (s *ChatService) updateAssistantMessage(userID int, conversationID, messageID string, generatedMessage *models.ChatMessage) error {
	// Acquire per-message mutex to prevent race conditions on message updates
	mu := s.getMessageMutex(messageID)
	mu.Lock()
	defer mu.Unlock()

	// Convert tool calls to JSON if present
	var toolCallsJSON *string
	if generatedMessage.ToolCalls != nil && len(generatedMessage.ToolCalls) > 0 {
		toolCallsData, err := json.Marshal(generatedMessage.ToolCalls)
		if err != nil {
			return err
		}
		toolCallsStr := string(toolCallsData)
		toolCallsJSON = &toolCallsStr
	}

	query := `
		UPDATE chat_messages
		SET content = $1, tool_calls = $2, status = 'completed'
		WHERE id = $3
	`
	_, err := s.DB.Exec(query, generatedMessage.Content, toolCallsJSON, messageID)

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
	// Convert result to JSON string for tool response
	resultJSON, _ := json.Marshal(result)
	resultStr := string(resultJSON)

	query := `
		INSERT INTO chat_messages (conversation_id, role, content, tool_call_id, status)
		VALUES ($1, 'tool', $2, $3, 'completed')
	`
	_, err := s.DB.Exec(query, conversationID, resultStr, toolCallID)
	return err
}
