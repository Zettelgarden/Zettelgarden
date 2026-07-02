package models

import (
	"database/sql"

	openai "github.com/sashabaranov/go-openai"
)

// LLMClient wraps an OpenAI-compatible client with per-request metadata.
// It is shared across all AI features (memory, entity recognition,
// summarization, etc.).
type LLMClient struct {
	Client      *openai.Client
	Testing     bool
	Model       string // Just the model identifier string
	UserID      int
	DB          *sql.DB
	RequestType string // e.g. "analysis", "summarization", "memory"
}

const MODEL = "gpt-3.5-turbo"
