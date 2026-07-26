// Package backlink provides shared backlink extraction and management functions.
// Both the services package and the services/tools/card package use these
// to avoid duplicating backlink logic.
package backlink

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// IsMarkdownLink checks if a bracket match is part of a markdown link [text](url).
func IsMarkdownLink(text, match string) bool {
	pos := strings.Index(text, match)
	if pos == -1 {
		return false
	}
	if pos+len(match) < len(text) && text[pos+len(match)] == '(' {
		return true
	}
	return false
}

// ExtractBacklinks extracts card references from markdown text.
// Supports both wiki-link syntax [[card_id]] and [[card_id|display text]],
// as well as the legacy [card_id] syntax (excluding markdown links).
// Both syntaxes work simultaneously for backwards compatibility.
func ExtractBacklinks(text string) []string {
	var backlinks []string

	// 1. Extract wiki-links: [[card_id]] or [[card_id|display text]]
	wikiRe := regexp.MustCompile(`\[\[([^\]|]+?)(?:\|[^\]]*)?\]\]`)
	for _, match := range wikiRe.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			backlinks = append(backlinks, strings.TrimSpace(match[1]))
		}
	}

	// 2. Extract legacy-style: [card_id] excluding markdown links and wiki-links
	oldRe := regexp.MustCompile(`\[([^\]]+)\]`)
	for _, match := range oldRe.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			captured := match[1]
			// Skip wiki-links: old regex on [[x]] captures "[x" with leading bracket
			if strings.HasPrefix(captured, "[") {
				continue
			}
			if !IsMarkdownLink(text, match[0]) {
				backlinks = append(backlinks, captured)
			}
		}
	}

	return backlinks
}

// GetSchemaByID fetches a schema definition by ID. Returns nil if not found or on error.
func GetSchemaByID(db models.Database, userID int, schemaID int) *models.SchemaDefinition {
	query := `SELECT id, name, slug, owner_id, fields, created_at, updated_at, is_deleted FROM schema_definitions WHERE id = $1 AND owner_id = $2 AND is_deleted = FALSE`
	schema, err := models.ScanSchemaDefinition(db.QueryRow(query, schemaID, userID))
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error fetching schema %d: %v", schemaID, err)
		}
		return nil
	}
	return schema
}

// ExtractBacklinksFromStructuredData extracts card IDs (as human-readable card_id strings) from structured_data JSONB.
// It only extracts values from fields that are defined as link_to_card type in the schema.
// If schema is nil, returns empty slice since we cannot determine which fields are links.
func ExtractBacklinksFromStructuredData(db models.Database, userID int, structuredData *json.RawMessage, schema *models.SchemaDefinition) []string {
	if structuredData == nil || len(*structuredData) == 0 || schema == nil {
		return []string{}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(*structuredData, &data); err != nil {
		log.Printf("Error unmarshaling structured data for backlink extraction: %v", err)
		return []string{}
	}

	linkFields := make(map[string]bool)
	for _, field := range schema.Fields {
		if field.Type == "link_to_card" {
			linkFields[field.Name] = true
		}
	}

	if len(linkFields) == 0 {
		return []string{}
	}

	var internalIDs []int
	for fieldName, value := range data {
		if !linkFields[fieldName] {
			continue
		}

		if value == nil {
			continue
		}

		var internalID int
		switch v := value.(type) {
		case float64:
			internalID = int(v)
		case int:
			internalID = v
		case int64:
			internalID = int(v)
		case string:
			if parsedID, err := strconv.Atoi(v); err == nil {
				internalID = parsedID
			} else {
				continue
			}
		default:
			continue
		}

		internalIDs = append(internalIDs, internalID)
	}

	if len(internalIDs) == 0 {
		return []string{}
	}

	args := append([]any{userID}, models.IntArgs(internalIDs)...)
	rows, err := db.Query(`
		SELECT card_id FROM cards
		WHERE user_id = $1 AND id IN `+models.InList(2, len(internalIDs))+` AND is_deleted = FALSE
	`, args...)
	if err != nil {
		log.Printf("Error batch looking up card_ids: %v", err)
		return []string{}
	}
	defer rows.Close()

	var backlinks []string
	for rows.Next() {
		var cardID string
		if err := rows.Scan(&cardID); err != nil {
			log.Printf("Error scanning card_id: %v", err)
			continue
		}
		backlinks = append(backlinks, cardID)
	}

	return backlinks
}

// UpdateBacklinks updates the backlinks table for a card.
func UpdateBacklinks(db models.Database, cardPK int, backlinks []string) error {
	_, err := db.Exec("DELETE FROM backlinks WHERE source_id_int = $1", cardPK)
	if err != nil {
		log.Printf("UpdateBacklinks: failed to delete backlinks: %v", err)
		return err
	}
	for _, targetID := range backlinks {
		_, err = db.Exec(`
	WITH target_id AS (
    SELECT id
    FROM cards
    WHERE card_id = $2
)
INSERT INTO backlinks (source_id_int, target_id_int, created_at, updated_at)
SELECT $1, target_id.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM target_id;
		`,
			cardPK, targetID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
