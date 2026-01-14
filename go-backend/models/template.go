package models

import "time"

// CardTemplate represents a saved template for creating new cards
type CardTemplate struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTemplateParams contains the parameters for creating a new template
type CreateTemplateParams struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// UpdateTemplateParams contains the parameters for updating an existing template
type UpdateTemplateParams struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Body  string `json:"body"`
}
