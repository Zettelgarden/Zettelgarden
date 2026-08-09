package services

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"strconv"
	"strings"
	"time"
)

// StructuredDataIsEmpty reports whether a structured_data payload carries no
// meaningful content: nil, JSON null, an empty object ({}, { }, ...), or an
// empty array. The REST create/update guards use this so that detaching a
// schema — which the editor sends as schema_id: null with structured_data: {} —
// is not mistaken for stray data (bead Zettelgarden-276).
func StructuredDataIsEmpty(raw *json.RawMessage) bool {
	if raw == nil {
		return true
	}
	trimmed := strings.TrimSpace(string(*raw))
	if trimmed == "" || trimmed == "null" {
		return true
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
		return len(obj) == 0
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
		return len(arr) == 0
	}
	return false
}

// ValidateStructuredData validates structured_data against a schema definition and returns cleaned data
func ValidateStructuredData(structuredData json.RawMessage, schema *models.SchemaDefinition) (json.RawMessage, error) {
	// Parse the structured data into a map
	var data map[string]interface{}
	if len(structuredData) > 0 {
		if err := json.Unmarshal(structuredData, &data); err != nil {
			return nil, fmt.Errorf("invalid structured_data JSON: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	// Build a map of field definitions for quick lookup
	fieldMap := make(map[string]models.FieldDefinition)
	for _, field := range schema.Fields {
		fieldMap[field.Name] = field
	}

	// Check all required fields are present AND non-empty (bead Zettelgarden-s2l).
	// A required field whose value is null, an empty/whitespace string, or an
	// empty array is treated as missing: the UI marks the field required, so a
	// "filled" value is the server-side contract.
	for _, field := range schema.Fields {
		if field.Required {
			if _, exists := data[field.Name]; !exists {
				return nil, fmt.Errorf("required field '%s' is missing", field.Name)
			}
			if isEmptyRequiredValue(data[field.Name]) {
				return nil, fmt.Errorf("required field '%s' cannot be empty", field.Name)
			}
		}
	}

	// Validate each field and clean data (remove fields not in schema)
	cleanedData := make(map[string]interface{})
	for fieldName, value := range data {
		fieldDef, exists := fieldMap[fieldName]
		if !exists {
			// Skip fields not defined in schema (remove old/renamed fields)
			continue
		}

		if err := ValidateFieldValue(fieldName, value, fieldDef); err != nil {
			return nil, err
		}
		cleanedData[fieldName] = value
	}

	// Marshal cleaned data back to JSON
	cleanedJSON, err := json.Marshal(cleanedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cleaned data: %w", err)
	}

	return cleanedJSON, nil
}

// isEmptyRequiredValue reports whether a value for a required field counts as
// empty: nil, a string of only whitespace, or an array with no elements.
// Zero numbers and false booleans are NOT empty (they are legitimate values).
func isEmptyRequiredValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	default:
		return false
	}
}

// ValidateCardStructuredData validates structured_data for a card against its
// schema, mirroring the REST create/update semantics so the sync push path
// behaves identically (bead Zettelgarden-s2l). When structuredData is nil or
// empty and the schema has required fields, returns an error. On success
// returns the cleaned data (or nil when there was nothing to clean) plus nil.
func ValidateCardStructuredData(structuredData *json.RawMessage, schema *models.SchemaDefinition) (*json.RawMessage, error) {
	if structuredData != nil && len(*structuredData) > 0 {
		cleaned, err := ValidateStructuredData(*structuredData, schema)
		if err != nil {
			return nil, err
		}
		return &cleaned, nil
	}
	for _, field := range schema.Fields {
		if field.Required {
			return nil, fmt.Errorf("schema requires field '%s' but no structured_data provided", field.Name)
		}
	}
	return nil, nil
}

// ValidationError marks a push change rejected because its payload violates a
// schema contract. The sync push handler maps it to a 400 with the message so
// the client can surface why the batch was refused (mirrors the REST path).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// ValidateFieldValue validates a single field value against its definition
func ValidateFieldValue(fieldName string, value interface{}, fieldDef models.FieldDefinition) error {
	if value == nil {
		if fieldDef.Required {
			return fmt.Errorf("required field '%s' cannot be null", fieldName)
		}
		return nil
	}

	switch fieldDef.Type {
	case "text":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}

	case "number":
		// Accept both float64 and int from JSON
		switch v := value.(type) {
		case float64, int, int64, float32:
			// Valid number types
		case string:
			// Try to parse string as number
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				return fmt.Errorf("field '%s' must be a number", fieldName)
			}
		default:
			return fmt.Errorf("field '%s' must be a number", fieldName)
		}

	case "date":
		// Accept ISO 8601 date string
		dateStr, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a date string", fieldName)
		}
		// Validate ISO 8601 format
		_, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// Also try date-only format
			_, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid ISO 8601 date", fieldName)
			}
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", fieldName)
		}

	case "select":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string for select type", fieldName)
		}
		// Check if value is in options
		validOption := false
		for _, opt := range fieldDef.Options {
			if opt == strVal {
				validOption = true
				break
			}
		}
		if !validOption {
			return fmt.Errorf("field '%s' value '%s' is not in valid options", fieldName, strVal)
		}

	case "multi-select":
		// Expect array of strings
		arrayVal, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("field '%s' must be an array for multi-select type", fieldName)
		}
		for _, item := range arrayVal {
			strItem, ok := item.(string)
			if !ok {
				return fmt.Errorf("field '%s' array items must be strings", fieldName)
			}
			// Check if value is in options
			validOption := false
			for _, opt := range fieldDef.Options {
				if opt == strItem {
					validOption = true
					break
				}
			}
			if !validOption {
				return fmt.Errorf("field '%s' value '%s' is not in valid options", fieldName, strItem)
			}
		}

	case "link_to_card":
		// Expect integer card ID
		// Note: We don't validate the card exists here to avoid circular dependencies
		// This will be validated at the handler level where we have DB access
		switch value.(type) {
		case float64, int, int64, float32:
			// Valid number types
		case string:
			// Try to parse string as int
			_, err := strconv.Atoi(value.(string))
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid card ID (integer)", fieldName)
			}
		default:
			return fmt.Errorf("field '%s' must be a valid card ID (integer)", fieldName)
		}

	default:
		return fmt.Errorf("unknown field type '%s' for field '%s'", fieldDef.Type, fieldName)
	}

	return nil
}

// ValidateLinkToCardFields validates that link_to_card fields reference valid cards
func ValidateLinkToCardFields(db models.Database, userID int, structuredData json.RawMessage, schema *models.SchemaDefinition) error {
	// Parse the structured data
	var data map[string]interface{}
	if err := json.Unmarshal(structuredData, &data); err != nil {
		return err
	}

	// Check each field
	for _, field := range schema.Fields {
		if field.Type == "link_to_card" {
			value, hasValue := data[field.Name]
			if !hasValue || value == nil {
				continue
			}

			// Extract card ID
			var cardID int
			switch v := value.(type) {
			case float64:
				cardID = int(v)
			case int:
				cardID = v
			case string:
				parsedID, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("field '%s': invalid card ID format", field.Name)
				}
				cardID = parsedID
			default:
				return fmt.Errorf("field '%s': invalid card ID type", field.Name)
			}

			// Validate card exists and belongs to user
			var cardExists bool
			err := db.QueryRow(`
				SELECT EXISTS(SELECT 1 FROM cards WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE)
			`, cardID, userID).Scan(&cardExists)
			if err != nil {
				return fmt.Errorf("field '%s': failed to validate card reference", field.Name)
			}
			if !cardExists {
				return fmt.Errorf("field '%s': referenced card (ID %d) does not exist", field.Name, cardID)
			}
		}
	}

	return nil
}
