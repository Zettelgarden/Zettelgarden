package chat_agent

import (
	"context"
	"go-backend/prompts"
	"go-backend/services"
	"log"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// generateConversationTitle generates a title for a conversation based on the user's first message
func (s *ChatService) generateConversationTitle(ctx context.Context, userID int, userMessage string) string {
	// Load title generation prompt
	titlePrompt, err := prompts.GetTitleGeneratorPrompt()
	if err != nil {
		log.Printf("Error loading title generation prompt: %v", err)
		// Fallback to truncated user message
		return truncateWithEllipsis(userMessage, 40)
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
		return truncateWithEllipsis(userMessage, 40)
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Ensure title isn't too long
	title = truncateWithEllipsis(title, 50)

	// If title generation failed or returned empty, use fallback
	if title == "" {
		return truncateWithEllipsis(userMessage, 40)
	}

	return title
}

// truncateWithEllipsis truncates a string to max length and adds "..." if truncated
func truncateWithEllipsis(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// determineModel selects the model to use based on override or conversation default
func determineModel(conversationModel string, modelOverride *string) string {
	if modelOverride != nil && *modelOverride != "" {
		return *modelOverride
	}
	return conversationModel
}
