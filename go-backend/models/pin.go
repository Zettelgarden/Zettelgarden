package models

import (
	"time"
)

// StarredCard represents a card that has been starred by a user
type StarredCard struct {
	ID        int       `json:"id"`
	CardPK    int       `json:"card_pk"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Card      Card      `json:"card,omitempty"` // Optional embedded card for API responses
}

// StarredCardResponse is used for API responses that include the full card data
type StarredCardResponse struct {
	ID        int       `json:"id"`
	CardPK    int       `json:"card_pk"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Card      Card      `json:"card"`
}
