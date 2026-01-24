package models

import (
	"database/sql"
	"encoding/json"
	"log"
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
