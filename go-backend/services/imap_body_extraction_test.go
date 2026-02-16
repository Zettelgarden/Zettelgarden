package services

import (
	"strings"
	"testing"
)

// TestIMAPClient_extractPlainText tests the plain text extraction fallback
func TestIMAPClient_extractPlainText(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple email with headers",
			input:    "Subject: Test\nFrom: sender@example.com\n\nThis is the body",
			expected: "This is the body",
		},
		{
			name:     "Email with CRLF line endings",
			input:    "Subject: Test\r\nFrom: sender@example.com\r\n\r\nBody content here",
			expected: "Body content here",
		},
		{
			name:     "Email with multiple newlines before body",
			input:    "Subject: Test\n\n\nBody after newlines",
			expected: "Body after newlines",
		},
		{
			name:     "No clear header boundary",
			input:    "Just some text without blank lines",
			expected: "Just some text without blank lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.extractPlainText([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("extractPlainText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestIMAPClient_extractPlainText_truncation tests that large emails are truncated
func TestIMAPClient_extractPlainText_truncation(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	// Create a large email (> 10000 chars) WITHOUT a clear header boundary
	// This tests the truncation fallback
	largeContent := strings.Repeat("A", 12000)
	input := largeContent // No "\n\n" or "\r\n\r\n" separator

	result := client.extractPlainText([]byte(input))

	// Should be truncated
	if len(result) > 10100 { // 10000 + "... (truncated)"
		t.Errorf("Expected truncated result, got length %d", len(result))
	}
	if !strings.Contains(result, "(truncated)") {
		t.Errorf("Expected truncation marker in result, got: %s", result[len(result)-15:])
	}
}
