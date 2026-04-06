package models

import "time"

// CreateAgentRequest represents the payload for creating a new agent.
// Name is required; description is optional.
type CreateAgentRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// Agent represents an autonomous agent that can interact with the system.
// Agents have unique API keys and track usage for auditing purposes.
type Agent struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// CreateAgentResponse is returned when a new agent is created.
// It includes the full Agent details plus the API key, which is only
// shown once at creation time and cannot be retrieved later.
type CreateAgentResponse struct {
	Agent
	APIKey string `json:"api_key"` // Only shown once!
}

// AgentActivityLog records actions performed by agents for auditing.
// Each log entry captures what action was taken, on what target, and
// any additional details relevant to the action.
type AgentActivityLog struct {
	ID         int                    `json:"id"`
	AgentID    int                    `json:"agent_id"`
	Action     string                 `json:"action"`
	TargetType string                 `json:"target_type"`
	TargetID   *int                   `json:"target_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
