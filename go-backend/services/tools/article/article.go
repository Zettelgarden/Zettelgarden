// Package article provides article-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The article domain contains tools for parsing URLs and creating article cards.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// For this implementation, the article domain package demonstrates the pattern
// for splitting article tools into a separate domain package. The registration
// is handled in services/article_tools.go to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions (ParseURL, CreateArticle)
// 2. Domain-specific business logic for article operations
// 3. URL parsing and content extraction support
//
// Tools provided:
// - parse_url: Parse a URL to extract article content for preview
// - create_article: Create a new article card from a URL
package article

import (
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

// ParseURL extracts article content from a URL for preview.
// This is the domain data access function for URL parsing operations.
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

// CreateCardFunc is the function signature for creating a card.
// This allows the domain package to call CreateCard without circular dependencies.
type CreateCardFunc func(db models.Database, userID int, params models.EditCardParams) (models.Card, error)

// CreateArticle creates a card from a URL with auto-parsed content.
// This is the domain data access function for article creation operations.
// It requires CreateCard to be passed as a dependency for card creation.
func CreateArticle(db models.Database, userID int, url, cardID, tags string, createCardFunc CreateCardFunc) (*models.Card, error) {
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

	card, err := createCardFunc(db, userID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	return &card, nil
}
