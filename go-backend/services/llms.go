package services

import (
	"context"
	"database/sql"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func NewDefaultClient(db *sql.DB, userID int) *models.LLMClient {
	config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
	config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")
	client := NewClient(db, config, userID)
	client.Model = os.Getenv("ZETTEL_LLM_DEFAULT_MODEL")
	return client
}

func NewClient(db *sql.DB, config openai.ClientConfig, userID int) *models.LLMClient {
	config.HTTPClient = &http.Client{
		Transport: headerTransport{http.DefaultTransport},
	}

	return &models.LLMClient{
		Client:      openai.NewClientWithConfig(config),
		Testing:     false,
		UserID:      userID,
		DB:          db,
		RequestType: "other", // default to chat, can be overridden
	}
}

type headerTransport struct {
	http.RoundTripper
}

func (t headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("HTTP-Referer", "http://zettelgarden.com")
	req.Header.Set("X-Title", "Zettelgarden")

	return t.RoundTripper.RoundTrip(req)
}

func ExecuteLLMRequest(ctx context.Context, c *models.LLMClient, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	log.Printf("request")
	resp, err := c.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    c.Model,
			Messages: messages,
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
func ExecuteLLMToolRequest(ctx context.Context, c *models.LLMClient, messages []openai.ChatCompletionMessage, tools []openai.Tool) (openai.ChatCompletionResponse, error) {
	log.Printf("request")
	resp, err := c.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    c.Model,
			Messages: messages,
			Tools:    tools,
		},
	)
	log.Printf("response")

	if err == nil {
		logLLMRequest(c, resp, c.RequestType)
	} else {
		log.Printf("errored resp %v", resp)
		log.Printf("error in getting a response: %v", err)
	}

	return resp, err
}

// StreamEvent represents a streaming event sent to the client
type StreamEvent struct {
	Type string      `json:"type"` // "content", "tool_call", "tool_result", "done", "error"
	Data interface{} `json:"data"`
}

// StreamLLMToolRequest executes an LLM request with tool support and streams the response
func StreamLLMToolRequest(ctx context.Context, c *models.LLMClient, messages []openai.ChatCompletionMessage, tools []openai.Tool) (*openai.ChatCompletionStream, error) {
	log.Printf("streaming request")
	stream, err := c.Client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model:    c.Model,
			Messages: messages,
			Tools:    tools,
			Stream:   true,
		},
	)

	if err != nil {
		log.Printf("error creating stream: %v", err)
		return nil, err
	}

	return stream, nil
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
		// simple model pricing table (per 1k tokens in USD)
		var modelPricing = map[string]struct {
			PromptPer1K     float64
			CompletionPer1K float64
		}{
			"google/gemini-2.5-flash":       {PromptPer1K: 0.0003, CompletionPer1K: 0.0025},
			"google/gemini-2.5-pro":         {PromptPer1K: 0.00125, CompletionPer1K: 0.010},
			"google/gemini-2.5-flash-lite":  {PromptPer1K: 0.0001, CompletionPer1K: 0.0004},
			"openai/gpt-5-chat":             {PromptPer1K: 0.00125, CompletionPer1K: 0.010},
			"openai/gpt-5.1-chat":           {PromptPer1K: 0.00125, CompletionPer1K: 0.010},
			"openai/gpt-5.2-chat":           {PromptPer1K: 0.00175, CompletionPer1K: 0.014},
			"openai/gpt-4o-mini":            {PromptPer1K: 0.00015, CompletionPer1K: 0.0006},
			"anthropic/claude-sonnet-4.5":   {PromptPer1K: 0.003, CompletionPer1K: 0.015},
			"google/gemini-3-pro-preview":   {PromptPer1K: 0.002, CompletionPer1K: 0.012},
			"google/gemini-3-flash-preview": {PromptPer1K: 0.0005, CompletionPer1K: 0.003},
		}

		var cost *float64
		if pricing, ok := modelPricing[c.Model]; ok {
			est := float64(resp.Usage.PromptTokens)/1000.0*pricing.PromptPer1K +
				float64(resp.Usage.CompletionTokens)/1000.0*pricing.CompletionPer1K
			cost = &est
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

func CreateChatCompletion(c *models.LLMClient, ctx context.Context, messages []models.ChatMessage, context string) (string, error) {
	var openaiMessages []openai.ChatCompletionMessage

	if context != "" {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. Please use the following context to answer the user's question:\n\n" + context,
		})
	}

	for _, msg := range messages {
		var content string
		if msg.Content != nil {
			content = *msg.Content
		}
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	resp, err := ExecuteLLMRequest(ctx, c, openaiMessages)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
