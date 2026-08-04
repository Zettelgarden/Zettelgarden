package models

import "time"

type Summarization struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	InputText        string    `json:"input_text"`
	Status           string    `json:"status"`
	Result           string    `json:"result,omitempty"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Cost             float64   `json:"cost"`
	Model            string    `json:"model"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Argument struct {
	Argument   string `json:"argument"`
	Importance int    `json:"importance"`
}

type SectionAnalysis struct {
	Section string        `json:"section"`
	Theses  []ThesisEntry `json:"theses"`
}

type ThesisEntry struct {
	Thesis    string     `json:"thesis"`
	Arguments []Argument `json:"arguments"`
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	TotalCost        float64
}
