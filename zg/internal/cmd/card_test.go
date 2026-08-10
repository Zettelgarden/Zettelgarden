package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// searchPageServer returns a mock that serves POST /api/search with results
// named "Card <n>" for the requested page (per_page items per page), so
// pagination math can be asserted by inspecting the served pages and output.
func searchPageServer(t *testing.T, requests *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode search request body: %v", err)
		}
		*requests = append(*requests, body)

		page := int(body["page"].(float64))
		perPage := int(body["per_page"].(float64))
		start := (page - 1) * perPage

		var results []map[string]any
		for i := start; i < start+perPage; i++ {
			results = append(results, map[string]any{
				"id":      fmt.Sprintf("%d", i),
				"type":    "card",
				"title":   fmt.Sprintf("Card %d", i),
				"preview": "body",
				"score":   1.0,
			})
		}
		resp := map[string]any{
			"results":     results,
			"page":        page,
			"per_page":    perPage,
			"total":       100,
			"total_pages": 10,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRunCardListUsesSearchEndpoint(t *testing.T) {
	var requests []map[string]any
	server := searchPageServer(t, &requests)
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
	if len(requests) != 1 {
		t.Fatalf("expected 1 search request, got %d", len(requests))
	}
	if requests[0]["search_term"] != "" {
		t.Errorf("expected empty search_term, got %q", requests[0]["search_term"])
	}
	if requests[0]["show_cards"] != true {
		t.Errorf("expected show_cards=true, got %v", requests[0]["show_cards"])
	}
	if requests[0]["per_page"] != float64(20) || requests[0]["page"] != float64(1) {
		t.Errorf("expected per_page=20 page=1, got per_page=%v page=%v", requests[0]["per_page"], requests[0]["page"])
	}

	// The returned card should appear in the output.
	if !strings.Contains(out, "Card 0") {
		t.Errorf("expected output to contain card title, got: %s", out)
	}
}

func TestRunCardListOffsetToPageConversion(t *testing.T) {
	var requests []map[string]any
	server := searchPageServer(t, &requests)
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newCardListCmd()
	if err := cmd.Flags().Set("limit", "10"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("offset", "25"); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := runCardList(cmd, nil); err != nil {
			t.Fatalf("runCardList failed: %v", err)
		}
	})

	// offset=25, limit=10: page = 25/10+1 = 3 with per_page=10, plus a second
	// request to page 4 so the leading remainder (5) can be dropped and the
	// exact window [25, 35) returned.
	if len(requests) != 2 {
		t.Fatalf("expected 2 search requests for non-multiple offset, got %d", len(requests))
	}
	if requests[0]["page"] != float64(3) || requests[0]["per_page"] != float64(10) {
		t.Errorf("expected first request page=3 per_page=10, got %v", requests[0])
	}
	if requests[1]["page"] != float64(4) || requests[1]["per_page"] != float64(10) {
		t.Errorf("expected second request page=4 per_page=10, got %v", requests[1])
	}

	// Output must be exactly cards 25..34 (no skew from the old page math).
	for _, want := range []string{"Card 25", "Card 29", "Card 34"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
	for _, notWant := range []string{"Card 24", "Card 35"} {
		if strings.Contains(out, notWant) {
			t.Errorf("did not expect %q in output, got: %s", notWant, out)
		}
	}
}

func TestRunCardListClampsLimitToBackendCap(t *testing.T) {
	var requests []map[string]any
	server := searchPageServer(t, &requests)
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newCardListCmd()
	if err := cmd.Flags().Set("limit", "500"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("offset", "500"); err != nil {
		t.Fatal(err)
	}

	if err := runCardList(cmd, nil); err != nil {
		t.Fatalf("runCardList failed: %v", err)
	}

	// limit is clamped to the backend cap (100), and offset is expressed
	// against that clamped page size.
	if requests[0]["per_page"] != float64(100) || requests[0]["page"] != float64(6) {
		t.Errorf("expected per_page=100 page=6, got %v", requests[0])
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
