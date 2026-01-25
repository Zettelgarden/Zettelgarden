package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// FieldDefinition represents a single field definition in a schema
type FieldDefinition struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`     // text, number, date, boolean, select, multi-select, link_to_card
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"` // For select/multi-select types
}

// SchemaDefinition represents a schema definition for structured card data
type SchemaDefinition struct {
	ID        int                      `json:"id"`
	Name      string                   `json:"name"`
	Slug      string                   `json:"slug"`
	OwnerID   int                      `json:"owner_id"`
	Fields    []FieldDefinition        `json:"fields"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	IsDeleted bool                     `json:"is_deleted"`
}

// ScanSchemaDefinition scans a single SchemaDefinition from a sql.Row
func ScanSchemaDefinition(row *sql.Row) (*SchemaDefinition, error) {
	var schema SchemaDefinition
	var fieldsJSON []byte

	err := row.Scan(
		&schema.ID,
		&schema.Name,
		&schema.Slug,
		&schema.OwnerID,
		&fieldsJSON,
		&schema.CreatedAt,
		&schema.UpdatedAt,
		&schema.IsDeleted,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("Error scanning schema definition: %v", err)
		return nil, err
	}

	// Unmarshal the JSONB fields
	if len(fieldsJSON) > 0 {
		if err := json.Unmarshal(fieldsJSON, &schema.Fields); err != nil {
			log.Printf("Error unmarshaling fields JSON: %v", err)
			return nil, err
		}
	}

	return &schema, nil
}

// ScanSchemaDefinitions scans multiple SchemaDefinitions from sql.Rows
func ScanSchemaDefinitions(rows *sql.Rows) ([]SchemaDefinition, error) {
	var schemas []SchemaDefinition

	defer rows.Close()

	for rows.Next() {
		var schema SchemaDefinition
		var fieldsJSON []byte

		if err := rows.Scan(
			&schema.ID,
			&schema.Name,
			&schema.Slug,
			&schema.OwnerID,
			&fieldsJSON,
			&schema.CreatedAt,
			&schema.UpdatedAt,
			&schema.IsDeleted,
		); err != nil {
			log.Printf("Error scanning schema definition: %v", err)
			return schemas, err
		}

		// Unmarshal the JSONB fields
		if len(fieldsJSON) > 0 {
			if err := json.Unmarshal(fieldsJSON, &schema.Fields); err != nil {
				log.Printf("Error unmarshaling fields JSON: %v", err)
				return schemas, err
			}
		}

		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating schema definitions: %v", err)
		return schemas, err
	}

	return schemas, nil
}

// FieldsToJSONB converts a slice of FieldDefinition to JSONB for database storage
func FieldsToJSONB(fields []FieldDefinition) ([]byte, error) {
	if fields == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(fields)
}

// CreateSchemaDefinitionParams represents the parameters needed to create a schema
type CreateSchemaDefinitionParams struct {
	Name    string           `json:"name"`
	OwnerID int              `json:"owner_id"`
	Fields  []FieldDefinition `json:"fields"`
}

// UpdateSchemaDefinitionParams represents the parameters needed to update a schema
type UpdateSchemaDefinitionParams struct {
	ID      int              `json:"id"`
	Name    string           `json:"name"`
	Fields  []FieldDefinition `json:"fields"`
}

// GenerateSlug generates a URL-safe slug from a schema name
// Example: "Book Review" -> "book-review"
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Trim whitespace
	slug = strings.TrimSpace(slug)

	// Replace any non-alphanumeric characters (except hyphens) with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Ensure slug is not empty
	if slug == "" {
		slug = "schema"
	}

	return slug
}

// GetUniqueSlug generates a unique slug for a schema by checking for duplicates
// and appending a number if necessary (e.g., "book-review-2")
func GetUniqueSlug(db *sql.DB, ownerID int, baseSlug string) (string, error) {
	// First, try the base slug
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM schema_definitions
		WHERE owner_id = $1 AND slug = $2 AND is_deleted = false
	`, ownerID, baseSlug).Scan(&count)
	if err != nil {
		return "", err
	}

	if count == 0 {
		return baseSlug, nil
	}

	// If base slug exists, find the next available number
	var maxNum int
	err = db.QueryRow(`
		SELECT COALESCE(MAX(CAST(NULLIF(regexp_replace(slug, '^' || $2 || '-?', '') AS INT)), 0)
		FROM schema_definitions
		WHERE owner_id = $1 AND slug ~ $2
	`, ownerID, baseSlug).Scan(&maxNum)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%d", baseSlug, maxNum+1), nil
}
