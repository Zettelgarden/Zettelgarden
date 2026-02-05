// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import (
	"fmt"
)

// RegisterArticleTools registers all article-related tools
func (tr *ToolRegistry) RegisterArticleTools() {
	RegisterTool(tr, "parse_url",
		"Parse a URL to extract article content (title, body, author, excerpt). Returns the parsed content for preview before creating a card.",
		handleParseURL,
		ToolParam{Name: "url", Type: "string", Required: true, Desc: "URL to parse and extract content from"},
	)

	RegisterTool(tr, "create_article",
		"Create a new article card from a URL. Automatically parses the URL, extracts content, adds the link, and tags with #to-read #reference by default.",
		handleCreateArticle,
		ToolParam{Name: "url", Type: "string", Required: true, Desc: "URL of the article to import"},
		ToolParam{Name: "card_id", Type: "string", Required: false, Desc: "Optional card_id (e.g., '1a'). Leave empty for auto-generated root ID."},
		ToolParam{Name: "tags", Type: "string", Required: false, Desc: "Optional custom tags (default: '#to-read #reference'). Provide as space-separated tags like '#tag1 #tag2'"},
	)
}

// Article tool handlers

func handleParseURL(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	url, err := getStringParam(args, "url")
	if err != nil {
		return nil, err
	}

	result, err := ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	return StructToMap(result), nil
}

func handleCreateArticle(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	url, err := getStringParam(args, "url")
	if err != nil {
		return nil, err
	}

	cardID, _ := getOptionalStringParam(args, "card_id")
	tags, _ := getOptionalStringParam(args, "tags")

	card, err := CreateArticle(ctx.DB, ctx.UserID, url, cardID, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create article: %v", err)
	}

	result := StructToMap(card)
	result["operation"] = "article_created"
	return result, nil
}
