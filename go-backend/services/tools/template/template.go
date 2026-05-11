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
//
// It handles two naming schemes:
//  1. Children whose card_id starts with the parent's card_id (e.g., parent "999",
//     children "999.1", "999.2")
//  2. Children whose card_id uses a different prefix than the parent's card_id
//     (e.g., parent card_id is "SP24/B.8" but children use "2483.1", "2483.82").
//     In this case, the function detects the common prefix among existing children
//     and increments the trailing number.
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

	if len(children) == 0 {
		return parentCardID + ".1", nil // No existing children, start with .1
	}

	// 3. Try to match children using parent's card_id as prefix
	childNumbers := make([]int, 0)
	parentIDLength := len(parentCardID)

	for _, child := range children {
		childID := child.CardID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}
		suffix := childID[parentIDLength:]
		re := regexp.MustCompile(`^[.\\/-]+(\d+)`)
		match := re.FindStringSubmatch(suffix)
		if len(match) == 2 {
			num, err := strconv.Atoi(match[1])
			if err == nil {
				childNumbers = append(childNumbers, num)
			}
		}
	}

	// If children matched the parent's card_id prefix, use the standard approach
	if len(childNumbers) > 0 {
		maxNumber := 0
		for _, num := range childNumbers {
			if num > maxNumber {
				maxNumber = num
			}
		}
		return fmt.Sprintf("%s.%d", parentCardID, maxNumber+1), nil
	}

	// 4. Children don't use the parent's card_id as prefix.
	//    Detect the common prefix scheme among existing children and increment.
	//    We look for a pattern like "PREFIX.NUMBER" among children and find
	//    the common prefix, then increment the max trailing number.
	type prefixEntry struct {
		prefix string
		sep    string // separator character used ('.', '/', or '-')
		num    int
	}

	re := regexp.MustCompile(`^(.+?)([./\-])(\d+)$`)
	var entries []prefixEntry

	for _, child := range children {
		match := re.FindStringSubmatch(child.CardID)
		if len(match) == 4 {
			num, err := strconv.Atoi(match[3])
			if err == nil {
				entries = append(entries, prefixEntry{
					prefix: match[1],
					sep:    match[2],
					num:    num,
				})
			}
		}
	}

	if len(entries) == 0 {
		// No children matched any PREFIX.SEP.NUMBER pattern; fall back to parent-based default
		return parentCardID + ".1", nil
	}

	// Find the most common prefix among entries (the dominant naming scheme)
	prefixCounts := make(map[string]int)
	for _, e := range entries {
		prefixCounts[e.prefix]++
	}
	var bestPrefix string
	bestCount := 0
	for p, c := range prefixCounts {
		if c > bestCount {
			bestCount = c
			bestPrefix = p
		}
	}

	// Use the separator from the first entry with the best prefix
	var bestSep string
	for _, e := range entries {
		if e.prefix == bestPrefix {
			bestSep = e.sep
			break
		}
	}

	// Find the max number among entries using the best prefix
	maxNumber := 0
	for _, e := range entries {
		if e.prefix == bestPrefix && e.num > maxNumber {
			maxNumber = e.num
		}
	}

	return fmt.Sprintf("%s%s%d", bestPrefix, bestSep, maxNumber+1), nil
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
