package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	readability "github.com/go-shiori/go-readability"
	"go-backend/models"
)

// ParseResult represents the result of parsing a URL
type ParseResult struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Author   string `json:"author,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
	SiteName string `json:"site_name,omitempty"`
	URL      string `json:"url"`
}

// ParseURL extracts article content from a URL for preview
func ParseURL(url string) (*ParseResult, error) {
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Parse the URL using readability
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Convert HTML content to markdown
	markdown, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		log.Printf("Failed to convert to markdown: %v", err)
		markdown = article.Content // Fallback to HTML
	}

	return &ParseResult{
		Title:    article.Title,
		Content:  markdown,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		SiteName: article.SiteName,
		URL:      url,
	}, nil
}

// CreateArticle creates a card from a URL with auto-parsed content
func CreateArticle(db *sql.DB, userID int, url, cardID, tags string) (*models.Card, error) {
	// Parse the URL
	parsed, err := ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Default tags if not provided
	if tags == "" {
		tags = "#to-read #reference"
	}

	// Combine tags with content
	body := fmt.Sprintf("%s\n\n%s", parsed.Content, tags)

	// Create the card
	params := models.EditCardParams{
		Title:  parsed.Title,
		Body:   body,
		Link:   url,
		CardID: cardID,
	}

	card, err := CreateCard(db, userID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	return &card, nil
}
