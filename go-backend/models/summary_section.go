package models

import "time"

// SummarySection represents a section identified during summarization analysis.
type SummarySection struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	CardPK          int       `json:"card_pk"`
	SummarizationID int       `json:"summarization_id"`
	SectionTitle    string    `json:"section_title"`
	CreatedAt       time.Time `json:"created_at"`
}
