package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// ChatConversation represents a chat conversation with enhanced metadata
type ChatConversation struct {
	ID           string    `json:"id"`
	UserID       int       `json:"user_id"`
	Title        *string   `json:"title"`
	Model        string    `json:"model"`
	SystemPrompt *string   `json:"system_prompt"`
	Starred      bool      `json:"starred"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count,omitempty"` // Computed field
}

// ChatMessage represents a single message in a conversation with tool calling support
type ChatMessage struct {
	ID             string           `json:"id"`
	ConversationID string           `json:"conversation_id"`
	Role           string           `json:"role"` // "user", "assistant", "system", "tool"
	Content        *string          `json:"content"`
	ToolCalls      []ChatToolCall   `json:"tool_calls,omitempty"`
	ToolCallID     *string          `json:"tool_call_id,omitempty"`
	SequenceNumber int              `json:"sequence_number"`
	CreatedAt      time.Time        `json:"created_at"`
}

// ChatToolCall represents a tool call within a message
type ChatToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "function"
	Function ChatToolCallFunction   `json:"function"`
}

// ChatToolCallFunction represents the function part of a tool call
type ChatToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ChatToolCallRecord tracks tool usage for analytics and quotas
type ChatToolCallRecord struct {
	ID              string                 `json:"id"`
	UserID          int                    `json:"user_id"`
	ConversationID  string                 `json:"conversation_id"`
	MessageID       string                 `json:"message_id"`
	ToolName        string                 `json:"tool_name"`
	ToolArguments   map[string]interface{} `json:"tool_arguments"`
	ToolResult      map[string]interface{} `json:"tool_result"`
	ExecutionTimeMs *int                   `json:"execution_time_ms"`
	CreatedAt       time.Time              `json:"created_at"`
}

// ChatUsageQuota tracks usage limits for rate limiting
type ChatUsageQuota struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	QuotaType    string    `json:"quota_type"`
	CurrentUsage int       `json:"current_usage"`
	MaxLimit     int       `json:"max_limit"`
	ResetDate    time.Time `json:"reset_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Legacy types for backward compatibility
// Conversation holds metadata for a chat session (legacy).
type Conversation struct {
	ID           string    `json:"id"`
	UserID       int       `json:"user_id"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	Model        string    `json:"model"`
	MessageCount int       `json:"message_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatMessage represents a single message in a conversation (legacy).
type LegacyChatMessage struct {
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

// JSON marshaling support for JSONB fields

// Type aliases for JSON marshaling
type ChatToolCallSlice []ChatToolCall
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for ChatToolCall slice
func (c ChatToolCallSlice) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for ChatToolCall slice
func (c *ChatToolCallSlice) Scan(value interface{}) error {
	if value == nil {
		*c = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, c)
}

// Value implements the driver.Valuer interface for map[string]interface{}
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for map[string]interface{}
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}
