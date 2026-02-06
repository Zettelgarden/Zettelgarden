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
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// RegisterTemplateTools registers all template-related tools
func (tr *ToolRegistry) RegisterTemplateTools() {
	RegisterTool(tr, "get_template",
		"Get a specific template by its numeric ID. Returns the full template details including name, title, and body templates.",
		handleGetTemplate,
		ToolParam{Name: "template_id", Type: "integer", Required: true, Desc: "The numeric ID of the template to retrieve"},
	)

	RegisterTool(tr, "list_templates",
		"Get all templates for the current user. Templates are reusable card structures with variable substitution.",
		handleListTemplates,
	)

	RegisterTool(tr, "get_next_child_id",
		"Get the next available child card ID for a parent card (e.g., '1a2.3'). This is useful for creating structured card hierarchies.",
		handleGetNextChildID,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the parent card"},
	)
}

// Template tool handlers

func handleGetTemplate(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templateID, err := getIntParam(args, "template_id")
	if err != nil {
		return nil, err
	}

	template, err := GetTemplate(ctx.DB, ctx.UserID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %v", err)
	}

	return StructToMap(template), nil
}

func handleListTemplates(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templates, err := GetTemplates(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %v", err)
	}

	var results []map[string]interface{}
	for _, template := range templates {
		results = append(results, StructToMap(template))
	}

	return map[string]interface{}{
		"templates": results,
		"total":     len(templates),
	}, nil
}

func handleGetNextChildID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	nextID, err := GetNextChildCardID(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get next child ID: %v", err)
	}

	if nextID == "" {
		return map[string]interface{}{
			"error":   true,
			"message": "Parent card not found or error occurred",
			"new_id":  "",
		}, nil
	}

	return map[string]interface{}{
		"error":   false,
		"message": "",
		"new_id":  nextID,
	}, nil
}

// Template service functions

// GetTemplate retrieves a template by ID for a specific user
func GetTemplate(db *sql.DB, userID, templateID int) (models.CardTemplate, error) {
	var template models.CardTemplate

	query := `
		SELECT id, user_id, name, title, body, created_at, updated_at
		FROM card_templates
		WHERE id = $1 AND user_id = $2
	`

	err := db.QueryRow(query, templateID, userID).Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return models.CardTemplate{}, fmt.Errorf("template not found")
	}

	return template, nil
}

// GetTemplates retrieves all templates for a specific user
func GetTemplates(db *sql.DB, userID int) ([]models.CardTemplate, error) {
	query := `
		SELECT id, user_id, name, title, body, created_at, updated_at
		FROM card_templates
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.CardTemplate
	for rows.Next() {
		var template models.CardTemplate
		if err := rows.Scan(
			&template.ID,
			&template.UserID,
			&template.Name,
			&template.Title,
			&template.Body,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// GetNextChildCardID returns the next available child card ID for a parent card
func GetNextChildCardID(db *sql.DB, userID int, parentID int) (string, error) {
	// 1. Get parent card's card_id (human readable ID)
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
	if err != nil {
		log.Printf("Error finding parent card ID for parentID %d: %v", parentID, err)
		return "", fmt.Errorf("parent card not found")
	}

	// 2. Get all existing children using service
	children, err := GetChildCards(db, userID, parentID)
	if err != nil {
		log.Printf("Error getting child cards for parentID %d: %v", parentID, err)
		return parentCardID + ".1", nil // Default to .1 if there's an error
	}

	// 3. Extract numeric suffixes from children's card_ids
	childNumbers := make([]int, 0)
	parentIDLength := len(parentCardID)

	for _, child := range children {
		childID := child.CardID

		// Verify this is actually a direct child by checking it starts with parent ID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}

		// Get the part after the parent ID
		suffix := childID[parentIDLength:]

		// Extract the first number after any separator using regex
		re := regexp.MustCompile(`^[.\\/-]+(\d+)`)
		match := re.FindStringSubmatch(suffix)
		if len(match) == 2 {
			num, err := strconv.Atoi(match[1])
			if err == nil {
				childNumbers = append(childNumbers, num)
			}
		}
	}

	// 4. Find the highest number and increment
	if len(childNumbers) == 0 {
		return parentCardID + ".1", nil // No existing children, start with 1
	}

	maxNumber := 0
	for _, num := range childNumbers {
		if num > maxNumber {
			maxNumber = num
		}
	}

	nextNumber := maxNumber + 1
	return fmt.Sprintf("%s.%d", parentCardID, nextNumber), nil
}
