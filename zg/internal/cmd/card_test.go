package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunCardListUsesSearchEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("failed to decode search request body: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "1", "type": "card", "title": "First card", "preview": "body", "score": 1.0}
			],
			"page": 1, "per_page": 20, "total": 1, "total_pages": 1
		}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newCardListCmd()
	out := captureStdout(t, func() {
		if err := runCardList(cmd, nil); err != nil {
			t.Fatalf("runCardList failed: %v", err)
		}
	})

	// The general GET /api/cards route no longer exists; listing must go
	// through POST /api/search with an empty query.
	if gotMethod != http.MethodPost || gotPath != "/api/search" {
		t.Errorf("expected POST /api/search, got %s %s", gotMethod, gotPath)
	}
	if gotBody["search_term"] != "" {
		t.Errorf("expected empty search_term, got %q", gotBody["search_term"])
	}
	if gotBody["show_cards"] != true {
		t.Errorf("expected show_cards=true, got %v", gotBody["show_cards"])
	}
	if gotBody["per_page"] != float64(20) || gotBody["page"] != float64(1) {
		t.Errorf("expected per_page=20 page=1, got per_page=%v page=%v", gotBody["per_page"], gotBody["page"])
	}

	// The returned card should appear in the output.
	if !strings.Contains(out, "First card") {
		t.Errorf("expected output to contain card title, got: %s", out)
	}
}

func TestRunCardListOffsetToPageConversion(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode search request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": [], "page": 3, "per_page": 10, "total": 30, "total_pages": 3}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newCardListCmd()
	if err := cmd.Flags().Set("limit", "10"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("offset", "25"); err != nil {
		t.Fatal(err)
	}

	if err := runCardList(cmd, nil); err != nil {
		t.Fatalf("runCardList failed: %v", err)
	}

	// offset=25, limit=10 -> page = 25/10 + 1 = 3, per_page = 10
	if gotBody["page"] != float64(3) || gotBody["per_page"] != float64(10) {
		t.Errorf("expected page=3 per_page=10, got page=%v per_page=%v", gotBody["page"], gotBody["per_page"])
	}
}

func TestRunCardListStarred(t *testing.T) {
	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"id": 1, "card_pk": 5, "user_id": 1, "created_at": "2026-01-01T00:00:00Z",
			 "card": {"id": 5, "card_id": "5", "title": "Starred card", "body": "body", "link": ""}}
		]`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newCardListCmd()
	if err := cmd.Flags().Set("starred", "true"); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := runCardList(cmd, nil); err != nil {
			t.Fatalf("runCardList failed: %v", err)
		}
	})

	if gotMethod != http.MethodGet || gotPath != "/api/cards/starred" {
		t.Errorf("expected GET /api/cards/starred, got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "Starred card") {
		t.Errorf("expected output to contain starred card title, got: %s", out)
	}
}
