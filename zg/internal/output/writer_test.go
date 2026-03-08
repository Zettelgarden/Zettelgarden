package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Test type that implements HumanFormatter
type testCard struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

func (c testCard) FormatHuman() string {
	return "Card: " + c.Title
}

func (c testCard) FormatListHeader() string {
	return "ID  Title"
}

func (c testCard) FormatListItem() string {
	return "Card item"
}

func TestWriteSuccessJSON(t *testing.T) {
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

func TestWriteSuccessPretty(t *testing.T) {
	var buf bytes.Buffer
	SetPretty(true)
	defer SetPretty(false)

	card := testCard{ID: 1, Title: "My Card"}
	WriteSuccess(&buf, card)

	result := buf.String()
	if !strings.Contains(result, "Card: My Card") {
		t.Errorf("Expected human format, got: %s", result)
	}
}

func TestWriteErrorJSON(t *testing.T) {
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

func TestWriteErrorPretty(t *testing.T) {
	var buf bytes.Buffer
	SetPretty(true)
	defer SetPretty(false)

	WriteError(&buf, "Something went wrong", "optional details")

	result := buf.String()
	if !strings.Contains(result, "Error: Something went wrong") {
		t.Errorf("Expected human error format, got: %s", result)
	}
	if !strings.Contains(result, "optional details") {
		t.Errorf("Expected details in output, got: %s", result)
	}
}

func TestWriteListJSON(t *testing.T) {
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

func TestWriteListPretty(t *testing.T) {
	var buf bytes.Buffer
	SetPretty(true)
	defer SetPretty(false)

	items := []testCard{{ID: 1, Title: "Card 1"}, {ID: 2, Title: "Card 2"}}
	WriteList(&buf, items, 10, 5, 0)

	result := buf.String()
	if !strings.Contains(result, "ID  Title") {
		t.Errorf("Expected header in output, got: %s", result)
	}
	if !strings.Contains(result, "Card item") {
		t.Errorf("Expected list items in output, got: %s", result)
	}
}

func TestWriteMessageJSON(t *testing.T) {
	var buf bytes.Buffer

	WriteMessage(&buf, "Card deleted")

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

func TestWriteMessagePretty(t *testing.T) {
	var buf bytes.Buffer
	SetPretty(true)
	defer SetPretty(false)

	WriteMessage(&buf, "Card deleted")

	result := buf.String()
	if result != "Card deleted\n" {
		t.Errorf("Expected simple message, got: %q", result)
	}
}

func TestIsTTY(t *testing.T) {
	// Just verify the function runs and returns a bool
	result := IsTTY()
	if result != true && result != false {
		t.Error("IsTTY should return boolean")
	}
}
