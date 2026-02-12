package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestSpreadsheetData tests SpreadsheetData struct and its JSON marshaling
func TestSpreadsheetData(t *testing.T) {
	data := SpreadsheetData{
		Rows: 3,
		Cols: 3,
		Data: map[string]SpreadsheetCell{
			"A1": {Value: "Hello", Formula: "", Computed: nil},
			"B1": {Value: "World", Formula: "", Computed: nil},
			"C1": {Value: "", Formula: "=A1&B1", Computed: ptrFloat64(0.0)},
		},
	}

	// Test JSON marshaling for JSONB storage
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal SpreadsheetData: %v", err)
	}

	// Verify JSON can be unmarshaled back
	var unmarshaled SpreadsheetData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SpreadsheetData: %v", err)
	}

	// Verify data integrity
	if unmarshaled.Rows != 3 {
		t.Errorf("Expected Rows to be 3, got %d", unmarshaled.Rows)
	}
	if unmarshaled.Cols != 3 {
		t.Errorf("Expected Cols to be 3, got %d", unmarshaled.Cols)
	}
	if len(unmarshaled.Data) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(unmarshaled.Data))
	}

	// Check specific cell
	cellA1, exists := unmarshaled.Data["A1"]
	if !exists {
		t.Error("Cell A1 not found in unmarshaled data")
	}
	if cellA1.Value != "Hello" {
		t.Errorf("Expected A1 Value to be 'Hello', got '%s'", cellA1.Value)
	}

	// Check formula cell
	cellC1, exists := unmarshaled.Data["C1"]
	if !exists {
		t.Error("Cell C1 not found in unmarshaled data")
	}
	if cellC1.Formula != "=A1&B1" {
		t.Errorf("Expected C1 Formula to be '=A1&B1', got '%s'", cellC1.Formula)
	}
}

// TestSpreadsheetCell tests SpreadsheetCell struct
func TestSpreadsheetCell(t *testing.T) {
	t.Run("Cell with value only", func(t *testing.T) {
		cell := SpreadsheetCell{
			Value:    "42",
			Formula:   "",
			Computed:  nil,
		}

		if cell.Value != "42" {
			t.Errorf("Expected Value to be '42', got '%s'", cell.Value)
		}
		if cell.Formula != "" {
			t.Errorf("Expected Formula to be empty, got '%s'", cell.Formula)
		}
		if cell.Computed != nil {
			t.Error("Expected Computed to be nil")
		}
	})

	t.Run("Cell with formula and computed value", func(t *testing.T) {
		val := 42.5
		cell := SpreadsheetCell{
			Value:    "",
			Formula:  "=SUM(A1:A10)",
			Computed:  &val,
		}

		if cell.Value != "" {
			t.Errorf("Expected Value to be empty, got '%s'", cell.Value)
		}
		if cell.Formula != "=SUM(A1:A10)" {
			t.Errorf("Expected Formula to be '=SUM(A1:A10)', got '%s'", cell.Formula)
		}
		if cell.Computed == nil {
			t.Error("Expected Computed to be non-nil")
		} else if *cell.Computed != 42.5 {
			t.Errorf("Expected Computed to be 42.5, got %f", *cell.Computed)
		}
	})

	t.Run("Cell marshaling", func(t *testing.T) {
		val := 100.0
		cell := SpreadsheetCell{
			Value:    "",
			Formula:  "=A1*2",
			Computed:  &val,
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(cell)
		if err != nil {
			t.Fatalf("Failed to marshal SpreadsheetCell: %v", err)
		}

		var unmarshaled SpreadsheetCell
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal SpreadsheetCell: %v", err)
		}

		if unmarshaled.Formula != "=A1*2" {
			t.Errorf("Expected Formula to be '=A1*2', got '%s'", unmarshaled.Formula)
		}
		if unmarshaled.Computed == nil {
			t.Error("Expected Computed to be non-nil")
		} else if *unmarshaled.Computed != 100.0 {
			t.Errorf("Expected Computed to be 100.0, got %f", *unmarshaled.Computed)
		}
	})
}

// TestSpreadsheetModel tests Spreadsheet struct
func TestSpreadsheetModel(t *testing.T) {
	spreadsheet := Spreadsheet{
		ID:        1,
		UserID:    1,
		CardID:    100,
		Name:      "My Budget",
		Rows:      10,
		Cols:      5,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if spreadsheet.Name != "My Budget" {
		t.Errorf("Expected Name to be 'My Budget', got '%s'", spreadsheet.Name)
	}

	if spreadsheet.Rows != 10 {
		t.Errorf("Expected Rows to be 10, got %d", spreadsheet.Rows)
	}

	if spreadsheet.Cols != 5 {
		t.Errorf("Expected Cols to be 5, got %d", spreadsheet.Cols)
	}
}

// TestSpreadsheetDataWithEmptyCells tests SpreadsheetData with empty cells map
func TestSpreadsheetDataWithEmptyCells(t *testing.T) {
	data := SpreadsheetData{
		Rows: 0,
		Cols: 0,
		Data: map[string]SpreadsheetCell{},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal empty SpreadsheetData: %v", err)
	}

	var unmarshaled SpreadsheetData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty SpreadsheetData: %v", err)
	}

	if unmarshaled.Rows != 0 {
		t.Errorf("Expected Rows to be 0, got %d", unmarshaled.Rows)
	}
	if unmarshaled.Cols != 0 {
		t.Errorf("Expected Cols to be 0, got %d", unmarshaled.Cols)
	}
	if len(unmarshaled.Data) != 0 {
		t.Errorf("Expected 0 cells, got %d", len(unmarshaled.Data))
	}
}

// TestSpreadsheetCellWithNilComputed tests cell with nil computed value
func TestSpreadsheetCellWithNilComputed(t *testing.T) {
	cell := SpreadsheetCell{
		Value:    "text",
		Formula:   "",
		Computed:  nil,
	}

	jsonData, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("Failed to marshal SpreadsheetCell with nil Computed: %v", err)
	}

	var unmarshaled SpreadsheetCell
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SpreadsheetCell with nil Computed: %v", err)
	}

	if unmarshaled.Computed != nil {
		t.Error("Expected Computed to remain nil after marshal/unmarshal")
	}
}

// Helper function to get a pointer to a float64
func ptrFloat64(f float64) *float64 {
	return &f
}
