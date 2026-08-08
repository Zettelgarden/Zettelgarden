package services

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/tests"
	"strings"
	"testing"
)

// TestValidateStructuredData_ValidDataGetsCleaned tests that valid structured data passes validation and is returned cleaned
func TestValidateStructuredData_ValidDataGetsCleaned(t *testing.T) {
	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "rating", Type: "number", Required: false},
			{Name: "created_at", Type: "date", Required: false},
		},
	}

	inputData := json.RawMessage(`{"title":"Test Card","rating":4.5,"created_at":"2024-01-15T10:30:00Z","extra_field":"should be removed"}`)

	result, err := ValidateStructuredData(inputData, schema)
	if err != nil {
		t.Fatalf("Expected no error for valid data, got: %v", err)
	}

	// Parse result to verify content
	var resultData map[string]interface{}
	if err := json.Unmarshal(result, &resultData); err != nil {
		t.Fatalf("Failed to parse result JSON: %v", err)
	}

	// Verify all expected fields are present
	if resultData["title"] != "Test Card" {
		t.Errorf("Expected title 'Test Card', got %v", resultData["title"])
	}
	if resultData["rating"] != 4.5 {
		t.Errorf("Expected rating 4.5, got %v", resultData["rating"])
	}
	if resultData["created_at"] != "2024-01-15T10:30:00Z" {
		t.Errorf("Expected created_at '2024-01-15T10:30:00Z', got %v", resultData["created_at"])
	}

	// Verify extra field was removed
	if _, exists := resultData["extra_field"]; exists {
		t.Error("Extra field should have been removed from cleaned data")
	}
}

// TestValidateStructuredData_RequiredFieldMissing tests that missing required fields return an error
func TestValidateStructuredData_RequiredFieldMissing(t *testing.T) {
	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "author", Type: "text", Required: true},
			{Name: "rating", Type: "number", Required: false},
		},
	}

	inputData := json.RawMessage(`{"title":"Test Card","rating":4.5}`)

	_, err := ValidateStructuredData(inputData, schema)
	if err == nil {
		t.Fatal("Expected error for missing required field, got nil")
	}

	expectedError := "required field 'author' is missing"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateStructuredData_RequiredFieldEmpty tests that required fields
// present but empty (empty/whitespace string, null, empty array) are rejected
// (bead Zettelgarden-s2l).
func TestValidateStructuredData_RequiredFieldEmpty(t *testing.T) {
	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "tags", Type: "multi-select", Required: true, Options: []string{"a", "b"}},
			{Name: "rating", Type: "number", Required: true},
			{Name: "active", Type: "boolean", Required: true},
		},
	}

	cases := []struct {
		name  string
		input string
	}{
		{"empty string", `{"title":"","tags":["a"],"rating":1,"active":true}`},
		{"whitespace only", `{"title":"   ","tags":["a"],"rating":1,"active":true}`},
		{"null value", `{"title":null,"tags":["a"],"rating":1,"active":true}`},
		{"empty array multi-select", `{"title":"x","tags":[],"rating":1,"active":true}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateStructuredData(json.RawMessage(tc.input), schema)
			if err == nil {
				t.Fatal("Expected error for empty required field, got nil")
			}
			if !strings.Contains(err.Error(), "required field '") {
				t.Errorf("Expected message naming the field, got: %v", err.Error())
			}
		})
	}

	// Zero numbers and false booleans are legitimate values, not empty.
	valid := `{"title":"x","tags":["a"],"rating":0,"active":false}`
	if _, err := ValidateStructuredData(json.RawMessage(valid), schema); err != nil {
		t.Errorf("Zero/false values should be accepted, got: %v", err)
	}
}

// TestValidateStructuredData_InvalidFieldType tests that invalid field types return an error
func TestValidateStructuredData_InvalidFieldType(t *testing.T) {
	testCases := []struct {
		name          string
		fieldType     string
		inputData     string
		expectedError string
	}{
		{
			name:          "text field with number",
			fieldType:     "text",
			inputData:     `{"title":12345}`,
			expectedError: "field 'title' must be a string",
		},
		{
			name:          "number field with text",
			fieldType:     "number",
			inputData:     `{"rating":"not a number"}`,
			expectedError: "field 'rating' must be a number",
		},
		{
			name:          "date field with invalid format",
			fieldType:     "date",
			inputData:     `{"created_at":"not a date"}`,
			expectedError: "field 'created_at' must be a valid ISO 8601 date",
		},
		{
			name:          "boolean field with string",
			fieldType:     "boolean",
			inputData:     `{"is_published":"yes"}`,
			expectedError: "field 'is_published' must be a boolean",
		},
		{
			name:          "select field with invalid option",
			fieldType:     "select",
			inputData:     `{"status":"draft"}`,
			expectedError: "field 'status' value 'draft' is not in valid options",
		},
		{
			name:          "multi-select field with non-array",
			fieldType:     "multi-select",
			inputData:     `{"tags":"single-tag"}`,
			expectedError: "field 'tags' must be an array for multi-select type",
		},
		{
			name:          "link_to_card field with invalid ID",
			fieldType:     "link_to_card",
			inputData:     `{"related_card":"invalid"}`,
			expectedError: "field 'related_card' must be a valid card ID (integer)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var fields []models.FieldDefinition

			switch tc.fieldType {
			case "select":
				fields = []models.FieldDefinition{
					{Name: "status", Type: "select", Required: false, Options: []string{"published", "archived"}},
				}
			case "multi-select":
				fields = []models.FieldDefinition{
					{Name: "tags", Type: "multi-select", Required: false, Options: []string{"tag1", "tag2"}},
				}
			default:
				fields = []models.FieldDefinition{
					{Name: getStatusField(tc.fieldType), Type: tc.fieldType, Required: false},
				}
			}

			schema := &models.SchemaDefinition{
				Fields: fields,
			}

			inputData := json.RawMessage(tc.inputData)

			_, err := ValidateStructuredData(inputData, schema)
			if err == nil {
				t.Fatalf("Expected error for invalid field type, got nil")
			}

			if err.Error() != tc.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tc.expectedError, err.Error())
			}
		})
	}
}

// Helper function to map field type to field name for tests
func getStatusField(fieldType string) string {
	switch fieldType {
	case "text", "number", "date", "boolean", "link_to_card":
		return getTitleForType(fieldType)
	default:
		return "field"
	}
}

func getTitleForType(fieldType string) string {
	switch fieldType {
	case "text":
		return "title"
	case "number":
		return "rating"
	case "date":
		return "created_at"
	case "boolean":
		return "is_published"
	case "link_to_card":
		return "related_card"
	default:
		return "field"
	}
}

// TestValidateStructuredData_FieldsNotInSchemaRemoved tests that fields not in schema are removed
func TestValidateStructuredData_FieldsNotInSchemaRemoved(t *testing.T) {
	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "author", Type: "text", Required: true},
		},
	}

	// Include extra fields that should be removed
	inputData := json.RawMessage(`{
		"title":"Test Card",
		"author":"John Doe",
		"old_field":"should be removed",
		"deprecated_field":"also removed",
		"another_extra":123
	}`)

	result, err := ValidateStructuredData(inputData, schema)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal(result, &resultData); err != nil {
		t.Fatalf("Failed to parse result JSON: %v", err)
	}

	// Verify only schema fields are present
	if len(resultData) != 2 {
		t.Errorf("Expected 2 fields in result, got %d", len(resultData))
	}

	if _, exists := resultData["old_field"]; exists {
		t.Error("Field 'old_field' should have been removed")
	}
	if _, exists := resultData["deprecated_field"]; exists {
		t.Error("Field 'deprecated_field' should have been removed")
	}
	if _, exists := resultData["another_extra"]; exists {
		t.Error("Field 'another_extra' should have been removed")
	}

	// Verify expected fields are still present
	if resultData["title"] != "Test Card" {
		t.Errorf("Expected title 'Test Card', got %v", resultData["title"])
	}
	if resultData["author"] != "John Doe" {
		t.Errorf("Expected author 'John Doe', got %v", resultData["author"])
	}
}

// TestValidateStructuredData_EmptyStructuredDataCreatesEmptyMap tests that empty structured_data creates an empty map
func TestValidateStructuredData_EmptyStructuredDataCreatesEmptyMap(t *testing.T) {
	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: false},
			{Name: "author", Type: "text", Required: false},
		},
	}

	// Empty JSON object
	inputData := json.RawMessage(`{}`)

	result, err := ValidateStructuredData(inputData, schema)
	if err != nil {
		t.Fatalf("Expected no error for empty data, got: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal(result, &resultData); err != nil {
		t.Fatalf("Failed to parse result JSON: %v", err)
	}

	if len(resultData) != 0 {
		t.Errorf("Expected empty map, got %d fields", len(resultData))
	}

	// Empty raw message
	emptyInput := json.RawMessage(``)
	result2, err := ValidateStructuredData(emptyInput, schema)
	if err != nil {
		t.Fatalf("Expected no error for empty raw message, got: %v", err)
	}

	var resultData2 map[string]interface{}
	if err := json.Unmarshal(result2, &resultData2); err != nil {
		t.Fatalf("Failed to parse result JSON: %v", err)
	}

	if len(resultData2) != 0 {
		t.Errorf("Expected empty map for empty raw message, got %d fields", len(resultData2))
	}
}

// TestValidateFieldValue_TextType tests text type validation
func TestValidateFieldValue_TextType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "title",
		Type:     "text",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid string",
			value:     "Test Title",
			wantError: false,
		},
		{
			name:      "empty string is valid",
			value:     "",
			wantError: false,
		},
		{
			name:      "number is invalid",
			value:     123,
			wantError: true,
			errorMsg:  "field 'title' must be a string",
		},
		{
			name:      "boolean is invalid",
			value:     true,
			wantError: true,
			errorMsg:  "field 'title' must be a string",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("title", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_NumberType tests number type validation (including string numbers)
func TestValidateFieldValue_NumberType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "rating",
		Type:     "number",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid float64",
			value:     4.5,
			wantError: false,
		},
		{
			name:      "valid int",
			value:     42,
			wantError: false,
		},
		{
			name:      "valid int64",
			value:     int64(42),
			wantError: false,
		},
		{
			name:      "valid float32",
			value:     float32(3.14),
			wantError: false,
		},
		{
			name:      "valid number as string",
			value:     "123.45",
			wantError: false,
		},
		{
			name:      "integer as string",
			value:     "42",
			wantError: false,
		},
		{
			name:      "negative number as string",
			value:     "-10.5",
			wantError: false,
		},
		{
			name:      "invalid string",
			value:     "not a number",
			wantError: true,
			errorMsg:  "field 'rating' must be a number",
		},
		{
			name:      "boolean is invalid",
			value:     true,
			wantError: true,
			errorMsg:  "field 'rating' must be a number",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("rating", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_DateType tests date type validation (ISO 8601 and date-only formats)
func TestValidateFieldValue_DateType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "created_at",
		Type:     "date",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid ISO 8601 date-time",
			value:     "2024-01-15T10:30:00Z",
			wantError: false,
		},
		{
			name:      "valid ISO 8601 with timezone offset",
			value:     "2024-01-15T10:30:00+08:00",
			wantError: false,
		},
		{
			name:      "valid date-only format",
			value:     "2024-01-15",
			wantError: false,
		},
		{
			name:      "invalid date format",
			value:     "01/15/2024",
			wantError: true,
			errorMsg:  "field 'created_at' must be a valid ISO 8601 date",
		},
		{
			name:      "not a date string",
			value:     "not a date",
			wantError: true,
			errorMsg:  "field 'created_at' must be a valid ISO 8601 date",
		},
		{
			name:      "number is invalid",
			value:     12345,
			wantError: true,
			errorMsg:  "field 'created_at' must be a date string",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("created_at", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_BooleanType tests boolean type validation
func TestValidateFieldValue_BooleanType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "is_published",
		Type:     "boolean",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid true",
			value:     true,
			wantError: false,
		},
		{
			name:      "valid false",
			value:     false,
			wantError: false,
		},
		{
			name:      "string is invalid",
			value:     "true",
			wantError: true,
			errorMsg:  "field 'is_published' must be a boolean",
		},
		{
			name:      "number is invalid",
			value:     1,
			wantError: true,
			errorMsg:  "field 'is_published' must be a boolean",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("is_published", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_SelectType tests select type with valid/invalid options
func TestValidateFieldValue_SelectType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "status",
		Type:     "select",
		Required: false,
		Options:  []string{"draft", "published", "archived"},
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid option - draft",
			value:     "draft",
			wantError: false,
		},
		{
			name:      "valid option - published",
			value:     "published",
			wantError: false,
		},
		{
			name:      "valid option - archived",
			value:     "archived",
			wantError: false,
		},
		{
			name:      "invalid option",
			value:     "deleted",
			wantError: true,
			errorMsg:  "field 'status' value 'deleted' is not in valid options",
		},
		{
			name:      "empty string is invalid",
			value:     "",
			wantError: true,
			errorMsg:  "field 'status' value '' is not in valid options",
		},
		{
			name:      "number is invalid type",
			value:     123,
			wantError: true,
			errorMsg:  "field 'status' must be a string for select type",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("status", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_MultiSelectType tests multi-select type validation
func TestValidateFieldValue_MultiSelectType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "tags",
		Type:     "multi-select",
		Required: false,
		Options:  []string{"tag1", "tag2", "tag3"},
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid array with one option",
			value:     []interface{}{"tag1"},
			wantError: false,
		},
		{
			name:      "valid array with multiple options",
			value:     []interface{}{"tag1", "tag2", "tag3"},
			wantError: false,
		},
		{
			name:      "empty array is valid",
			value:     []interface{}{},
			wantError: false,
		},
		{
			name:      "array with invalid option",
			value:     []interface{}{"tag1", "invalid_tag"},
			wantError: true,
			errorMsg:  "field 'tags' value 'invalid_tag' is not in valid options",
		},
		{
			name:      "array with non-string item",
			value:     []interface{}{"tag1", 123},
			wantError: true,
			errorMsg:  "field 'tags' array items must be strings",
		},
		{
			name:      "string instead of array",
			value:     "tag1",
			wantError: true,
			errorMsg:  "field 'tags' must be an array for multi-select type",
		},
		{
			name:      "number instead of array",
			value:     123,
			wantError: true,
			errorMsg:  "field 'tags' must be an array for multi-select type",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("tags", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_LinkToCardType tests link_to_card type validation
func TestValidateFieldValue_LinkToCardType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "related_card",
		Type:     "link_to_card",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid float64 card ID",
			value:     float64(123),
			wantError: false,
		},
		{
			name:      "valid int card ID",
			value:     123,
			wantError: false,
		},
		{
			name:      "valid int64 card ID",
			value:     int64(123),
			wantError: false,
		},
		{
			name:      "valid float32 card ID",
			value:     float32(123),
			wantError: false,
		},
		{
			name:      "valid string card ID",
			value:     "123",
			wantError: false,
		},
		{
			name:      "invalid string card ID",
			value:     "abc",
			wantError: true,
			errorMsg:  "field 'related_card' must be a valid card ID (integer)",
		},
		{
			name:      "boolean is invalid",
			value:     true,
			wantError: true,
			errorMsg:  "field 'related_card' must be a valid card ID (integer)",
		},
		{
			name:      "null for optional field is valid",
			value:     nil,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("related_card", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_UnknownFieldType tests that unknown field type returns error
func TestValidateFieldValue_UnknownFieldType(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "unknown_field",
		Type:     "unknown_type",
		Required: false,
	}

	err := ValidateFieldValue("unknown_field", "some value", fieldDef)
	if err == nil {
		t.Fatal("Expected error for unknown field type, got nil")
	}

	expectedError := "unknown field type 'unknown_type' for field 'unknown_field'"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateFieldValue_RequiredNullFieldReturnsError tests that required null field returns error
func TestValidateFieldValue_RequiredNullFieldReturnsError(t *testing.T) {
	fieldTypes := []string{"text", "number", "date", "boolean", "select", "multi-select", "link_to_card"}

	for _, fieldType := range fieldTypes {
		t.Run(fieldType, func(t *testing.T) {
			fieldDef := models.FieldDefinition{
				Name:     getTitleForType(fieldType),
				Type:     fieldType,
				Required: true,
				Options:  getOptionsForType(fieldType),
			}

			err := ValidateFieldValue(getTitleForType(fieldType), nil, fieldDef)
			if err == nil {
				t.Fatalf("Expected error for null required field, got nil")
			}

			expectedError := fmt.Sprintf("required field '%s' cannot be null", getTitleForType(fieldType))
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

// TestValidateFieldValue_OptionalNullFieldReturnsNil tests that optional null field returns nil (no error)
func TestValidateFieldValue_OptionalNullFieldReturnsNil(t *testing.T) {
	fieldTypes := []string{"text", "number", "date", "boolean", "select", "multi-select", "link_to_card"}

	for _, fieldType := range fieldTypes {
		t.Run(fieldType, func(t *testing.T) {
			fieldDef := models.FieldDefinition{
				Name:     getTitleForType(fieldType),
				Type:     fieldType,
				Required: false,
				Options:  getOptionsForType(fieldType),
			}

			err := ValidateFieldValue(getTitleForType(fieldType), nil, fieldDef)
			if err != nil {
				t.Fatalf("Expected no error for null optional field, got: %v", err)
			}
		})
	}
}

// Helper function to get options for specific field types
func getOptionsForType(fieldType string) []string {
	switch fieldType {
	case "select", "multi-select":
		return []string{"option1", "option2"}
	default:
		return nil
	}
}

// TestValidateLinkToCardFields_ValidCardReferencePasses tests that valid card reference passes validation
func TestValidateLinkToCardFields_ValidCardReferencePasses(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card to reference
	params := models.EditCardParams{
		Title:  "Referenced Card",
		Body:   "This card will be referenced",
		CardID: "ref_card",
		Link:   "",
	}
	card, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "related_card", Type: "link_to_card", Required: true},
		},
	}

	structuredData := json.RawMessage(fmt.Sprintf(`{"title":"Test","related_card":%d}`, card.ID))

	err = ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err != nil {
		t.Fatalf("Expected no error for valid card reference, got: %v", err)
	}
}

// TestValidateLinkToCardFields_InvalidCardIDReturnsError tests that invalid card ID returns error
func TestValidateLinkToCardFields_InvalidCardIDReturnsError(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	nonExistentCardID := 99999

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "related_card", Type: "link_to_card", Required: true},
		},
	}

	structuredData := json.RawMessage(fmt.Sprintf(`{"related_card":%d}`, nonExistentCardID))

	err := ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err == nil {
		t.Fatal("Expected error for non-existent card, got nil")
	}

	expectedError := fmt.Sprintf("field 'related_card': referenced card (ID %d) does not exist", nonExistentCardID)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateLinkToCardFields_NonExistentCardReturnsError tests that non-existent card returns error
func TestValidateLinkToCardFields_NonExistentCardReturnsError(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "related_card", Type: "link_to_card", Required: true},
		},
	}

	// Use a card ID that doesn't exist
	structuredData := json.RawMessage(`{"related_card":99999}`)

	err := ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err == nil {
		t.Fatal("Expected error for non-existent card, got nil")
	}

	expectedError := "field 'related_card': referenced card (ID 99999) does not exist"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateLinkToCardFields_NonLinkToCardFieldsIgnored tests that non-link_to_card fields are ignored
func TestValidateLinkToCardFields_NonLinkToCardFieldsIgnored(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "rating", Type: "number", Required: false},
			{Name: "created_at", Type: "date", Required: false},
			{Name: "is_published", Type: "boolean", Required: false},
			{Name: "status", Type: "select", Required: false, Options: []string{"draft", "published"}},
			{Name: "tags", Type: "multi-select", Required: false, Options: []string{"tag1", "tag2"}},
		},
	}

	structuredData := json.RawMessage(`{
		"title":"Test Card",
		"rating":4.5,
		"created_at":"2024-01-15T10:30:00Z",
		"is_published":true,
		"status":"published",
		"tags":["tag1"]
	}`)

	err := ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err != nil {
		t.Fatalf("Expected no error for schema without link_to_card fields, got: %v", err)
	}
}

// TestValidateLinkToCardFields_MultipleLinkToCardFields tests validation with multiple link_to_card fields
func TestValidateLinkToCardFields_MultipleLinkToCardFields(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create two cards to reference
	card1, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Card 1",
		Body:   "First card",
		CardID: "card1",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card1: %v", err)
	}

	card2, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Card 2",
		Body:   "Second card",
		CardID: "card2",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card2: %v", err)
	}

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "related_card", Type: "link_to_card", Required: true},
			{Name: "another_card", Type: "link_to_card", Required: true},
		},
	}

	structuredData := json.RawMessage(fmt.Sprintf(`{"related_card":%d,"another_card":%d}`, card1.ID, card2.ID))

	err = ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err != nil {
		t.Fatalf("Expected no error for valid multiple card references, got: %v", err)
	}
}

// TestValidateLinkToCardFields_NullLinkToCardField tests that null link_to_card fields are skipped
func TestValidateLinkToCardFields_NullLinkToCardField(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "title", Type: "text", Required: true},
			{Name: "related_card", Type: "link_to_card", Required: false},
		},
	}

	// Test with null value
	structuredData := json.RawMessage(`{"title":"Test","related_card":null}`)

	err := ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err != nil {
		t.Fatalf("Expected no error for null optional link_to_card field, got: %v", err)
	}

	// Test without the field at all
	structuredData2 := json.RawMessage(`{"title":"Test"}`)

	err = ValidateLinkToCardFields(s.DB, userID, structuredData2, schema)
	if err != nil {
		t.Fatalf("Expected no error when link_to_card field is missing, got: %v", err)
	}
}

// TestValidateLinkToCardFields_CardOwnershipValidation tests that card ownership is validated
func TestValidateLinkToCardFields_CardOwnershipValidation(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID1 := 1
	userID2 := 2

	// Create a card for user 1
	card1, err := CreateCard(s.DB, userID1, models.EditCardParams{
		Title:  "User 1 Card",
		Body:   "This card belongs to user 1",
		CardID: "ownership_user1_card",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card for user 1: %v", err)
	}

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "related_card", Type: "link_to_card", Required: true},
		},
	}

	// User 2 should not be able to reference user 1's card
	structuredData := json.RawMessage(fmt.Sprintf(`{"related_card":%d}`, card1.ID))

	err = ValidateLinkToCardFields(s.DB, userID2, structuredData, schema)
	if err == nil {
		t.Fatal("Expected error when user 2 references user 1's card, got nil")
	}

	expectedError := fmt.Sprintf("field 'related_card': referenced card (ID %d) does not exist", card1.ID)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateLinkToCardFields_DeletedCardValidation tests that deleted cards are not valid references
func TestValidateLinkToCardFields_DeletedCardValidation(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Create a card
	card, err := CreateCard(s.DB, userID, models.EditCardParams{
		Title:  "Card to Delete",
		Body:   "This card will be deleted",
		CardID: "to_delete",
		Link:   "",
	})
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}

	// Delete the card
	err = DeleteCard(s.DB, userID, card.ID)
	if err != nil {
		t.Fatalf("Failed to delete card: %v", err)
	}

	schema := &models.SchemaDefinition{
		Fields: []models.FieldDefinition{
			{Name: "related_card", Type: "link_to_card", Required: true},
		},
	}

	// Try to reference the deleted card
	structuredData := json.RawMessage(fmt.Sprintf(`{"related_card":%d}`, card.ID))

	err = ValidateLinkToCardFields(s.DB, userID, structuredData, schema)
	if err == nil {
		t.Fatal("Expected error when referencing deleted card, got nil")
	}

	expectedError := fmt.Sprintf("field 'related_card': referenced card (ID %d) does not exist", card.ID)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestValidateStructuredData_IntegrationTests provides integration tests that combine multiple validations
func TestValidateStructuredData_IntegrationTests(t *testing.T) {
	testCases := []struct {
		name         string
		schema       models.SchemaDefinition
		inputData    string
		wantError    bool
		errorMsg     string
		validateData func(map[string]interface{}) error
	}{
		{
			name: "complex schema with all field types",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "title", Type: "text", Required: true},
					{Name: "author", Type: "text", Required: true},
					{Name: "rating", Type: "number", Required: false},
					{Name: "published_date", Type: "date", Required: true},
					{Name: "is_featured", Type: "boolean", Required: false},
					{Name: "status", Type: "select", Required: true, Options: []string{"draft", "published", "archived"}},
					{Name: "tags", Type: "multi-select", Required: false, Options: []string{"tech", "science", "art"}},
				},
			},
			inputData: `{
				"title":"Test Article",
				"author":"John Doe",
				"rating":4.5,
				"published_date":"2024-01-15T10:30:00Z",
				"is_featured":true,
				"status":"published",
				"tags":["tech","science"]
			}`,
			wantError: false,
		},
		{
			name: "schema with optional fields only - empty data",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "title", Type: "text", Required: false},
					{Name: "description", Type: "text", Required: false},
				},
			},
			inputData: `{}`,
			wantError: false,
			validateData: func(data map[string]interface{}) error {
				if len(data) != 0 {
					return fmt.Errorf("expected empty map, got %d fields", len(data))
				}
				return nil
			},
		},
		{
			name: "removes old fields when schema is updated",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "new_field", Type: "text", Required: true},
				},
			},
			inputData: `{
				"new_field":"value",
				"old_field_1":"should be removed",
				"old_field_2":123
			}`,
			wantError: false,
			validateData: func(data map[string]interface{}) error {
				if len(data) != 1 {
					return fmt.Errorf("expected 1 field, got %d", len(data))
				}
				if _, exists := data["old_field_1"]; exists {
					return fmt.Errorf("old_field_1 should have been removed")
				}
				if _, exists := data["old_field_2"]; exists {
					return fmt.Errorf("old_field_2 should have been removed")
				}
				return nil
			},
		},
		{
			name: "mixed valid and invalid multi-select values",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "title", Type: "text", Required: true},
					{Name: "tags", Type: "multi-select", Required: false, Options: []string{"tag1", "tag2"}},
				},
			},
			inputData: `{
				"title":"Test",
				"tags":["tag1","invalid_tag"]
			}`,
			wantError: true,
			errorMsg:  "field 'tags' value 'invalid_tag' is not in valid options",
		},
		{
			name: "all required fields present",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "field1", Type: "text", Required: true},
					{Name: "field2", Type: "text", Required: true},
					{Name: "field3", Type: "text", Required: true},
				},
			},
			inputData: `{
				"field1":"value1",
				"field2":"value2",
				"field3":"value3"
			}`,
			wantError: false,
		},
		{
			name: "one required field missing",
			schema: models.SchemaDefinition{
				Fields: []models.FieldDefinition{
					{Name: "field1", Type: "text", Required: true},
					{Name: "field2", Type: "text", Required: true},
					{Name: "field3", Type: "text", Required: true},
				},
			},
			inputData: `{
				"field1":"value1",
				"field2":"value2"
			}`,
			wantError: true,
			errorMsg:  "required field 'field3' is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputData := json.RawMessage(tc.inputData)

			result, err := ValidateStructuredData(inputData, &tc.schema)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			var resultData map[string]interface{}
			if err := json.Unmarshal(result, &resultData); err != nil {
				t.Fatalf("Failed to parse result JSON: %v", err)
			}

			if tc.validateData != nil {
				if err := tc.validateData(resultData); err != nil {
					t.Errorf("Data validation failed: %v", err)
				}
			}
		})
	}
}

// TestValidateFieldValue_DateEdgeCases tests edge cases for date validation
func TestValidateFieldValue_DateEdgeCases(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "date_field",
		Type:     "date",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name:      "date with milliseconds",
			value:     "2024-01-15T10:30:00.123Z",
			wantError: false,
		},
		{
			name:      "date with microseconds",
			value:     "2024-01-15T10:30:00.123456Z",
			wantError: false,
		},
		{
			name:      "date with negative timezone",
			value:     "2024-01-15T10:30:00-05:00",
			wantError: false,
		},
		{
			name:      "date only - far past",
			value:     "1900-01-01",
			wantError: false,
		},
		{
			name:      "date only - far future",
			value:     "2100-12-31",
			wantError: false,
		},
		{
			name:      "leap year date",
			value:     "2024-02-29",
			wantError: false,
		},
		{
			name:      "invalid date format - month name",
			value:     "January 15, 2024",
			wantError: true,
		},
		{
			name:      "invalid date - February 30",
			value:     "2024-02-30",
			wantError: true,
		},
		{
			name:      "timestamp number is invalid",
			value:     1705315800,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("date_field", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Errorf("Expected error for value '%v', got nil", tc.value)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for value '%v', got: %v", tc.value, err)
				}
			}
		})
	}
}

// TestValidateFieldValue_NumberEdgeCases tests edge cases for number validation
func TestValidateFieldValue_NumberEdgeCases(t *testing.T) {
	fieldDef := models.FieldDefinition{
		Name:     "number_field",
		Type:     "number",
		Required: false,
	}

	testCases := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name:      "zero",
			value:     0,
			wantError: false,
		},
		{
			name:      "negative number",
			value:     -42.5,
			wantError: false,
		},
		{
			name:      "very large number",
			value:     1e308,
			wantError: false,
		},
		{
			name:      "very small positive number",
			value:     1e-308,
			wantError: false,
		},
		{
			name:      "scientific notation string",
			value:     "1.5e10",
			wantError: false,
		},
		{
			name:      "negative scientific notation string",
			value:     "-1.5e-10",
			wantError: false,
		},
		{
			name:      "string with leading zeros",
			value:     "042",
			wantError: false,
		},
		{
			name:      "string with trailing decimal",
			value:     "42.",
			wantError: false,
		},
		{
			name:      "empty string",
			value:     "",
			wantError: true,
		},
		{
			name:      "infinity",
			value:     "Infinity",
			wantError: false, // Go's strconv.ParseFloat accepts "Infinity"
		},
		{
			name:      "NaN string",
			value:     "NaN",
			wantError: false, // Go's strconv.ParseFloat accepts "NaN"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldValue("number_field", tc.value, fieldDef)

			if tc.wantError {
				if err == nil {
					t.Errorf("Expected error for value '%v', got nil", tc.value)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for value '%v', got: %v", tc.value, err)
				}
			}
		})
	}
}
