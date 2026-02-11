package handlers

// DEPRECATED: This file is being refactored.
//
// The chat agent functionality has been split into focused files in the chat_agent package:
//   - chat_agent/chat_service.go    - Core service struct with dependency injection
//   - chat_agent/errors.go          - Error handling utilities
//   - chat_agent/messages.go        - Message management
//   - chat_agent/compaction.go      - Conversation compaction
//   - chat_agent/tools_execution.go - Tool execution logic
//   - chat_agent/prompt.go          - Prompt building
//   - chat_agent/conversation.go    - Conversation management
//   - chat_agent/streaming.go       - Streaming response handling
//
// During the migration period, methods are being moved from here to the new package.
// The Handler.ChatService field provides access to the new ChatService.
//
// TODO: Migrate all methods to ChatService and remove deprecated functions from this file

import (
	"context"
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

// getUserFacingMessage returns a user-friendly error message using the canonical error classifier
// DEPRECATED: This function is being moved to the chat_agent package
func getUserFacingMessage(err error, customMessage string) string {
	// Use the canonical error classifier from services package
	errInfo := services.ClassifyToolError("", nil, err)
	if customMessage != "" {
		return customMessage
	}
	if errInfo != nil {
		return errInfo.Suggestion
	}
	// Fallback for context length errors (special case for LLM chat)
	if services.IsContextLengthError(err) {
		return "I apologize, but this conversation has become too long for me to process. Please consider starting a new conversation or summarizing the key points you'd like to continue discussing."
	}
	return "Something went wrong. Please try again."
}

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

// generateTitleIfNeeded generates a title for a conversation if it doesn't have one
func (s *Handler) generateTitleIfNeeded(ctx context.Context, userID int, conversation *models.ChatConversation) {
	if conversation.Title != nil && *conversation.Title != "" {
		return
	}

	messages, err := s.GetConversationMessages(conversation.ID)
	if err != nil || len(messages) == 0 {
		return
	}

	// Find first user message
	var userContent string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != nil {
			userContent = *msg.Content
			break
		}
	}

	if userContent != "" {
		generatedTitle := s.generateConversationTitle(ctx, userID, userContent)
		if err := s.UpdateConversationTitle(conversation.ID, generatedTitle); err != nil {
			log.Printf("Error updating conversation title: %v", err)
		}
	}
}

// determineModel selects the model to use based on override or conversation default
func determineModel(conversation *models.ChatConversation, modelOverride *string) string {
	if modelOverride != nil && *modelOverride != "" {
		return *modelOverride
	}
	return conversation.Model
}

// ProcessAssistantResponse handles the async processing of the assistant response
func (s *Handler) ProcessAssistantResponse(userID int, conversation *models.ChatConversation, assistantMessageID string, modelOverride *string) {
	// Update status to processing
	if err := s.UpdateMessageStatus(assistantMessageID, "processing"); err != nil {
		log.Printf("Error updating message status to processing: %v", err)
		return
	}

	// Generate title if needed (with timeout for async processing)
	titleCtx, cancelTitle := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelTitle()
	s.generateTitleIfNeeded(titleCtx, userID, conversation)

	// Get conversation history for LLM
	messages, err := s.GetConversationMessagesUpTo(conversation.ID, assistantMessageID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		return
	}

	// Determine which model to use
	modelToUse := determineModel(conversation, modelOverride)

	// Generate LLM response with tools (with timeout for async processing)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	finalAssistantMessage, err := s.GenerateChatResponse(ctx, userID, conversation, messages, modelToUse, assistantMessageID)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		return
	}

	log.Printf("finally update assistant %v", assistantMessageID)
	s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalAssistantMessage)
}

// updateAssistantMessage updates an existing assistant message with the generated content (thread-safe)
func (s *Handler) updateAssistantMessage(userID int, conversationID, messageID string, generatedMessage *models.ChatMessage) error {
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
	systemPrompt, err := prompts.GetZettelgardenAssistantPrompt()
	if err != nil {
		log.Printf("Error loading system prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		systemPrompt = "You are the Zettelgarden Assistant, a daily productivity companion for managing a Zettelkasten knowledge base."
	}

	// Add primary card context if this conversation is about a specific card
	if conversation != nil && conversation.PrimaryCardID != nil {
		card, cardErr := s.QueryFullCard(userID, *conversation.PrimaryCardID)
		if cardErr == nil {
			systemPrompt += "## Primary Focus Card\n\n"
			systemPrompt += fmt.Sprintf("This conversation is primarily about card '%s' (ID: %s).\n", card.Title, card.CardID)
			systemPrompt += "Use the get_card_by_id tool to retrieve the full content when needed.\n"
			systemPrompt += "Reference this card's content to help the user explore and develop related ideas.\n"
		}
	}

	// Note: User memory is now available via the get_user_memory tool
	// This allows just-in-time retrieval instead of pre-loading into context

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

// estimateTokenCount provides a rough estimate of token count for messages
// Uses approximation: ~4 characters per token
func estimateTokenCount(messages []openai.ChatCompletionMessage) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
		// Add tool call content
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Function.Name)
			totalChars += len(tc.Function.Arguments)
		}
	}
	// Rough estimate: 4 characters per token, plus overhead for structure
	return (totalChars / 4) + (len(messages) * 10)
}

// getModelContextLimit returns the context window size for a given model
func getModelContextLimit(model string) int {
	limits := map[string]int{
		"google/gemini-2.5-flash":       1000000,
		"google/gemini-2.5-pro":         2000000,
		"google/gemini-2.5-flash-lite":  1000000,
		"google/gemini-3-flash-preview": 1000000,
		"google/gemini-3-pro-preview":   1000000,
		"openai/gpt-5-chat":             128000,
		"openai/gpt-5.1-chat":           128000,
		"openai/gpt-5.2-chat":           128000,
		"openai/gpt-4o-mini":            128000,
		"anthropic/claude-sonnet-4":     200000,
		"anthropic/claude-sonnet-4.5": 1000000,
		"anthropic/claude-opus-4.6": 1000000,
	}

	if limit, ok := limits[model]; ok {
		return limit
	}
	// Default conservative limit
	return 100000
}

// summarizeConversationHistory creates a compact summary of older messages
func (s *Handler) summarizeConversationHistory(ctx context.Context, userID int, messages []openai.ChatCompletionMessage, model string) (openai.ChatCompletionMessage, error) {
	// Load compaction prompt
	compactionPrompt, err := prompts.GetConversationCompactionPrompt()
	if err != nil {
		log.Printf("Error loading compaction prompt: %v", err)
		compactionPrompt = "Summarize the following conversation history, preserving all critical information while reducing length. Focus on key decisions, findings, references, and unresolved issues."
	}

	// Build the conversation text to summarize
	var conversationText strings.Builder
	conversationText.WriteString("# Conversation History to Summarize\n\n")
	for _, msg := range messages {
		conversationText.WriteString(fmt.Sprintf("**%s**: %s\n\n", strings.Title(msg.Role), msg.Content))
	}

	// Use a fast, cheap model for summarization
	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
	client.Model = "google/gemini-2.5-flash-lite"
	client.RequestType = "chat"

	summaryMessages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: compactionPrompt},
		{Role: openai.ChatMessageRoleUser, Content: conversationText.String()},
	}

	resp, err := services.ExecuteLLMRequest(ctx, client, summaryMessages)
	if err != nil {
		return openai.ChatCompletionMessage{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := resp.Choices[0].Message.Content
	log.Printf("Compacted %d messages into summary of %d characters", len(messages), len(summary))

	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: summary,
	}, nil
}

// compactConversationIfNeeded checks if compaction is needed and performs it
func (s *Handler) compactConversationIfNeeded(ctx context.Context, userID int, messages []openai.ChatCompletionMessage, model string) ([]openai.ChatCompletionMessage, error) {
	tokenCount := estimateTokenCount(messages)
	contextLimit := getModelContextLimit(model)

	// Trigger compaction at 60% of context limit
	threshold := int(float64(contextLimit) * 0.6)

	if tokenCount < threshold {
		return messages, nil
	}

	log.Printf("Conversation approaching token limit (%d/%d tokens). Performing compaction...", tokenCount, contextLimit)

	// Don't compact if conversation is too short
	if len(messages) < 10 {
		log.Printf("Conversation too short to compact (%d messages)", len(messages))
		return messages, nil
	}

	// Split: first 50% gets summarized, last 50% kept
	pivotPoint := len(messages) / 2

	// Keep system prompt separate (it's always first)
	systemPrompt := messages[0]
	olderMessages := messages[1:pivotPoint]
	recentMessages := messages[pivotPoint:]

	// Summarize older half
	summary, err := s.summarizeConversationHistory(ctx, userID, olderMessages, model)
	if err != nil {
		log.Printf("Error during compaction: %v. Continuing without compaction.", err)
		return messages, nil
	}

	// Rebuild: system prompt + summary + recent messages
	compactedMessages := []openai.ChatCompletionMessage{systemPrompt, summary}
	compactedMessages = append(compactedMessages, recentMessages...)

	newTokenCount := estimateTokenCount(compactedMessages)
	log.Printf("Compaction complete: %d -> %d tokens (%d%% reduction)", tokenCount, newTokenCount, (tokenCount-newTokenCount)*100/tokenCount)

	return compactedMessages, nil
}


// executeToolCall executes a single tool call with simple retry logic
func (s *Handler) executeToolCall(toolRegistry *services.ToolRegistry, tc openai.ToolCall, ctx *services.ToolContext) map[string]interface{} {
	var args map[string]interface{}
	json.Unmarshal([]byte(tc.Function.Arguments), &args)

	// Use simplified retry logic (max 1 retry)
	result, err := services.ExecuteToolWithRetry(
		context.Background(),
		tc.Function.Name,
		args,
		toolRegistry,
		ctx,
	)

	// Handle execution error
	if err != nil {
		log.Printf("Tool %s: failed - %v", tc.Function.Name, err)
		return services.WrapToolError(tc.Function.Name, args, err)
	}

	return result
}

// executeAndSaveToolCalls executes all tool calls and saves their responses
func (s *Handler) executeAndSaveToolCalls(toolRegistry *services.ToolRegistry, toolCalls []openai.ToolCall, userID int, conversation *models.ChatConversation, assistantMessageID string, model string, loopDetector *services.LoopDetector) error {
	ctx := &services.ToolContext{
		UserID:          userID,
		DB:              s.DB,
		TypesenseClient: s.Server.TypesenseClient,
		ConversationID:  &conversation.ID,
		MessageID:       &assistantMessageID,
		Model:           model,
		Context:         context.Background(),
	}

	for _, tc := range toolCalls {
		result := s.executeToolCall(toolRegistry, tc, ctx)

		// Increment loop detector counter for each tool call
		if loopDetector != nil {
			loopDetector.Increment()
		}

		// Convert result to JSON string for tool response
		resultJSON, _ := json.Marshal(result)
		resultStr := string(resultJSON)

		// Save tool response message
		_, err := s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID, nil, "completed")
		if err != nil {
			log.Printf("Error saving tool response: %v", err)
		}
	}

	return nil
}

// handleLLMError handles common LLM errors and returns appropriate message
func (s *Handler) handleLLMError(err error, userID int, conversation *models.ChatConversation, assistantMessageID string) (*models.ChatMessage, error) {
	log.Printf("executing LLM error for conversation %s: %v", conversation.ID, err)

	userMessage := getUserFacingMessage(err, "")

	finalMessage := &models.ChatMessage{
		ID:      assistantMessageID,
		Content: &userMessage,
	}
	if updateErr := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage); updateErr != nil {
		log.Printf("Error updating message with error: %v", updateErr)
		return nil, updateErr
	}
	return finalMessage, nil
}

// finalizeChatMessage validates and saves the final chat message
func (s *Handler) finalizeChatMessage(content string, userID int, conversation *models.ChatConversation, assistantMessageID string) (*models.ChatMessage, error) {
	if strings.TrimSpace(content) == "" {
		log.Printf("Warning: LLM returned empty content for conversation %s", conversation.ID)
		content = getUserFacingMessage(fmt.Errorf("empty response"), "I apologize, but I wasn't able to generate a proper response. Could you please try rephrasing your question?")
	}

	finalMessage := &models.ChatMessage{
		ID:      assistantMessageID,
		Content: &content,
	}

	if err := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage); err != nil {
		return nil, err
	}

	return finalMessage, nil
}

// GenerateChatResponse generates an LLM response with tool calling support
func (s *Handler) GenerateChatResponse(ctx context.Context, userID int, conversation *models.ChatConversation, messages []models.ChatMessage, model string, assistantMessageID string) (*models.ChatMessage, error) {
	systemPrompt, err := s.buildSystemPrompt(userID, conversation)
	if err != nil {
		return nil, err
	}

	converter := services.NewMessageConverter()
	openaiMessages := converter.ToOpenAI(messages, systemPrompt)

	// Check if compaction is needed and perform it proactively
	openaiMessages, err = s.compactConversationIfNeeded(ctx, userID, openaiMessages, model)
	if err != nil {
		log.Printf("Error during compaction: %v", err)
		// Continue anyway - compaction is an optimization, not required
	}

	// Create LLM client
	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
	client.Model = model
	client.RequestType = "chat"

	// Get tools registry
	toolRegistry := services.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	// Initialize loop detector for this conversation
	loopDetector := services.NewLoopDetector()

	// Track iteration count for progress logging
	iterationCount := 0
	const progressFeedbackInterval = 5 // Log progress every 5 iterations

	// Loop until no more tool calls are needed
	for {
		// Check if loop detector has reached max iterations
		// We check against maxIterations-1 because Increment() is called in executeAndSaveToolCalls
		if loopDetector.GetIteration() >= 9 { // maxIterations is 10 in NewLoopDetector
			log.Printf("[LoopDetector] Max iterations reached for conversation %s", conversation.ID)
			interventionMsg := loopDetector.GetInterventionMessage()
			// Add intervention as a system message to break the loop
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: interventionMsg,
			})
			// Clear the loop detector history after intervention
			loopDetector.Reset()
		}

		resp, err := services.ExecuteLLMToolRequest(ctx, client, openaiMessages, tools)
		if err != nil {
			return s.handleLLMError(err, userID, conversation, assistantMessageID)
		}

		// Validate response
		if len(resp.Choices) == 0 {
			log.Printf("Warning: LLM returned no choices for conversation %s", conversation.ID)
			return s.finalizeChatMessage(getUserFacingMessage(fmt.Errorf("no choices"), ""), userID, conversation, assistantMessageID)
		}

		assistantMessage := resp.Choices[0].Message

		// Handle empty response - provide immediate specific error instead of silent retry
		if strings.TrimSpace(assistantMessage.Content) == "" && len(assistantMessage.ToolCalls) == 0 {
			log.Printf("[EmptyResponse] LLM returned empty content for conversation %s", conversation.ID)
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			return s.finalizeChatMessage(getUserFacingMessage(fmt.Errorf("empty response"), errorMsg), userID, conversation, assistantMessageID)
		}

		// Increment iteration counter and log progress
		iterationCount++
		if iterationCount%progressFeedbackInterval == 0 && iterationCount > 0 {
			log.Printf("[Progress] Chat response iteration %d for conversation %s", iterationCount, conversation.ID)
		}

		// If no tool calls, update the existing assistant message and return
		if len(assistantMessage.ToolCalls) == 0 {
			log.Printf("no more tool calls")
			return s.finalizeChatMessage(assistantMessage.Content, userID, conversation, assistantMessageID)
		}

		// Convert and save tool calls
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
		if err = s.updateAssistantMessageWithToolCalls(assistantMessageID, &assistantMessage.Content, toolCalls); err != nil {
			return nil, err
		}

		// Execute tool calls and save tool responses
		if err = s.executeAndSaveToolCalls(toolRegistry, assistantMessage.ToolCalls, userID, conversation, assistantMessageID, model, loopDetector); err != nil {
			return nil, err
		}

		// Get updated messages including tool responses for next iteration
		updatedMessages, err := s.GetConversationMessages(conversation.ID)
		if err != nil {
			return nil, err
		}

		// Rebuild system prompt for next iteration
		currentSystemPrompt, promptErr := s.buildSystemPrompt(userID, conversation)
		if promptErr != nil {
			log.Printf("Error reloading system prompt: %v, using previous", promptErr)
			currentSystemPrompt = systemPrompt
		}

		openaiMessages = converter.ToOpenAI(updatedMessages, currentSystemPrompt)
	}
}

// generateConversationTitle generates a title for a conversation based on the user's first message
func (s *Handler) generateConversationTitle(ctx context.Context, userID int, userMessage string) string {
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
	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
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

	resp, err := services.ExecuteLLMRequest(ctx, client, messages)

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

// generateTitleIfNeededWithEvent generates a title and sends an SSE event
func (s *Handler) generateTitleIfNeededWithEvent(ctx context.Context, userID int, conversation *models.ChatConversation, sendEvent func(string, interface{}) error) {
	if conversation.Title != nil && *conversation.Title != "" {
		return
	}

	messages, err := s.GetConversationMessages(conversation.ID)
	if err != nil || len(messages) == 0 {
		return
	}

	// Find first user message
	var userContent string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != nil {
			userContent = *msg.Content
			break
		}
	}

	if userContent != "" {
		generatedTitle := s.generateConversationTitle(ctx, userID, userContent)
		if err := s.UpdateConversationTitle(conversation.ID, generatedTitle); err != nil {
			log.Printf("Error updating conversation title: %v", err)
		} else {
			sendEvent("title", map[string]string{"title": generatedTitle})
		}
	}
}

// streamAssistantResponse handles streaming the assistant response
func (s *Handler) streamAssistantResponse(ctx context.Context, w http.ResponseWriter, userID int, conversation *models.ChatConversation, assistantMessageID string, modelOverride *string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming unsupported")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		return
	}

	// Ensure message status is cleaned up on any exit path
	// If we complete successfully, updateStatus will be set to true
	updateStatus := true // Default to success (will be marked "completed" by updateAssistantMessage)
	defer func() {
		// Check if context was cancelled (client disconnect)
		if ctx.Err() != nil {
			log.Printf("Client disconnected during streaming, marking message as failed: %s", assistantMessageID)
			s.UpdateMessageStatus(assistantMessageID, "failed")
			return
		}
		// If there was an error during execution, status would have been set to "failed" already
		// Only mark as completed if we reached the end successfully
		if updateStatus {
			// Status is already set by updateAssistantMessage to "completed"
		}
	}()

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

	// Check for client disconnect early
	select {
	case <-ctx.Done():
		log.Printf("Client disconnected before streaming started: %s", assistantMessageID)
		return
	default:
	}

	// Generate title if needed and send event
	s.generateTitleIfNeededWithEvent(ctx, userID, conversation, sendEvent)

	// Get conversation history
	messages, err := s.GetConversationMessagesUpTo(conversation.ID, assistantMessageID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": "Failed to get conversation history"})
		return
	}

	// Determine which model to use
	modelToUse := determineModel(conversation, modelOverride)

	// Generate response with streaming
	err = s.streamChatResponse(ctx, w, userID, conversation, messages, modelToUse, assistantMessageID, sendEvent)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		s.UpdateMessageStatus(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": err.Error()})
		return
	}

	sendEvent("done", map[string]string{"message_id": assistantMessageID})
}

// processStreamResponse processes a stream response and accumulates content and tool calls
func processStreamResponse(ctx context.Context, stream *openai.ChatCompletionStream, sendEvent func(string, interface{}) error) (string, []openai.ToolCall, error) {
	var currentContent string
	var currentToolCalls []openai.ToolCall

	for {
		// Check for client disconnect before receiving next chunk
		select {
		case <-ctx.Done():
			log.Printf("Client disconnected during stream processing")
			return "", nil, ctx.Err()
		default:
		}

		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Stream error: %v", err)
			return "", nil, err
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		// Handle content delta
		if delta.Content != "" {
			currentContent += delta.Content
			sendEvent("content", map[string]string{"delta": delta.Content})
		}

		// Handle tool calls - accumulate deltas
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
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

	return currentContent, currentToolCalls, nil
}

// convertAndBroadcastToolCalls converts tool calls to our format and sends events
func convertAndBroadcastToolCalls(toolCalls []openai.ToolCall, sendEvent func(string, interface{}) error) []models.ChatToolCall {
	var convertedToolCalls []models.ChatToolCall

	for _, tc := range toolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		convertedToolCalls = append(convertedToolCalls, models.ChatToolCall{
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

	return convertedToolCalls
}

// executeAndBroadcastToolCalls executes tool calls and sends result events
func (s *Handler) executeAndBroadcastToolCalls(toolRegistry *services.ToolRegistry, toolCalls []openai.ToolCall, userID int, conversation *models.ChatConversation, assistantMessageID string, model string, sendEvent func(string, interface{}) error, loopDetector *services.LoopDetector) error {
	ctx := &services.ToolContext{
		UserID:          userID,
		DB:              s.DB,
		TypesenseClient: s.Server.TypesenseClient,
		ConversationID:  &conversation.ID,
		MessageID:       &assistantMessageID,
		Model:           model,
		Context:         context.Background(),
	}

	for _, tc := range toolCalls {
		result := s.executeToolCall(toolRegistry, tc, ctx)

		// Increment loop detector counter for each tool call
		if loopDetector != nil {
			loopDetector.Increment()
		}

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
			"timestamp":    time.Now().Format(time.RFC3339),
		}
		sendEvent("tool_result", eventData)

		// Save tool response
		resultJSON, _ := json.Marshal(result)
		resultStr := string(resultJSON)
		_, err := s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID, nil, "completed")
		if err != nil {
			log.Printf("Error saving tool response: %v", err)
		}
	}

	return nil
}

// streamChatResponse generates a chat response with streaming and tool support
func (s *Handler) streamChatResponse(ctx context.Context, w http.ResponseWriter, userID int, conversation *models.ChatConversation, messages []models.ChatMessage, model string, assistantMessageID string, sendEvent func(string, interface{}) error) error {
	systemPrompt, err := s.buildSystemPrompt(userID, conversation)
	if err != nil {
		return err
	}

	converter := services.NewMessageConverter()
	openaiMessages := converter.ToOpenAI(messages, systemPrompt)

	// Check if compaction is needed and perform it proactively
	openaiMessages, err = s.compactConversationIfNeeded(ctx, userID, openaiMessages, model)
	if err != nil {
		log.Printf("Error during compaction: %v", err)
		// Continue anyway - compaction is an optimization, not required
	}

	// Create LLM client
	isTesting := s.Server != nil && s.Server.Testing
	client := services.NewDefaultClient(s.DB, userID, isTesting)
	client.Model = model
	client.RequestType = "chat"

	// Get tools registry
	toolRegistry := services.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	// Initialize loop detector for this conversation
	loopDetector := services.NewLoopDetector()

	// Track iteration count for progress feedback
	iterationCount := 0
	const progressFeedbackInterval = 5 // Send progress every 5 iterations

	// Loop until no more tool calls are needed
	for {
		// Check if loop detector has reached max iterations
		// We check against maxIterations-1 because Increment() is called in executeAndSaveToolCalls
		if loopDetector.GetIteration() >= 9 { // maxIterations is 10 in NewLoopDetector
			log.Printf("[LoopDetector] Max iterations reached for conversation %s", conversation.ID)
			interventionMsg := loopDetector.GetInterventionMessage()
			// Add intervention as a system message to break the loop
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: interventionMsg,
			})
			// Clear the loop detector history after intervention
			loopDetector.Reset()
		}

		stream, err := services.StreamLLMToolRequest(ctx, client, openaiMessages, tools)
		if err != nil {
			userMsg := getUserFacingMessage(err, "")
			// Check for context length error specifically
			if services.IsContextLengthError(err) {
				s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, &models.ChatMessage{
					ID:      assistantMessageID,
					Content: &userMsg,
				})
				return sendEvent("error", map[string]string{"error": userMsg})
			}
			// For other errors, return the user-friendly message
			return sendEvent("error", map[string]string{"error": userMsg})
		}
		defer stream.Close()

		// Process the stream and collect results
		currentContent, currentToolCalls, err := processStreamResponse(ctx, stream, sendEvent)
		if err != nil {
			return err
		}

		// Handle empty response - provide immediate specific error instead of silent retry
		if strings.TrimSpace(currentContent) == "" && len(currentToolCalls) == 0 {
			log.Printf("[EmptyResponse] Streaming LLM returned empty content for conversation %s", conversation.ID)
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			userMsg := getUserFacingMessage(fmt.Errorf("empty response"), errorMsg)
			s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &userMsg,
			})
			return sendEvent("error", map[string]string{"error": userMsg})
		}

		// Increment iteration counter and send progress feedback
		iterationCount++
		if iterationCount%progressFeedbackInterval == 0 && iterationCount > 0 {
			progressMsg := fmt.Sprintf("Working on it... (step %d)", iterationCount)
			sendEvent("progress", map[string]interface{}{
				"step":    iterationCount,
				"message": progressMsg,
			})
			log.Printf("[Progress] Chat response iteration %d for conversation %s", iterationCount, conversation.ID)
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
			return s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
		}

		// Convert tool calls and broadcast events
		toolCalls := convertAndBroadcastToolCalls(currentToolCalls, sendEvent)

		// Update message with tool calls
		if err = s.updateAssistantMessageWithToolCalls(assistantMessageID, &currentContent, toolCalls); err != nil {
			return err
		}

		// Execute tool calls and broadcast results
		if err = s.executeAndBroadcastToolCalls(toolRegistry, currentToolCalls, userID, conversation, assistantMessageID, model, sendEvent, loopDetector); err != nil {
			return err
		}

		// Get updated messages for next iteration
		updatedMessages, err := s.GetConversationMessages(conversation.ID)
		if err != nil {
			return err
		}

		// Rebuild system prompt and messages for next iteration
		currentSystemPrompt, promptErr := s.buildSystemPrompt(userID, conversation)
		if promptErr != nil {
			currentSystemPrompt = systemPrompt
		}

		openaiMessages = converter.ToOpenAI(updatedMessages, currentSystemPrompt)
	}
}
