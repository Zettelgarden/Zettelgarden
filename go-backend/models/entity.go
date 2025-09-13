package models

import (
	"time"
)

type Entity struct {
	ID          int          `json:"id"`
	UserID      int          `json:"user_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CardCount   int          `json:"card_count"`
	CardPK      *int         `json:"card_pk"`
	Card        *PartialCard `json:"card,omitempty"`
}

type EntityCardJunction struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	EntityID  int       `json:"entity_id"`
	CardPK    int       `json:"card_pk"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
