package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunCardStar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/cards/3/star" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runCardStar(cardStarCmd, []string{"3"}); err != nil {
			t.Fatalf("runCardStar: %v", err)
		}
	})
	if !strings.Contains(out, "Card 3 starred") {
		t.Errorf("expected star message, got %q", out)
	}
}

func TestRunCardUnstar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/cards/3/star" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runCardUnstar(cardUnstarCmd, []string{"3"}); err != nil {
			t.Fatalf("runCardUnstar: %v", err)
		}
	})
	if !strings.Contains(out, "Card 3 unstarred") {
		t.Errorf("expected unstar message, got %q", out)
	}
}

func TestRunCardChildren(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/cards/2/children" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 5, "card_id": "2.1", "title": "Child one"},
			{"id": 6, "card_id": "2.2", "title": "Child two"},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runCardChildren(cardChildrenCmd, []string{"2"}); err != nil {
			t.Fatalf("runCardChildren: %v", err)
		}
	})
	if !strings.Contains(out, `"title":"Child one"`) || !strings.Contains(out, `"title":"Child two"`) {
		t.Errorf("expected children in output, got %q", out)
	}
}
