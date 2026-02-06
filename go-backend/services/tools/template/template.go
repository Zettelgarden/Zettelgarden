// Package template provides template-related data access and business logic
// for the Zettelgarden tool registry.
//
// This package contains functions for managing card templates, including
// CRUD operations and child ID generation for hierarchical card structures.
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
		return models.CardTemplate{}, fmt.Errorf("template not found: %w", err)
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
// (e.g., '1a2.3'). This is useful for creating structured card hierarchies.
func GetNextChildCardID(db *sql.DB, userID int, parentID int) (string, error) {
	// 1. Get parent card's card_id (human readable ID)
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
	if err != nil {
		log.Printf("Error finding parent card ID for parentID %d: %v", parentID, err)
		return "", fmt.Errorf("parent card not found: %w", err)
	}

	// 2. Get all existing children
	children, err := getChildCards(db, userID, parentID)
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

// getChildCards retrieves child cards for a given parent card.
// This is a local copy of the function from services/cards.go to avoid circular imports.
func getChildCards(db *sql.DB, userID int, cardID int) ([]models.PartialCard, error) {
	// Find child cards based on card_id hierarchy
	query := `
		SELECT id, card_id, user_id, title, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND parent_id = $2 and id != $3 AND is_deleted = FALSE
		ORDER BY card_id
	`

	rows, err := db.Query(query, userID, cardID, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards, err := models.ScanPartialCards(rows)
	if err != nil {
		return nil, err
	}

	return cards, nil
}
