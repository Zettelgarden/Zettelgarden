package llms

import (
	"context"
	"database/sql"
	"go-backend/models"
	"log"
	"net/http"
	"os"

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
		Client:  openai.NewClientWithConfig(config),
		Testing: false,
		UserID:  userID,
		DB:      db,
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

func ExecuteLLMRequest(c *models.LLMClient, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	resp, err := c.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    c.Model,
			Messages: messages,
		},
	)

	if err == nil {
		logLLMRequest(c, resp)
	}

	return resp, err
}

func logLLMRequest(c *models.LLMClient, resp openai.ChatCompletionResponse) {
	// fire and forget
	go func() {
		// simple model pricing table (per 1k tokens in USD)
		var modelPricing = map[string]struct {
			PromptPer1K     float64
			CompletionPer1K float64
		}{
			"google/gemini-2.5-flash": {PromptPer1K: 0.0003, CompletionPer1K: 0.0025},
			"google/gemini-2.5-pro":   {PromptPer1K: 0.00125, CompletionPer1K: 0.010},
			"openai/gpt-5-chat":       {PromptPer1K: 0.00125, CompletionPer1K: 0.010},
		}

		var cost *float64
		if pricing, ok := modelPricing[c.Model]; ok {
			est := float64(resp.Usage.PromptTokens)/1000.0*pricing.PromptPer1K +
				float64(resp.Usage.CompletionTokens)/1000.0*pricing.CompletionPer1K
			cost = &est
		}

		_, err := c.DB.Exec(`
			INSERT INTO llm_query_log (user_id, model, prompt_tokens, completion_tokens, cost_usd)
			VALUES ($1, $2, $3, $4, $5)
		`, c.UserID, c.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, cost)
		if err != nil {
			log.Printf("Error logging llm request: %v", err)
		}
	}()
}
