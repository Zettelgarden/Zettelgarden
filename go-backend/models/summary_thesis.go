package models

import "time"

// SummaryThesis represents a thesis identified during summarization analysis.
type SummaryThesis struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	CardPK          int       `json:"card_pk"`
	SummarizationID int       `json:"summarization_id"`
	SectionID       int       `json:"section_id"`
	Thesis          string    `json:"thesis"`
	CreatedAt       time.Time `json:"created_at"`
}
