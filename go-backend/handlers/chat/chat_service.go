// Package chat provides the core chat service for handling AI agent conversations.
// It encapsulates all chat-related business logic, LLM interactions, and tool execution.
package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"go-backend/server"
	"log"
	"net/http"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/typesense/typesense-go/typesense"
)

// Service handles all chat operations including LLM interactions and tool execution.
// It provides dependency injection for database, LLM client, and tool registry.
type Service struct {
	db              *sql.DB
	llmClient       *models.LLMClient
	registry        *services.ToolRegistry
	typesenseClient *typesense.Client
	srv             *server.Server // For testing flag
	logger          *log.Logger
}

// NewService creates a new chat service with all required dependencies.
func NewService(
	db *sql.DB,
	llmClient *models.LLMClient,
	registry *services.ToolRegistry,
	typesenseClient *typesense.Client,
	srv *server.Server,
	logger *log.Logger,
) *Service {
	if registry == nil {
		registry = services.NewToolRegistry()
	}
	return &Service{
		db:              db,
		llmClient:       llmClient,
		registry:        registry,
		typesenseClient: typesenseClient,
		srv:             srv,
		logger:          logger,
	}
}

// ProcessAssistantResponse handles the async processing of the assistant response.
func (s *Service) ProcessAssistantResponse(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	modelOverride *string,
	getMessages func(conversationID string) ([]models.ChatMessage, error),
	updateMessage func(messageID string, content *string, toolCallsJSON *string, status string) error,
) error {
	// Update status to processing
	if err := updateMessage(assistantMessageID, nil, nil, "processing"); err != nil {
		s.logError("Error updating message status to processing: %v", err)
		return err
	}

	// Generate title if needed (with timeout for async processing)
	titleCtx, cancelTitle := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTitle()
	if err := s.GenerateTitleIfNeeded(titleCtx, userID, conversation, getMessages); err != nil {
		s.logError("Error generating title: %v", err)
		// Don't fail the whole request for title generation errors
	}

	// Get conversation history for LLM
	messages, err := getMessages(conversation.ID)
	if err != nil {
		s.logError("Error getting conversation history: %v", err)
		updateMessage(assistantMessageID, nil, nil, "failed")
		return err
	}

	// Determine which model to use
	modelToUse := s.determineModel(conversation, modelOverride)

	// Generate LLM response with tools
	finalAssistantMessage, err := s.GenerateChatResponse(ctx, userID, conversation, messages, modelToUse, assistantMessageID)
	if err != nil {
		s.logError("Error generating chat response: %v", err)
		updateMessage(assistantMessageID, nil, nil, "failed")
		return err
	}

	// Convert tool calls to JSON if present
	var toolCallsJSON *string
	if finalAssistantMessage.ToolCalls != nil && len(finalAssistantMessage.ToolCalls) > 0 {
		toolCallsData, err := json.Marshal(finalAssistantMessage.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsStr := string(toolCallsData)
		toolCallsJSON = &toolCallsStr
	}

	// Update the assistant message with final content and mark as completed
	if err := updateMessage(assistantMessageID, finalAssistantMessage.Content, toolCallsJSON, "completed"); err != nil {
		return fmt.Errorf("failed to update assistant message: %w", err)
	}

	// Update user memory based on chat exchange (async) - if there's actual content
	if finalAssistantMessage.Content != nil && *finalAssistantMessage.Content != "" {
		go func() {
			userMessage := s.GetLatestUserMessage(conversation.ID)
			if userMessage != "" {
				s.GenerateChatMemory(userID, userMessage, *finalAssistantMessage.Content)
			}
		}()
	}

	return nil
}

// GenerateChatResponse generates an LLM response with tool calling support (non-streaming).
func (s *Service) GenerateChatResponse(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	messages []models.ChatMessage,
	model string,
	assistantMessageID string,
) (*models.ChatMessage, error) {
	systemPrompt, err := s.buildSystemPrompt(ctx, userID, conversation)
	if err != nil {
		return nil, err
	}

	converter := services.NewMessageConverter()
	openaiMessages := converter.ToOpenAI(messages, systemPrompt)

	// Check if compaction is needed and perform it proactively
	openaiMessages, err = s.compactConversationIfNeeded(ctx, userID, openaiMessages, model)
	if err != nil {
		s.logError("Error during compaction: %v", err)
		// Continue anyway - compaction is an optimization, not required
	}

	// Create LLM client
	client := s.createLLMClient(userID, model)

	// Get tools definitions
	tools := s.registry.GetToolDefinitions()

	// Initialize loop detector for this conversation
	loopDetector := services.NewLoopDetector()

	// Track iteration count for progress logging
	iterationCount := 0
	const progressFeedbackInterval = 5

	// Loop until no more tool calls are needed
	for {
		// Check loop detector
		if loopDetector.GetIteration() >= 9 {
			s.logError("[LoopDetector] Max iterations reached for conversation %s", conversation.ID)
			interventionMsg := loopDetector.GetInterventionMessage()
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: interventionMsg,
			})
			loopDetector.Reset()
		}

		resp, err := services.ExecuteLLMToolRequest(ctx, client, openaiMessages, tools)
		if err != nil {
			return s.handleLLMError(ctx, err, userID, conversation, assistantMessageID)
		}

		// Validate response
		if len(resp.Choices) == 0 {
			s.logError("Warning: LLM returned no choices for conversation %s", conversation.ID)
			errorMsg := GetUserFacingMessage(fmt.Errorf("no choices"), "")
			return &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &errorMsg,
			}, nil
		}

		assistantMessage := resp.Choices[0].Message

		// Handle empty response
		if strings.TrimSpace(assistantMessage.Content) == "" && len(assistantMessage.ToolCalls) == 0 {
			s.logError("[EmptyResponse] LLM returned empty content for conversation %s", conversation.ID)
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			userMsg := GetUserFacingMessage(fmt.Errorf("empty response"), errorMsg)
			return &models.ChatMessage{
				ID:      assistantMessageID,
				Content: &userMsg,
			}, nil
		}

		// Increment iteration counter and log progress
		iterationCount++
		if iterationCount%progressFeedbackInterval == 0 && iterationCount > 0 {
			s.logInfo("[Progress] Chat response iteration %d for conversation %s", iterationCount, conversation.ID)
		}

		// If no tool calls, return the final message
		if len(assistantMessage.ToolCalls) == 0 {
			content := assistantMessage.Content
			return &models.ChatMessage{
				ID:             assistantMessageID,
				Content:        &content,
				ToolCalls:      ConvertToolCalls(assistantMessage.ToolCalls),
				Role:           "assistant",
				CreatedAt:      time.Now(),
				SequenceNumber: 0, // Will be set by caller
			}, nil
		}

		// Execute tool calls
		if err := s.ExecuteToolCalls(assistantMessage.ToolCalls, userID, conversation, assistantMessageID, model, loopDetector); err != nil {
			return nil, err
		}

		// OpenAI messages need to be rebuilt with tool responses
		// This is a limitation of the current design - we need access to the message fetcher
		// For now, break the loop to avoid infinite iteration
		s.logError("Warning: Tool loop continuation requires message fetcher callback")
		break
	}

	return &models.ChatMessage{
		ID:      assistantMessageID,
		Content: strPtr("I apologize, but I encountered an issue processing the tool calls."),
	}, nil
}

// StreamChatResponse generates a chat response with streaming and tool support.
func (s *Service) StreamChatResponse(
	ctx context.Context,
	w http.ResponseWriter,
	userID int,
	conversation *models.ChatConversation,
	messages []models.ChatMessage,
	model string,
	assistantMessageID string,
	sendEvent func(string, interface{}) error,
	updateMessage func(messageID string, content *string, toolCallsJSON *string, status string) error,
) error {
	systemPrompt, err := s.buildSystemPrompt(ctx, userID, conversation)
	if err != nil {
		return err
	}

	converter := services.NewMessageConverter()
	openaiMessages := converter.ToOpenAI(messages, systemPrompt)

	// Check if compaction is needed
	openaiMessages, err = s.compactConversationIfNeeded(ctx, userID, openaiMessages, model)
	if err != nil {
		s.logError("Error during compaction: %v", err)
	}

	// Create LLM client
	client := s.createLLMClient(userID, model)

	// Get tools definitions
	tools := s.registry.GetToolDefinitions()

	// Initialize loop detector
	loopDetector := services.NewLoopDetector()

	// Track iteration count
	iterationCount := 0
	const progressFeedbackInterval = 5

	// Loop until no more tool calls
	for {
		if loopDetector.GetIteration() >= 9 {
			s.logError("[LoopDetector] Max iterations reached for conversation %s", conversation.ID)
			interventionMsg := loopDetector.GetInterventionMessage()
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: interventionMsg,
			})
			loopDetector.Reset()
		}

		stream, err := services.StreamLLMToolRequest(ctx, client, openaiMessages, tools)
		if err != nil {
			userMsg := GetUserFacingMessage(err, "")
			if services.IsContextLengthError(err) {
				_ = updateMessage(assistantMessageID, &userMsg, nil, "failed")
			}
			return sendEvent("error", map[string]string{"error": userMsg})
		}

		// Process stream and collect results
		currentContent, currentToolCalls, err := s.processStreamResponse(ctx, stream, sendEvent)
		if err != nil {
			return err
		}

		// Handle empty response
		if strings.TrimSpace(currentContent) == "" && len(currentToolCalls) == 0 {
			s.logError("[EmptyResponse] Streaming LLM returned empty content for conversation %s", conversation.ID)
			errorMsg := "I'm having trouble connecting to my language model right now. It returned an empty response. Please try again in a moment."
			userMsg := GetUserFacingMessage(fmt.Errorf("empty response"), errorMsg)
			_ = updateMessage(assistantMessageID, &userMsg, nil, "failed")
			return sendEvent("error", map[string]string{"error": userMsg})
		}

		// Increment iteration counter
		iterationCount++
		if iterationCount%progressFeedbackInterval == 0 && iterationCount > 0 {
			progressMsg := fmt.Sprintf("Working on it... (step %d)", iterationCount)
			sendEvent("progress", map[string]interface{}{
				"step":    iterationCount,
				"message": progressMsg,
			})
			s.logInfo("[Progress] Chat response iteration %d for conversation %s", iterationCount, conversation.ID)
		}

		// If no tool calls, we're done
		if len(currentToolCalls) == 0 {
			if strings.TrimSpace(currentContent) == "" {
				currentContent = "I apologize, but I wasn't able to generate a proper response."
			}
			return updateMessage(assistantMessageID, &currentContent, nil, "completed")
		}

		// Convert tool calls and broadcast events
		toolCalls := ConvertToolCalls(currentToolCalls)
		for _, tc := range currentToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			sendEvent("tool_call", map[string]interface{}{
				"id":        tc.ID,
				"name":      tc.Function.Name,
				"arguments": args,
			})
		}

		// Update message with tool calls
		toolCallsJSON, _ := json.Marshal(toolCalls)
		toolCallsStr := string(toolCallsJSON)
		if err := updateMessage(assistantMessageID, &currentContent, &toolCallsStr, "processing"); err != nil {
			return err
		}

		// Execute tool calls
		if err := s.ExecuteToolCallsWithBroadcast(currentToolCalls, userID, conversation, assistantMessageID, model, sendEvent, loopDetector); err != nil {
			return err
		}

		// TODO: Fetch updated messages for next iteration
		// This requires a callback to the handler
		s.logError("Warning: Tool loop continuation requires message fetcher callback")
		break
	}

	return nil
}

// Helper methods

func (s *Service) logError(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func (s *Service) logInfo(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

// GenerateChatMemory updates user memory based on a chat exchange.
func (s *Service) GenerateChatMemory(userID int, userMessage, assistantMessage string) {
	// TODO: Implement memory generation
	s.logInfo("Generating chat memory for user %d", userID)
}
