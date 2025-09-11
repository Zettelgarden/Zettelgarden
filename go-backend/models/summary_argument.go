package models

import "time"

// SummaryArgument represents an argument identified during summarization analysis.
type SummaryArgument struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	CardPK          int       `json:"card_pk"`
	SummarizationID int       `json:"summarization_id"`
	ThesisID        int       `json:"thesis_id"`
	Argument        string    `json:"argument"`
	Importance      int       `json:"importance"`
	CreatedAt       time.Time `json:"created_at"`
}
