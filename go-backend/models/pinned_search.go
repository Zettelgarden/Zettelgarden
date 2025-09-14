package models

import (
	"encoding/json"
	"time"
)

// StarredSearch represents a search that has been starred by a user
type StarredSearch struct {
	ID           int             `json:"id"`
	UserID       int             `json:"user_id"`
	Title        string          `json:"title"`
	SearchTerm   string          `json:"searchTerm"`
	SearchConfig json.RawMessage `json:"searchConfig"`
	CreatedAt    time.Time       `json:"created_at"`
}

// StarredSearchRequest is used for API requests to create a starred search
type StarredSearchRequest struct {
	Title        string          `json:"title"`
	SearchTerm   string          `json:"search_term"`
	SearchConfig json.RawMessage `json:"search_config"`
}
