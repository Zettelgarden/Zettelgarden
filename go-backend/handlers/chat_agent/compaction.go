package chat_agent

import (
	"context"
	"fmt"
	"go-backend/prompts"
	"go-backend/services"
	"log"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// compactionThresholdPercent is the percentage of context limit at which to trigger compaction
	compactionThresholdPercent = 0.60
	// minMessagesForCompaction is the minimum number of messages before considering compaction
	minMessagesForCompaction = 10
	// compactionSplitPercent is the percentage of messages to summarize (oldest half)
	compactionSplitPercent = 0.50
)

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
	}

	if limit, ok := limits[model]; ok {
		return limit
	}
	// Default conservative limit
	return 100000
}

// summarizeConversationHistory creates a compact summary of older messages
func (s *ChatService) summarizeConversationHistory(ctx context.Context, userID int, messages []openai.ChatCompletionMessage, model string) (openai.ChatCompletionMessage, error) {
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
func (s *ChatService) compactConversationIfNeeded(ctx context.Context, userID int, messages []openai.ChatCompletionMessage, model string) ([]openai.ChatCompletionMessage, error) {
	tokenCount := estimateTokenCount(messages)
	contextLimit := getModelContextLimit(model)

	// Trigger compaction at threshold percentage of context limit
	threshold := int(float64(contextLimit) * compactionThresholdPercent)

	if tokenCount < threshold {
		return messages, nil
	}

	log.Printf("Conversation approaching token limit (%d/%d tokens). Performing compaction...", tokenCount, contextLimit)

	// Don't compact if conversation is too short
	if len(messages) < minMessagesForCompaction {
		log.Printf("Conversation too short to compact (%d messages)", len(messages))
		return messages, nil
	}

	// Split: oldest portion gets summarized, recent portion kept
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
