package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSchemaList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/schemas" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Person", "slug": "person", "fields": []any{}, "card_count": 3},
			{"id": 2, "name": "Book", "slug": "book", "fields": []any{}},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runSchemaList(schemaListCmd, nil); err != nil {
			t.Fatalf("runSchemaList: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"Person"`) || !strings.Contains(out, `"name":"Book"`) {
		t.Errorf("expected schemas in output, got %q", out)
	}
	if !strings.Contains(out, `"card_count":3`) {
		t.Errorf("expected card_count in output, got %q", out)
	}
}

func TestRunSchemaGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/schemas/4" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   4,
			"name": "Meeting",
			"slug": "meeting",
			"fields": []map[string]any{
				{"name": "date", "type": "date"},
			},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runSchemaGet(schemaGetCmd, []string{"4"}); err != nil {
			t.Fatalf("runSchemaGet: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"Meeting"`) || !strings.Contains(out, `"date"`) {
		t.Errorf("expected schema fields in output, got %q", out)
	}
}
