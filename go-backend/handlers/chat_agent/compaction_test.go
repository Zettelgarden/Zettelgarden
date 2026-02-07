package chat_agent

import (
	openai "github.com/sashabaranov/go-openai"
	"testing"
)

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		messages []openai.ChatCompletionMessage
		minTokens int
		maxTokens int
	}{
		{
			name:     "empty messages",
			messages: []openai.ChatCompletionMessage{},
			minTokens: 0,
			maxTokens: 10,
		},
		{
			name: "single short message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
			},
			minTokens: 1,  // 5 chars / 4 = 1.25 + overhead
			maxTokens: 20,
		},
		{
			name: "long message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: string(make([]byte, 1000))},
			},
			minTokens: 240,  // 1000 chars / 4 = 250 - overhead
			maxTokens: 270,
		},
		{
			name: "message with tool calls",
			messages: []openai.ChatCompletionMessage{
				{
					Role:    "assistant",
					Content: "Let me help",
					ToolCalls: []openai.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: openai.FunctionCall{
								Name:      "search",
								Arguments: `{"query":"test"}`,
							},
						},
					},
				},
			},
			minTokens: 10,  // Should account for tool call content
			maxTokens: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokenCount(tt.messages)
			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("estimateTokenCount() = %v, expected between %v and %v", result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestGetModelContextLimit(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{
			name:     "gemini flash",
			model:    "google/gemini-2.5-flash",
			expected: 1000000,
		},
		{
			name:     "gemini pro",
			model:    "google/gemini-2.5-pro",
			expected: 2000000,
		},
		{
			name:     "gpt-4o-mini",
			model:    "openai/gpt-4o-mini",
			expected: 128000,
		},
		{
			name:     "claude sonnet",
			model:    "anthropic/claude-sonnet-4",
			expected: 200000,
		},
		{
			name:     "unknown model returns default",
			model:    "unknown/model",
			expected: 100000,
		},
		{
			name:     "empty string returns default",
			model:    "",
			expected: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModelContextLimit(tt.model)
			if result != tt.expected {
				t.Errorf("getModelContextLimit(%v) = %v, expected %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			s:        "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "string equals max",
			s:        "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "string longer than max",
			s:        "Hello world",
			maxLen:   5,
			expected: "Hello...",
		},
		{
			name:     "empty string",
			s:        "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithEllipsis(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateWithEllipsis(%q, %v) = %v, expected %v", tt.s, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestDetermineModel(t *testing.T) {
	tests := []struct {
		name            string
		conversationModel string
		modelOverride    *string
		expected         string
	}{
		{
			name:            "override takes precedence",
			conversationModel: "gpt-4",
			modelOverride:    strPtr("gpt-3.5"),
			expected:         "gpt-3.5",
		},
		{
			name:            "empty override is ignored",
			conversationModel: "gpt-4",
			modelOverride:    strPtr(""),
			expected:         "gpt-4",
		},
		{
			name:            "nil override uses conversation model",
			conversationModel: "gpt-4",
			modelOverride:    nil,
			expected:         "gpt-4",
		},
		{
			name:            "whitespace override is used",
			conversationModel: "gpt-4",
			modelOverride:    strPtr("  "),
			expected:         "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineModel(tt.conversationModel, tt.modelOverride)
			if result != tt.expected {
				t.Errorf("determineModel(%q, %v) = %v, expected %v", tt.conversationModel, tt.modelOverride, result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestCompactionConstants(t *testing.T) {
	// Verify constants are set to expected values
	if compactionThresholdPercent != 0.60 {
		t.Errorf("compactionThresholdPercent = %v, expected 0.60", compactionThresholdPercent)
	}
	if minMessagesForCompaction != 10 {
		t.Errorf("minMessagesForCompaction = %v, expected 10", minMessagesForCompaction)
	}
	if compactionSplitPercent != 0.50 {
		t.Errorf("compactionSplitPercent = %v, expected 0.50", compactionSplitPercent)
	}

	// Also verify streaming constants
	if maxLoopIterations != 9 {
		t.Errorf("maxLoopIterations = %v, expected 9", maxLoopIterations)
	}
	if progressFeedbackInterval != 5 {
		t.Errorf("progressFeedbackInterval = %v, expected 5", progressFeedbackInterval)
	}
}
