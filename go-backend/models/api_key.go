package models

import (
	"time"
)

// APIKey represents an API key in the database
type APIKey struct {
	ID          int        `json:"id" db:"id"`
	UserID      int        `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	KeyHash     string     `json:"-" db:"key_hash"` // Never serialize the hash
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	RevokedAt   *time.Time `json:"revoked_at" db:"revoked_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	Description string     `json:"description" db:"description"`
}

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// APIKeyResponse represents API key data sent to clients (without sensitive info)
type APIKeyResponse struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	IsActive    bool       `json:"is_active"`
	Description string     `json:"description"`
}

// CreateAPIKeyResponse represents the response when creating a new API key
// This includes the actual key value that is shown only once
type CreateAPIKeyResponse struct {
	APIKeyResponse
	Key string `json:"key"` // The actual API key - shown only on creation
}

// ListAPIKeysResponse represents the list of API keys for a user
type ListAPIKeysResponse struct {
	APIKeys []APIKeyResponse `json:"api_keys"`
}