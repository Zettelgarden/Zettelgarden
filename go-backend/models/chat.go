package models

import (
	"database/sql"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Conversation holds metadata for a chat session.
type Conversation struct {
	ID           string    `json:"id"`
	UserID       int       `json:"user_id"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	Model        string    `json:"model"`
	MessageCount int       `json:"message_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	ID             int       `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // "user" or "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	CardChunks     []int     `json:"card_chunks" db:"card_chunks"`
}

// LLMClient will be simplified for the new implementation.
type LLMClient struct {
	Client  *openai.Client
	Testing bool
	Model   string // Just the model identifier string
	UserID  int
	DB      *sql.DB
}

const MODEL = "gpt-3.5-turbo" // We can keep this constant for now.
