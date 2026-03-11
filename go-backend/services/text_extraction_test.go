package services

import (
	"strings"
	"testing"
)

func TestExtractTextFromPlainText(t *testing.T) {
	text := "Hello, this is a test document."
	reader := strings.NewReader(text)

	result, err := ExtractText("text/plain", reader)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if result != text {
		t.Errorf("Expected %q, got %q", text, result)
	}
}

func TestExtractTextFromUnsupportedType(t *testing.T) {
	text, err := ExtractText("image/png", nil)
	if err != nil {
		t.Errorf("Expected no error for unsupported type, got: %v", err)
	}
	if text != "" {
		t.Error("Expected empty string for unsupported type")
	}
}

func TestExtractTextTruncation(t *testing.T) {
	// Create a large text that exceeds 100KB
	largeText := strings.Repeat("x", 150*1024)
	reader := strings.NewReader(largeText)

	result, err := ExtractText("text/plain", reader)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if !strings.Contains(result, "[TRUNCATED]") {
		t.Error("Expected [TRUNCATED] marker in result")
	}

	if len(result) > 110*1024 { // Allow some overhead
		t.Errorf("Result too large: %d bytes", len(result))
	}
}

func TestExtractTextFromPDF(t *testing.T) {
	// This test would require a real PDF file
	// Skip if no test data available
	t.Skip("Requires sample PDF file in testdata/")
}
