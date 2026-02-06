// Package template provides template-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The template domain contains tools for managing card templates and child ID generation.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// The template domain package demonstrates the pattern for splitting tools into
// separate domain packages. The registration is handled in services/template_tools.go
// to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions (GetTemplate, GetTemplates, GetNextChildCardID)
// 2. Domain-specific business logic
// 3. Template and card hierarchy management
package template

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"go-backend/models"
)

// GetTemplate retrieves a template by ID for a specific user.
// This is the domain data access function for template operations.
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

// GetTemplates retrieves all templates for a specific user.
// This is the domain data access function for template listing.
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

// PartialCard represents a minimal card structure for child card operations.
type PartialCard struct {
	ID       int
	CardID   string
	UserID   int
	Title    string
	ParentID *int
}

// GetChildCards retrieves child cards for a given parent card.
// This is a domain data access function for card hierarchy operations.
func GetChildCards(db *sql.DB, userID int, cardID int) ([]PartialCard, error) {
	query := `
		SELECT id, card_id, user_id, title, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND parent_id = $2 AND id != $3 AND is_deleted = FALSE
		ORDER BY card_id
	`

	rows, err := db.Query(query, userID, cardID, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []PartialCard
	for rows.Next() {
		var child PartialCard
		if err := rows.Scan(
			&child.ID,
			&child.CardID,
			&child.UserID,
			&child.Title,
			&child.ParentID,
			new(interface{}), // created_at - unused
			new(interface{}), // updated_at - unused
		); err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	return children, nil
}

// GetNextChildCardID returns the next available child card ID for a parent card.
// This is the domain business logic for card ID generation.
func GetNextChildCardID(db *sql.DB, userID int, parentID int) (string, error) {
	// 1. Get parent card's card_id (human readable ID)
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
	if err != nil {
		log.Printf("Error finding parent card ID for parentID %d: %v", parentID, err)
		return "", fmt.Errorf("parent card not found")
	}

	// 2. Get all existing children
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
		re := regexp.MustCompile(`^[.\-/_]+(\d+)`)
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
