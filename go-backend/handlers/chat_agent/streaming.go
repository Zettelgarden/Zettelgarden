package chat_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// StreamEventHandler is a callback function for sending SSE events
type StreamEventHandler func(eventType string, data interface{}) error

// GetConversationMessagesFunc is the function signature for fetching conversation messages
type GetConversationMessagesFunc func(conversationID string) ([]models.ChatMessage, error)

const (
	// maxLoopIterations is the maximum number of tool loop iterations before intervention
	// The LoopDetector has maxIterations=10, so we intervene at 9 to allow one more cycle
	maxLoopIterations = 9
	// progressFeedbackInterval is how often to send progress updates during tool loops
	progressFeedbackInterval = 5
	// absoluteMaxLoops is a hard limit to prevent infinite loops even with interventions
	absoluteMaxLoops = 20
)

// StreamAssistantResponse handles streaming the assistant response (exported for Handler integration)
func (s *ChatService) StreamAssistantResponse(
	ctx context.Context,
	w http.ResponseWriter,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	modelOverride *string,
	getMessagesFn GetConversationMessagesFunc,
	updateMessageStatusFn func(string, string) error,
) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming unsupported")
		return fmt.Errorf("streaming unsupported")
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

	// Check for client disconnect early
	select {
	case <-ctx.Done():
		log.Printf("Client disconnected before streaming started: %s", assistantMessageID)
		return ctx.Err()
	default:
	}

	// Determine which model to use
	modelToUse := determineModel(conversation.Model, modelOverride)

	// Get conversation history
	messages, err := getMessagesFn(conversation.ID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		updateMessageStatusFn(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": "Failed to get conversation history"})
		return err
	}

	// Generate response with streaming
	err = s.streamChatResponse(ctx, userID, conversation, messages, modelToUse, assistantMessageID, sendEvent, getMessagesFn)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		updateMessageStatusFn(assistantMessageID, "failed")
		sendEvent("error", map[string]string{"error": err.Error()})
		return err
	}

	sendEvent("done", map[string]string{"message_id": assistantMessageID})
	return nil
}

// processStreamResponse processes a stream response and accumulates content and tool calls
func processStreamResponse(ctx context.Context, stream *openai.ChatCompletionStream, sendEvent StreamEventHandler) (string, []openai.ToolCall, error) {
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
func convertAndBroadcastToolCalls(toolCalls []openai.ToolCall, sendEvent StreamEventHandler) []models.ChatToolCall {
	var convertedToolCalls []models.ChatToolCall

	for _, tc := range toolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			log.Printf("Failed to unmarshal tool call arguments for %s during broadcast: %v", tc.Function.Name, err)
		}

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

// ProcessAssistantResponse handles the async (non-streaming) processing of the assistant response.
// This is used by SendMessageRoute and RegenerateMessageRoute when SSE streaming is not used.
func (s *ChatService) ProcessAssistantResponse(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	modelOverride *string,
	getMessagesFn GetConversationMessagesFunc,
	updateMessageStatusFn func(string, string) error,
) error {
	// Determine which model to use
	modelToUse := determineModel(conversation.Model, modelOverride)

	// Update status to processing
	if err := updateMessageStatusFn(assistantMessageID, "processing"); err != nil {
		log.Printf("Error updating message status to processing: %v", err)
		return err
	}

	// Get conversation history
	messages, err := getMessagesFn(conversation.ID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		updateMessageStatusFn(assistantMessageID, "failed")
		return err
	}

	// Generate response without streaming
	err = s.processChatResponse(ctx, userID, conversation, messages, modelToUse, assistantMessageID, getMessagesFn)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		updateMessageStatusFn(assistantMessageID, "failed")
		return err
	}

	return nil
}

// streamChatResponse generates a chat response with streaming and tool support
func (s *ChatService) streamChatResponse(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	messages []models.ChatMessage,
	model string,
	assistantMessageID string,
	sendEvent StreamEventHandler,
	getMessagesFn GetConversationMessagesFunc,
) error {
	// buildSystemPrompt will be called with appropriate callbacks
	systemPrompt, err := s.buildSystemPrompt(userID, conversation, nil, func(id int) (string, error) {
		return s.getUserInstructions(id), nil
	})
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
	tools := s.getToolRegistry().GetToolDefinitions()

	// Initialize loop detector for this conversation
	loopDetector := services.NewLoopDetector()

	// Track iteration count for progress feedback
	iterationCount := 0

	// Track total iterations to prevent infinite loops
	totalIterations := 0

	// Loop until no more tool calls are needed
	for totalIterations < absoluteMaxLoops {
		totalIterations++
		// Check if loop detector has reached max iterations
		if loopDetector.GetIteration() >= maxLoopIterations {
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
			userMsg := getUserFacingErrorMessage(err, "")
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
			userMsg := getUserFacingErrorMessage(fmt.Errorf("empty response"), errorMsg)
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
		if err = s.executeAndBroadcastToolCalls(currentToolCalls, userID, conversation.ID, assistantMessageID, model, sendEvent, loopDetector); err != nil {
			return err
		}

		// Get updated messages including tool responses for next iteration
		updatedMessages, err := getMessagesFn(conversation.ID)
		if err != nil {
			log.Printf("[ChatAgentV2] Error getting updated messages: %v", err)
			return err
		}

		// Rebuild system prompt for next iteration
		currentSystemPrompt, err := s.buildSystemPrompt(userID, conversation, nil, func(id int) (string, error) {
			return s.getUserInstructions(id), nil
		})
		if err != nil {
			log.Printf("[ChatAgentV2] Error rebuilding system prompt: %v", err)
			// Continue with previous system prompt
		} else {
			systemPrompt = currentSystemPrompt
		}

		// Convert updated messages to OpenAI format and continue loop
		converter := services.NewMessageConverter()
		openaiMessages = converter.ToOpenAI(updatedMessages, systemPrompt)
		// Loop continues to get LLM's final response with tool results
	}

	// If we exited due to hitting absolute max, provide an error message
	if totalIterations >= absoluteMaxLoops {
		errorMsg := "I'm having trouble completing this task due to a technical issue. The conversation has become too complex. Please try starting a new conversation or simplifying your request."
		return s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, &models.ChatMessage{
			ID:      assistantMessageID,
			Content: &errorMsg,
		})
	}

	return nil
}

// processChatResponse generates a chat response without streaming (async processing)
func (s *ChatService) processChatResponse(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	messages []models.ChatMessage,
	model string,
	assistantMessageID string,
	getMessagesFn GetConversationMessagesFunc,
) error {
	// buildSystemPrompt will be called with appropriate callbacks
	systemPrompt, err := s.buildSystemPrompt(userID, conversation, nil, func(id int) (string, error) {
		return s.getUserInstructions(id), nil
	})
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
	tools := s.getToolRegistry().GetToolDefinitions()

	// Initialize loop detector for this conversation
	loopDetector := services.NewLoopDetector()

	// Track total iterations to prevent infinite loops
	totalIterations := 0

	// Loop until no more tool calls are needed
	for totalIterations < absoluteMaxLoops {
		totalIterations++
		// Check if loop detector has reached max iterations
		if loopDetector.GetIteration() >= maxLoopIterations {
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

		// Make non-streaming request to LLM
		response, err := services.ExecuteLLMToolRequest(ctx, client, openaiMessages, tools)
		if err != nil {
			userMsg := getUserFacingErrorMessage(err, "")
			// Check for context length error specifically
			if services.IsContextLengthError(err) {
				finalMessage := &models.ChatMessage{
					ID:      assistantMessageID,
					Content: &userMsg,
				}
				updateErr := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
				if updateErr != nil {
					log.Printf("Error updating message with context length error: %v", updateErr)
				}
				return err
			}
			// For other errors, update message with user-friendly message
			finalMessage := &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &userMsg,
			}
			updateErr := s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, finalMessage)
			if updateErr != nil {
				log.Printf("Error updating message with error: %v", updateErr)
			}
			return err
		}

		if len(response.Choices) == 0 {
			log.Printf("[EmptyResponse] LLM returned no choices")
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			s.updateAssistantMessageWithError(assistantMessageID, errorMsg)
			return fmt.Errorf("empty response from LLM")
		}

		choice := response.Choices[0]
		currentContent := choice.Message.Content
		currentToolCalls := choice.Message.ToolCalls

		// Handle empty response
		if strings.TrimSpace(currentContent) == "" && len(currentToolCalls) == 0 {
			log.Printf("[EmptyResponse] LLM returned empty content for conversation %s", conversation.ID)
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			s.updateAssistantMessageWithError(assistantMessageID, errorMsg)
			return fmt.Errorf("empty response from LLM")
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

		// Convert tool calls
		var toolCalls []models.ChatToolCall
		for _, tc := range currentToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Printf("Failed to unmarshal tool call arguments for %s: %v", tc.Function.Name, err)
			}

			toolCalls = append(toolCalls, models.ChatToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: models.ChatToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			})
		}

		// Update message with tool calls
		if err = s.updateAssistantMessageWithToolCalls(assistantMessageID, &currentContent, toolCalls); err != nil {
			return err
		}

		// Execute tool calls
		if err = s.executeAndSaveToolCalls(currentToolCalls, userID, conversation.ID, assistantMessageID, model, loopDetector); err != nil {
			log.Printf("Error executing tool calls: %v", err)
			return err
		}

		// Get updated messages including tool responses for next iteration
		updatedMessages, err := getMessagesFn(conversation.ID)
		if err != nil {
			log.Printf("[ChatAgent] Error getting updated messages: %v", err)
			return err
		}

		// Rebuild system prompt for next iteration
		currentSystemPrompt, err := s.buildSystemPrompt(userID, conversation, nil, func(id int) (string, error) {
			return s.getUserInstructions(id), nil
		})
		if err != nil {
			log.Printf("[ChatAgent] Error rebuilding system prompt: %v", err)
			// Continue with previous system prompt
		} else {
			systemPrompt = currentSystemPrompt
		}

		// Convert updated messages to OpenAI format and continue loop
		converter = services.NewMessageConverter()
		openaiMessages = converter.ToOpenAI(updatedMessages, systemPrompt)
		// Loop continues to get LLM's final response with tool results
	}

	// If we exited due to hitting absolute max, provide an error message
	if totalIterations >= absoluteMaxLoops {
		errorMsg := "I'm having trouble completing this task due to a technical issue. The conversation has become too complex. Please try starting a new conversation or simplifying your request."
		return s.updateAssistantMessage(userID, conversation.ID, assistantMessageID, &models.ChatMessage{
			ID:      assistantMessageID,
			Content: &errorMsg,
		})
	}

	return nil
}
