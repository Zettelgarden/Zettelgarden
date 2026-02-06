// Package chat provides conversation management functionality for the chat service.
package chat

import (
	"context"
	"go-backend/models"
	"go-backend/prompts"
	"go-backend/services"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// GenerateTitleIfNeeded generates a title for a conversation if it doesn't have one.
func (s *Service) GenerateTitleIfNeeded(
	ctx context.Context,
	userID int,
	conversation *models.ChatConversation,
	getMessages func(conversationID string) ([]models.ChatMessage, error),
) error {
	if conversation.Title != nil && *conversation.Title != "" {
		return nil
	}

	messages, err := getMessages(conversation.ID)
	if err != nil || len(messages) == 0 {
		return nil
	}

	// Find first user message
	var userContent string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != nil {
			userContent = *msg.Content
			break
		}
	}

	if userContent == "" {
		return nil
	}

	generatedTitle := s.generateConversationTitle(ctx, userID, userContent)

	// Update the title in the database
	query := `
		UPDATE chat_conversations
		SET title = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = s.db.Exec(query, generatedTitle, conversation.ID)
	return err
}

// generateConversationTitle generates a concise title for a conversation based on the first user message.
func (s *Service) generateConversationTitle(ctx context.Context, userID int, userMessage string) string {
	titlePrompt, err := prompts.GetTitleGeneratorPrompt()
	if err != nil {
		s.logError("Error loading title generation prompt: %v", err)
		return truncateString(userMessage, 40)
	}

	client := s.createLLMClient(userID, "google/gemini-2.5-flash-lite")

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: titlePrompt},
		{Role: openai.ChatMessageRoleUser, Content: userMessage},
	}

	resp, err := services.ExecuteLLMRequest(ctx, client, messages)
	if err != nil {
		s.logError("Error generating conversation title: %v", err)
		return truncateString(userMessage, 40)
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	if title == "" {
		return truncateString(userMessage, 40)
	}

	return title
}

// GetLatestUserMessage retrieves the most recent user message in a conversation.
func (s *Service) GetLatestUserMessage(conversationID string) string {
	var content *string
	query := `
		SELECT content
		FROM chat_messages
		WHERE conversation_id = $1 AND role = 'user' AND content IS NOT NULL
		ORDER BY sequence_number DESC
		LIMIT 1
	`
	err := s.db.QueryRow(query, conversationID).Scan(&content)
	if err != nil || content == nil {
		return ""
	}
	return *content
}

// DetermineModel determines which model to use for a conversation, preferring override if provided.
func (s *Service) DetermineModel(conversation *models.ChatConversation, modelOverride *string) string {
	if modelOverride != nil && *modelOverride != "" {
		return *modelOverride
	}
	return conversation.Model
}

// getLatestUserMessage is an internal helper (kept for backward compatibility).
func (s *Service) getLatestUserMessage(conversationID string) string {
	return s.GetLatestUserMessage(conversationID)
}

// determineModel is an internal helper (kept for backward compatibility).
func (s *Service) determineModel(conversation *models.ChatConversation, modelOverride *string) string {
	return s.DetermineModel(conversation, modelOverride)
}

// createLLMClient creates an LLM client configured for chat operations.
func (s *Service) createLLMClient(userID int, model string) *models.LLMClient {
	isTesting := s.srv != nil && s.srv.Testing
	client := services.NewDefaultClient(s.db, userID, isTesting)
	client.Model = model
	client.RequestType = "chat"
	return client
}
