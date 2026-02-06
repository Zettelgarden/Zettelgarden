// Package chat provides conversation compaction functionality for the chat service.
package chat

import (
	"context"
	"fmt"
	"go-backend/prompts"
	"go-backend/services"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// CompactionThreshold is the percentage of context limit at which to trigger compaction.
	CompactionThreshold = 0.6

	// MinimumMessagesForCompaction is the minimum number of messages required before compacting.
	MinimumMessagesForCompaction = 10
)

// CompactConversationIfNeeded checks if the conversation needs compacting and performs compaction if necessary.
func (s *Service) CompactConversationIfNeeded(
	ctx context.Context,
	userID int,
	messages []openai.ChatCompletionMessage,
	model string,
) ([]openai.ChatCompletionMessage, error) {
	return s.compactConversationIfNeeded(ctx, userID, messages, model)
}

// compactConversationIfNeeded checks if the conversation needs compacting and performs compaction if necessary.
func (s *Service) compactConversationIfNeeded(
	ctx context.Context,
	userID int,
	messages []openai.ChatCompletionMessage,
	model string,
) ([]openai.ChatCompletionMessage, error) {
	tokenCount := s.estimateTokenCount(messages)
	contextLimit := getModelContextLimit(model)
	threshold := int(float64(contextLimit) * CompactionThreshold)

	if tokenCount < threshold || len(messages) < MinimumMessagesForCompaction {
		return messages, nil
	}

	s.logError("Conversation approaching token limit (%d/%d tokens). Performing compaction...", tokenCount, contextLimit)

	compacted, err := s.summarizeConversationHistory(ctx, userID, messages)
	if err != nil {
		s.logError("Error during compaction: %v", err)
		return messages, nil
	}

	return compacted, nil
}

// SummarizeConversationHistory summarizes older messages in a conversation to reduce token count.
func (s *Service) SummarizeConversationHistory(
	ctx context.Context,
	userID int,
	messages []openai.ChatCompletionMessage,
) ([]openai.ChatCompletionMessage, error) {
	return s.summarizeConversationHistory(ctx, userID, messages)
}

// summarizeConversationHistory summarizes older messages in a conversation to reduce token count.
func (s *Service) summarizeConversationHistory(
	ctx context.Context,
	userID int,
	messages []openai.ChatCompletionMessage,
) ([]openai.ChatCompletionMessage, error) {
	// Load compaction prompt
	compactionPrompt, err := prompts.GetConversationCompactionPrompt()
	if err != nil {
		s.logError("Error loading compaction prompt: %v", err)
		return messages, nil
	}

	// Split messages for summarization
	pivotPoint := len(messages) / 2
	systemPrompt := messages[0]
	olderMessages := messages[1:pivotPoint]

	// Build conversation text
	var conversationText strings.Builder
	conversationText.WriteString("# Conversation History to Summarize\n\n")
	for _, msg := range olderMessages {
		conversationText.WriteString(fmt.Sprintf("**%s**: %s\n\n", strings.Title(msg.Role), msg.Content))
	}

	// Use fast model for summarization
	client := s.createLLMClient(userID, "google/gemini-2.5-flash-lite")

	summaryMessages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: compactionPrompt},
		{Role: openai.ChatMessageRoleUser, Content: conversationText.String()},
	}

	resp, err := services.ExecuteLLMRequest(ctx, client, summaryMessages)
	if err != nil {
		s.logError("Error generating summary: %v", err)
		return messages, nil
	}

	summary := resp.Choices[0].Message.Content
	s.logInfo("Compacted %d messages into summary of %d characters", len(olderMessages), len(summary))

	// Rebuild messages
	compactedMessages := []openai.ChatCompletionMessage{systemPrompt, {Role: openai.ChatMessageRoleSystem, Content: summary}}
	compactedMessages = append(compactedMessages, messages[pivotPoint:]...)

	return compactedMessages, nil
}

// estimateTokenCount estimates the token count for a list of messages.
func (s *Service) estimateTokenCount(messages []openai.ChatCompletionMessage) int {
	return estimateTokenCount(messages)
}

// EstimateTokenCount estimates the token count for a list of messages.
func EstimateTokenCount(messages []openai.ChatCompletionMessage) int {
	return estimateTokenCount(messages)
}
