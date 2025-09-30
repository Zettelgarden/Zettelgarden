package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// isToolResultEmpty checks if a tool result is effectively empty
func isToolResultEmpty(result map[string]interface{}) bool {
	if result == nil || len(result) == 0 {
		return true
	}

	// Check if result only contains empty values
	for key, value := range result {
		// Skip error field for this check
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

// processAssistantResponse handles the async processing of the assistant response
func (s *Handler) processAssistantResponse(userID int, conversation *models.ChatConversation, assistantMessageID string, modelOverride *string) {
	// Update status to processing
	if err := s.UpdateMessageStatus(assistantMessageID, "processing"); err != nil {
		log.Printf("Error updating message status to processing: %v", err)
		return
	}

	// Generate title if this is the first message (and conversation has no title)
	if conversation.Title == nil || *conversation.Title == "" {
		// Get the latest user message to generate title
		messages, err := s.GetConversationMessages(conversation.ID)
		if err == nil && len(messages) > 0 {
			var userContent string
			for _, msg := range messages {
				if msg.Role == "user" && msg.Content != nil {
					userContent = *msg.Content
					break
				}
			}
			if userContent != "" {
				generatedTitle := s.generateConversationTitle(userID, userContent)
				err := s.UpdateConversationTitle(conversation.ID, generatedTitle)
				if err != nil {
					log.Printf("Error updating conversation title: %v", err)
				}
			}
		}
	}

	// Get conversation history for LLM (up to but not including the message being regenerated)
	messages, err := s.GetConversationMessagesUpTo(conversation.ID, assistantMessageID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		return
	}

	// Determine which model to use - request override or conversation default
	modelToUse := conversation.Model
	if modelOverride != nil && *modelOverride != "" {
		modelToUse = *modelOverride
	}

	// Generate LLM response with tools
	finalAssistantMessage, err := s.GenerateChatResponse(userID, conversation, messages, modelToUse, assistantMessageID)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		return
	}

	log.Printf("finally update assistant %v", assistantMessageID)
	// Update the assistant message with the actual content and mark as completed
	s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalAssistantMessage)
}

// updateAssistantMessage updates an existing assistant message with the generated content
func (s *Handler) updateAssistantMessage(userID int, conversationID, messageID string, generatedMessage *models.ChatMessage) error {
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
	if err == nil && generatedMessage.Content != nil && *generatedMessage.Content != "" {
		// Get the most recent user message to analyze the chat exchange
		go func() {
			userMessage := s.getLatestUserMessage(conversationID)
			if userMessage != "" {
				s.GenerateChatMemory(uint(userID), userMessage, *generatedMessage.Content)
			}
		}()
	}

	return err
}

// updateAssistantMessageWithToolCalls updates an existing assistant message with tool calls (but keeps status as processing)
func (s *Handler) updateAssistantMessageWithToolCalls(messageID string, content *string, toolCalls []models.ChatToolCall) error {
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

// getLatestUserMessage gets the most recent user message from a conversation
func (s *Handler) getLatestUserMessage(conversationID string) string {
	var content *string
	query := `
		SELECT content
		FROM chat_messages
		WHERE conversation_id = $1 AND role = 'user' AND content IS NOT NULL
		ORDER BY sequence_number DESC
		LIMIT 1
	`
	err := s.DB.QueryRow(query, conversationID).Scan(&content)
	if err != nil || content == nil {
		return ""
	}
	return *content
}

// buildSystemPrompt constructs the complete system prompt including memory, instructions, and context
func (s *Handler) buildSystemPrompt(userID int, conversation *models.ChatConversation) (string, error) {
	systemPrompt, err := prompts.GetResearchAssistantPrompt()
	if err != nil {
		log.Printf("Error loading system prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		systemPrompt = "You are the Research Assistant for a Zettelkasten knowledge base. Help users explore and synthesize information across their cards."
	}

	// Add primary card context if this conversation is about a specific card
	if conversation != nil && conversation.PrimaryCardID != nil {
		card, cardErr := s.QueryFullCard(userID, *conversation.PrimaryCardID)
		if cardErr == nil {
			systemPrompt += "\n\n## Primary Focus Card\n\n"
			systemPrompt += fmt.Sprintf("This conversation is primarily about the following card:\n\n")
			systemPrompt += fmt.Sprintf("**Card ID**: %s\n", card.CardID)
			systemPrompt += fmt.Sprintf("**Title**: %s\n", card.Title)
			systemPrompt += fmt.Sprintf("**Content**:\n%s\n\n", card.Body)
			systemPrompt += "When responding, keep this card as the main focus of the conversation unless the user explicitly asks about something else. Reference this card's content and help the user explore and develop ideas related to it."
		}
	}

	// Add user memory if available
	memory, memErr := GetUserMemory(s.DB, userID)
	if memErr == nil && memory != "" {
		systemPrompt += "\n\n## Your Memory Of The User\n\n"
		systemPrompt += memory
	}

	// Add user's chat instructions if they exist
	instructions, instrErr := s.GetChatInstructions(userID)
	if instrErr == nil && instructions.Instructions != "" {
		systemPrompt += "\n\n## User Instructions\n\n"
		systemPrompt += instructions.Instructions
	}

	// Add current date and time
	currentTime := time.Now()
	systemPrompt += "\n\n## Current Date and Time\n\n"
	systemPrompt += fmt.Sprintf("Today's date is %s (UTC: %s)",
		currentTime.Format("Monday, January 2, 2006"),
		currentTime.UTC().Format("2006-01-02 15:04:05 UTC"))

	return systemPrompt, nil
}

// GenerateChatResponse generates an LLM response with tool calling support
func (s *Handler) GenerateChatResponse(userID int, conversation *models.ChatConversation, messages []models.ChatMessage, model string, assistantMessageID string) (*models.ChatMessage, error) {
	var openaiMessages []openai.ChatCompletionMessage

	systemPrompt, err := s.buildSystemPrompt(userID, conversation)
	if err != nil {
		return nil, err
	}

	// Collect all referenced card IDs from user messages
	var allReferencedCardIDs []string
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.ReferencedCards) > 0 {
			allReferencedCardIDs = append(allReferencedCardIDs, msg.ReferencedCards...)
		}
	}

	openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	// Add conversation messages
	for _, msg := range messages {
		var content string
		if msg.Content != nil {
			content = *msg.Content
		}

		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			var toolCalls []openai.ToolCall
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// Handle tool call responses
		if msg.ToolCallID != nil {
			openaiMsg.ToolCallID = *msg.ToolCallID
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Create LLM client
	client := services.NewDefaultClient(s.DB, userID)
	client.Model = model
	client.RequestType = "chat"

	// Get tools registry
	toolRegistry := services.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	// Loop until no more tool calls are needed
	for {
		resp, err := services.ExecuteLLMToolRequest(client, openaiMessages, tools)

		if err != nil {
			log.Printf("executing LLM error for conversation %s: %v", conversation.ID, err)

			// Check if this is a context length error
			if services.IsContextLengthError(err) {
				log.Printf("Context length exceeded for conversation %s", conversation.ID)
				// Update the message with a helpful error message instead of failing completely
				errorMessage := "I apologize, but this conversation has become too long for me to process. The context exceeds the model's token limit. Please consider starting a new conversation or summarizing the key points you'd like to continue discussing."
				finalMessage := &models.ChatMessage{
					ID:      assistantMessageID,
					Content: &errorMessage,
				}
				err := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
				if err != nil {
					log.Printf("Error updating message with context length error: %v", err)
					return nil, err
				}
				return finalMessage, nil
			}

			return nil, err
		}

		// Validate response
		if len(resp.Choices) == 0 {
			log.Printf("Warning: LLM returned no choices for conversation %s", conversation.ID)
			errorMessage := "I apologize, but I encountered an issue generating a response. Please try again."
			finalMessage := &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &errorMessage,
			}
			err := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
			if err != nil {
				return nil, err
			}
			return finalMessage, nil
		}

		assistantMessage := resp.Choices[0].Message

		// If no tool calls, update the existing assistant message and return
		if len(assistantMessage.ToolCalls) == 0 {
			log.Printf("assisantmessage %v", resp)

			// Check if content is empty and provide fallback
			content := assistantMessage.Content
			if strings.TrimSpace(content) == "" {
				log.Printf("Warning: LLM returned empty content for conversation %s", conversation.ID)
				content = "I apologize, but I wasn't able to generate a proper response. Could you please try rephrasing your question?"
			}

			finalMessage := &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &content,
			}
			log.Printf("no more tool calls")
			err := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
			if err != nil {
				return nil, err
			}
			return finalMessage, nil
		}

		// Save assistant message with tool calls
		var toolCalls []models.ChatToolCall
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			toolCalls = append(toolCalls, models.ChatToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: models.ChatToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			})
		}

		// Update the existing pending assistant message with tool calls
		err = s.updateAssistantMessageWithToolCalls(assistantMessageID, &assistantMessage.Content, toolCalls)
		if err != nil {
			return nil, err
		}

		// Create a message object to reference for tool context
		assistantMsg := &models.ChatMessage{
			ID: assistantMessageID,
		}

		// Execute tool calls and save tool responses
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			ctx := &services.ToolContext{
				UserID:          userID,
				DB:              s.DB,
				TypesenseClient: s.Server.TypesenseClient,
				ConversationID:  &conversation.ID,
				MessageID:       &assistantMsg.ID,
				Model:           model,
			}
			result, err := toolRegistry.ExecuteTool(tc.Function.Name, args, ctx)
			if err != nil {
				log.Printf("Error executing tool %s: %v", tc.Function.Name, err)
				result = map[string]interface{}{
					"error": err.Error(),
				}
			}

			// Check if result is empty and retry once
			if isToolResultEmpty(result) && err == nil {
				log.Printf("Tool %s returned empty result, retrying once...", tc.Function.Name)
				result, err = toolRegistry.ExecuteTool(tc.Function.Name, args, ctx)
				if err != nil {
					log.Printf("Error on retry executing tool %s: %v", tc.Function.Name, err)
					result = map[string]interface{}{
						"error": err.Error(),
					}
				}
				if isToolResultEmpty(result) {
					log.Printf("Tool %s returned empty result after retry", tc.Function.Name)
				}
			}

			// Convert result to JSON string for tool response
			resultJSON, _ := json.Marshal(result)
			resultStr := string(resultJSON)

			// Save tool response message
			_, err = s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID, nil, "completed")
			if err != nil {
				log.Printf("Error saving tool response: %v", err)
			}
		}

		// Get updated messages including tool responses for next iteration
		updatedMessages, err := s.GetConversationMessages(conversation.ID)
		if err != nil {
			return nil, err
		}

		// Convert to OpenAI format for next request
		// Load system prompt again for consistency
		currentSystemPrompt, promptErr := s.buildSystemPrompt(userID, conversation)
		if promptErr != nil {
			log.Printf("Error reloading system prompt: %v, using previous", promptErr)
			currentSystemPrompt = systemPrompt // Use the previously loaded prompt
		}

		openaiMessages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: currentSystemPrompt,
			},
		}

		for _, msg := range updatedMessages {
			var content string
			if msg.Content != nil {
				content = *msg.Content
			}

			openaiMsg := openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: content,
			}

			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ToolCall
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Function.Arguments)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   tc.ID,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: string(argsJSON),
						},
					})
				}
				openaiMsg.ToolCalls = toolCalls
			}

			if msg.ToolCallID != nil {
				openaiMsg.ToolCallID = *msg.ToolCallID
			}

			openaiMessages = append(openaiMessages, openaiMsg)
		}
	}
}

// generateConversationTitle generates a title for a conversation based on the user's first message
func (s *Handler) generateConversationTitle(userID int, userMessage string) string {
	// Load title generation prompt
	titlePrompt, err := prompts.GetTitleGeneratorPrompt()
	if err != nil {
		log.Printf("Error loading title generation prompt: %v", err)
		// Fallback to truncated user message
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	// Create LLM client for title generation
	client := services.NewDefaultClient(s.DB, userID)
	client.RequestType = "chat"
	client.Model = "google/gemini-2.5-flash-lite" // Use faster, cheaper model for title generation

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: titlePrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		},
	}

	resp, err := services.ExecuteLLMRequest(client, messages)

	if err != nil {
		log.Printf("Error generating conversation title: %v", err)
		// Fallback to truncated user message
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Ensure title isn't too long
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	// If title generation failed or returned empty, use fallback
	if title == "" {
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	return title
}

// streamAssistantResponse handles streaming the assistant response
func (s *Handler) streamAssistantResponse(w http.ResponseWriter, userID int, conversation *models.ChatConversation, assistantMessageID string, modelOverride *string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming unsupported")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Helper to send SSE events
	sendEvent := func(eventType string, data interface{}) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
		flusher.Flush()
		return nil
	}

	// Generate title if this is the first message
	if conversation.Title == nil || *conversation.Title == "" {
		messages, err := s.GetConversationMessages(conversation.ID)
		if err == nil && len(messages) > 0 {
			var userContent string
			for _, msg := range messages {
				if msg.Role == "user" && msg.Content != nil {
					userContent = *msg.Content
					break
				}
			}
			if userContent != "" {
				generatedTitle := s.generateConversationTitle(userID, userContent)
				err := s.UpdateConversationTitle(conversation.ID, generatedTitle)
				if err != nil {
					log.Printf("Error updating conversation title: %v", err)
				} else {
					sendEvent("title", map[string]string{"title": generatedTitle})
				}
			}
		}
	}

	// Get conversation history
	messages, err := s.GetConversationMessagesUpTo(conversation.ID, assistantMessageID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": "Failed to get conversation history"})
		return
	}

	// Determine which model to use
	modelToUse := conversation.Model
	if modelOverride != nil && *modelOverride != "" {
		modelToUse = *modelOverride
	}

	// Generate response with streaming
	err = s.streamChatResponse(w, userID, conversation, messages, modelToUse, assistantMessageID, sendEvent)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": err.Error()})
		return
	}

	sendEvent("done", map[string]string{"message_id": assistantMessageID})
}

// streamChatResponse generates a chat response with streaming and tool support
func (s *Handler) streamChatResponse(w http.ResponseWriter, userID int, conversation *models.ChatConversation, messages []models.ChatMessage, model string, assistantMessageID string, sendEvent func(string, interface{}) error) error {
	var openaiMessages []openai.ChatCompletionMessage

	systemPrompt, err := s.buildSystemPrompt(userID, conversation)
	if err != nil {
		return err
	}

	openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	// Add conversation messages
	for _, msg := range messages {
		var content string
		if msg.Content != nil {
			content = *msg.Content
		}

		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			var toolCalls []openai.ToolCall
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// Handle tool call responses
		if msg.ToolCallID != nil {
			openaiMsg.ToolCallID = *msg.ToolCallID
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Create LLM client
	client := services.NewDefaultClient(s.DB, userID)
	client.Model = model
	client.RequestType = "chat"

	// Get tools registry
	toolRegistry := services.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	var fullContent strings.Builder

	// Loop until no more tool calls are needed
	for {
		stream, err := services.StreamLLMToolRequest(client, openaiMessages, tools)
		if err != nil {
			if services.IsContextLengthError(err) {
				errorMessage := "I apologize, but this conversation has become too long for me to process. The context exceeds the model's token limit."
				s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, &models.ChatMessage{
					ID:      assistantMessageID,
					Content: &errorMessage,
				})
				return sendEvent("error", map[string]string{"error": errorMessage})
			}
			return err
		}
		defer stream.Close()

		var currentContent string
		var currentToolCalls []openai.ToolCall

		// Process stream
		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Stream error: %v", err)
				return err
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta

			// Handle content delta
			if delta.Content != "" {
				currentContent += delta.Content
				fullContent.WriteString(delta.Content)
				sendEvent("content", map[string]string{"delta": delta.Content})
			}

			// Handle tool calls
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					// Accumulate tool call data
					if tc.Index != nil {
						idx := *tc.Index
						// Ensure we have enough space
						for len(currentToolCalls) <= idx {
							currentToolCalls = append(currentToolCalls, openai.ToolCall{})
						}
						if tc.ID != "" {
							currentToolCalls[idx].ID = tc.ID
							currentToolCalls[idx].Type = tc.Type
						}
						if tc.Function.Name != "" {
							currentToolCalls[idx].Function.Name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							currentToolCalls[idx].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}
		}

		// If no tool calls, we're done
		if len(currentToolCalls) == 0 {
			if strings.TrimSpace(currentContent) == "" {
				currentContent = "I apologize, but I wasn't able to generate a proper response."
			}

			finalMessage := &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &currentContent,
			}
			err := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
			return err
		}

		// Convert and save tool calls
		var toolCalls []models.ChatToolCall
		for _, tc := range currentToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			toolCalls = append(toolCalls, models.ChatToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: models.ChatToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			})

			// Send tool call event
			sendEvent("tool_call", map[string]interface{}{
				"id":        tc.ID,
				"name":      tc.Function.Name,
				"arguments": args,
			})
		}

		// Update message with tool calls
		err = s.updateAssistantMessageWithToolCalls(assistantMessageID, &currentContent, toolCalls)
		if err != nil {
			return err
		}

		// Execute tool calls
		assistantMsg := &models.ChatMessage{ID: assistantMessageID}
		for _, tc := range currentToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			ctx := &services.ToolContext{
				UserID:          userID,
				DB:              s.DB,
				TypesenseClient: s.Server.TypesenseClient,
				ConversationID:  &conversation.ID,
				MessageID:       &assistantMsg.ID,
				Model:           model,
			}

			result, err := toolRegistry.ExecuteTool(tc.Function.Name, args, ctx)
			if err != nil {
				log.Printf("Error executing tool %s: %v", tc.Function.Name, err)
				result = map[string]interface{}{"error": err.Error()}
			}

			// Check if result is empty and retry once
			if isToolResultEmpty(result) && err == nil {
				log.Printf("Tool %s returned empty result, retrying once...", tc.Function.Name)
				result, err = toolRegistry.ExecuteTool(tc.Function.Name, args, ctx)
				if err != nil {
					log.Printf("Error on retry executing tool %s: %v", tc.Function.Name, err)
					result = map[string]interface{}{"error": err.Error()}
				}
				if isToolResultEmpty(result) {
					log.Printf("Tool %s returned empty result after retry", tc.Function.Name)
				}
			}

			// Send tool result event
			sendEvent("tool_result", map[string]interface{}{
				"tool_call_id": tc.ID,
				"name":         tc.Function.Name,
				"result":       result,
			})

			// Save tool response
			resultJSON, _ := json.Marshal(result)
			resultStr := string(resultJSON)
			_, err = s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID, nil, "completed")
			if err != nil {
				log.Printf("Error saving tool response: %v", err)
			}
		}

		// Get updated messages for next iteration
		updatedMessages, err := s.GetConversationMessages(conversation.ID)
		if err != nil {
			return err
		}

		// Rebuild openaiMessages
		currentSystemPrompt, promptErr := s.buildSystemPrompt(userID, conversation)
		if promptErr != nil {
			currentSystemPrompt = systemPrompt
		}

		openaiMessages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: currentSystemPrompt,
			},
		}

		for _, msg := range updatedMessages {
			var content string
			if msg.Content != nil {
				content = *msg.Content
			}

			openaiMsg := openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: content,
			}

			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ToolCall
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Function.Arguments)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   tc.ID,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: string(argsJSON),
						},
					})
				}
				openaiMsg.ToolCalls = toolCalls
			}

			if msg.ToolCallID != nil {
				openaiMsg.ToolCallID = *msg.ToolCallID
			}

			openaiMessages = append(openaiMessages, openaiMsg)
		}
	}
}