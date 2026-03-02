package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"id": 123, "title": "Test"}

	WriteSuccess(&buf, data)

	result := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
	if parsed["data"] == nil {
		t.Error("Expected data field")
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer

	WriteError(&buf, "Something went wrong", "optional details")

	result := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["success"] != false {
		t.Errorf("Expected success=false, got %v", parsed["success"])
	}
	if parsed["error"] != "Something went wrong" {
		t.Errorf("Expected error message, got %v", parsed["error"])
	}
}

func TestWriteList(t *testing.T) {
	var buf bytes.Buffer
	items := []string{"item1", "item2"}

	WriteList(&buf, items, 10, 5, 0)

	result := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
	if parsed["total"] != 10.0 {
		t.Errorf("Expected total=10, got %v", parsed["total"])
	}
	if parsed["limit"] != 5.0 {
		t.Errorf("Expected limit=5, got %v", parsed["limit"])
	}
}

func TestIsTTY(t *testing.T) {
	// Just verify the function runs and returns a bool
	result := IsTTY()
	if result != true && result != false {
		t.Error("IsTTY should return boolean")
	}
}
