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

// GetNextChildCardID returns the next available child card ID for a parent card.
//
// It analyses the existing children's card_ids to detect the naming scheme
// and returns the next ID in the same scheme. It handles:
//
//   - Numeric suffixes: parent "999", children "999.1", "999.2" → "999.3"
//   - Alphabetic suffixes: parent "SP24/B.8", children "SP24/B.8/A", "SP24/B.8/B" → "SP24/B.8/C"
//   - Mismatched prefixes: parent card_id "SP24/B.8" but children use "2483.1",
//     "2483.82" → detects dominant prefix and increments to "2483.83"
//   - No children: returns parentCardID + ".1"
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
		return parentCardID + ".1", nil
	}

	if len(children) == 0 {
		return parentCardID + ".1", nil
	}

	// 3. Classify children by their suffix type relative to the parent's card_id.
	//    We look for children that start with the parent's card_id followed by
	//    a separator and a suffix (numeric or single uppercase letter).
	parentIDLength := len(parentCardID)

	type suffixEntry struct {
		sep      string // separator character ('.', '/', or '-')
		suffix   string // the suffix part ("1", "82", "A", "B")
		isAlpha  bool   // true if suffix is a single uppercase letter
	}

	var matchedEntries []suffixEntry

	// Regex for numeric suffix: separator + digits
	numRe := regexp.MustCompile(`^[.\\/-]+(\d+)$`)
	// Regex for alphabetic suffix: separator + single uppercase letter
	alphaRe := regexp.MustCompile(`^[.\\/-]+([A-Z])$`)

	for _, child := range children {
		childID := child.CardID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}
		suffix := childID[parentIDLength:]

		if match := numRe.FindStringSubmatch(suffix); len(match) == 2 {
			matchedEntries = append(matchedEntries, suffixEntry{sep: string(suffix[0]), suffix: match[1], isAlpha: false})
		} else if match := alphaRe.FindStringSubmatch(suffix); len(match) == 2 {
			matchedEntries = append(matchedEntries, suffixEntry{sep: string(suffix[0]), suffix: match[1], isAlpha: true})
		}
	}

	// 4. If children matched the parent's card_id prefix, determine next suffix
	if len(matchedEntries) > 0 {
		// Determine the dominant separator and suffix type
		type schemeKey struct {
			sep     string
			isAlpha bool
		}
		schemeCounts := make(map[schemeKey]int)
		for _, e := range matchedEntries {
			schemeCounts[schemeKey{e.sep, e.isAlpha}]++
		}
		var bestScheme schemeKey
		bestCount := 0
		for s, c := range schemeCounts {
			if c > bestCount {
				bestCount = c
				bestScheme = s
			}
		}

		if bestScheme.isAlpha {
			// Alphabetic suffix: find the highest letter and increment
			maxLetter := byte('A')
			for _, e := range matchedEntries {
				if e.isAlpha && e.sep == bestScheme.sep && e.suffix[0] > maxLetter {
					maxLetter = e.suffix[0]
				}
			}
			nextLetter := string(rune(maxLetter) + 1)
			return parentCardID + bestScheme.sep + nextLetter, nil
		}

		// Numeric suffix: find the highest number and increment
		maxNumber := 0
		for _, e := range matchedEntries {
			if !e.isAlpha && e.sep == bestScheme.sep {
				num, _ := strconv.Atoi(e.suffix)
				if num > maxNumber {
					maxNumber = num
				}
			}
		}
		return fmt.Sprintf("%s%s%d", parentCardID, bestScheme.sep, maxNumber+1), nil
	}

	// 5. Children don't use the parent's card_id as prefix.
	//    Detect the common prefix scheme among existing children and increment.
	type prefixEntry struct {
		prefix string
		sep    string
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
		return parentCardID + ".1", nil
	}

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

	var bestSep string
	for _, e := range entries {
		if e.prefix == bestPrefix {
			bestSep = e.sep
			break
		}
	}

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
