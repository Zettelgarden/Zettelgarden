package models

import (
	"database/sql"
	"time"
)

// Fact represents a single extracted fact stored for a user + card.
type Fact struct {
	ID              int           `json:"id"`
	UserID          int           `json:"user_id"`
	CardPK          int           `json:"card_pk"`
	Fact            string        `json:"fact"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	SummarizationID sql.NullInt64 `json:"summarization_id"`
	ThesisID        sql.NullInt64 `json:"thesis_id"`
}
