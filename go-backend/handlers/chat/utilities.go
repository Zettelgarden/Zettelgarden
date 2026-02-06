// Package chat provides utility functions for the chat service.
package chat

import (
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// truncateString truncates a string to a maximum length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}

// getModelContextLimit returns the context token limit for a given model.
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
	return 100000
}

// estimateTokenCount estimates the token count for a list of messages.
// This is a rough estimation based on character count.
func estimateTokenCount(messages []openai.ChatCompletionMessage) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Function.Name)
			totalChars += len(tc.Function.Arguments)
		}
	}
	return (totalChars / 4) + (len(messages) * 10)
}

// isEmptyContent checks if content is empty or whitespace only.
func isEmptyContent(content *string) bool {
	return content == nil || strings.TrimSpace(*content) == ""
}

// hasToolCalls checks if there are any tool calls.
func hasToolCalls(toolCalls []openai.ToolCall) bool {
	return len(toolCalls) > 0
}

// coalesceString returns the first non-empty string from the arguments.
func coalesceString(strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}
	return ""
}

// safeStringPointer returns a pointer to the string, or nil if empty.
func safeStringPointer(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
