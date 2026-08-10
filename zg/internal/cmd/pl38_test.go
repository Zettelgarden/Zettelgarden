package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunCardUnsorted(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"cards": []map[string]any{
				{"id": 1, "card_id": "", "title": "Unsorted one"},
			},
			"page": 1, "per_page": 20, "total": 1, "total_pages": 1,
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	cmd := newCardListCmd() // binds the same --limit/--offset globals as card unsorted
	out := captureStdout(t, func() {
		if err := runCardUnsorted(cmd, nil); err != nil {
			t.Fatalf("runCardUnsorted: %v", err)
		}
	})
	if !strings.Contains(gotPath, "/api/cards/unsorted?page=1&per_page=20") {
		t.Errorf("unexpected request %q", gotPath)
	}
	if !strings.Contains(out, `"title":"Unsorted one"`) {
		t.Errorf("expected unsorted card in output, got %q", out)
	}
}

func TestRunCardSuggestTitle(t *testing.T) {
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/cards/suggest-title" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"suggested_title": "A Great Title"})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runCardSuggestTitle(cardSuggestTitleCmd, []string{"Some body text here"}); err != nil {
			t.Fatalf("runCardSuggestTitle: %v", err)
		}
	})
	if gotBody["body"] != "Some body text here" {
		t.Errorf("body sent = %q", gotBody["body"])
	}
	if !strings.Contains(out, `"suggested_title":"A Great Title"`) {
		t.Errorf("expected suggested title in output, got %q", out)
	}
}

func TestRunStats(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats/daily" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"stats": []map[string]any{
				{"date": "2026-08-01T00:00:00Z", "cards_created": 3, "tasks_created": 1, "tasks_completed": 2},
			},
			"total": map[string]any{"cards_created": 3, "tasks_created": 1, "tasks_completed": 2},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runStats(statsCmd, nil); err != nil {
			t.Fatalf("runStats: %v", err)
		}
	})
	if !strings.Contains(gotQuery, "start_date=") || !strings.Contains(gotQuery, "end_date=") {
		t.Errorf("expected date range query, got %q", gotQuery)
	}
	if !strings.Contains(out, `"cards_created":3`) {
		t.Errorf("expected stats totals in output, got %q", out)
	}
}

func TestRunStatsInvalidDays(t *testing.T) {
	writeTestConfig(t, "http://example.com")
	statsDays = 0
	t.Cleanup(func() { statsDays = 30 })

	out := captureStdout(t, func() {
		if err := runStats(statsCmd, nil); err != nil {
			t.Fatalf("runStats: %v", err)
		}
	})
	if !strings.Contains(out, "Invalid --days") {
		t.Errorf("expected invalid-days error, got %q", out)
	}
}

func TestRunAuthKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/api-keys" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"api_keys": []map[string]any{
				{"id": 7, "name": "zg-cli", "is_active": true},
			},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runAuthKeys(authKeysCmd, nil); err != nil {
			t.Fatalf("runAuthKeys: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"zg-cli"`) || !strings.Contains(out, `"is_active":true`) {
		t.Errorf("expected key metadata in output, got %q", out)
	}
}
