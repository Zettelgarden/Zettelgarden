package services

import (
	"context"
	"database/sql"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func init() {
	// Initialize AI configuration from config file
	// If the config file is not found, hardcoded defaults will be used
	configPath := GetConfigPath()
	if err := LoadAIConfig(configPath); err != nil {
		log.Printf("Warning: Failed to load AI config from %s: %v (using defaults)", configPath, err)
	}
}

func NewDefaultClient(db *sql.DB, userID int, testing bool) *models.LLMClient {
	config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
	config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")
	client := NewClient(db, config, userID, testing)
	client.Model = os.Getenv("ZETTEL_LLM_DEFAULT_MODEL")
	return client
}

func NewClient(db *sql.DB, config openai.ClientConfig, userID int, testing bool) *models.LLMClient {
	config.HTTPClient = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   120 * time.Second,
	}

	return &models.LLMClient{
		Client:      openai.NewClientWithConfig(config),
		Testing:     testing,
		UserID:      userID,
		DB:          db,
		RequestType: "other", // default to chat, can be overridden
	}
}

// mockLLMResponse creates a mock response for testing mode
func mockLLMResponse(c *models.LLMClient) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "Test response",
				},
				Index: 0,
			},
		},
		Usage: openai.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
}
func ExecuteLLMRequest(ctx context.Context, c *models.LLMClient, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	if c.Testing {
		return mockLLMResponse(c), nil
	}

	// Add stop sequences to prevent model from continuing user text
	stopSequences := []string{"\n\nUser:", "\n\nuser:", "\n\nUSER:"}

	log.Printf("request")
	resp, err := c.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    c.Model,
			Messages: messages,
			Stop:     stopSequences,
		},
	)
	log.Printf("response")

	if err == nil {
		logLLMRequest(c, resp, c.RequestType)
	} else {
		log.Printf("error in getting a response: %v", err)
	}

	return resp, err
}

// IsContextLengthError checks if the error is related to context length limits
func IsContextLengthError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	// Check for common context length error patterns
	return strings.Contains(errorStr, "context_length_exceeded") ||
		strings.Contains(errorStr, "maximum context length") ||
		strings.Contains(errorStr, "context length") ||
		strings.Contains(errorStr, "token limit") ||
		strings.Contains(errorStr, "too many tokens") ||
		strings.Contains(errorStr, "exceeds") ||
		strings.Contains(errorStr, "413")
}
func logLLMRequest(c *models.LLMClient, resp openai.ChatCompletionResponse, requestType string) {
	// fire and forget
	go func() {
		var cost *float64

		// Try to get pricing from config first
		if modelConfig, err := GetModelConfig(c.Model); err == nil {
			est := float64(resp.Usage.PromptTokens)/1000.0*modelConfig.PromptPer1K +
				float64(resp.Usage.CompletionTokens)/1000.0*modelConfig.CompletionPer1K
			cost = &est
		} else if pricing, ok := ValidChatModels[c.Model]; ok {
			// Fallback to ValidChatModels for backward compatibility
			est := float64(resp.Usage.PromptTokens)/1000.0*pricing.PromptPer1K +
				float64(resp.Usage.CompletionTokens)/1000.0*pricing.CompletionPer1K
			cost = &est
		} else {
			// Neither config lookup nor ValidChatModels fallback succeeded
			log.Printf("Warning: Unable to find pricing information for model %s - cost will not be calculated", c.Model)
		}

		_, err := c.DB.Exec(`
			INSERT INTO llm_query_log (user_id, model, prompt_tokens, completion_tokens, cost_usd, request_type)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, c.UserID, c.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, cost, requestType)
		if err != nil {
			log.Printf("Error logging llm request: %v", err)
		}
	}()
}
