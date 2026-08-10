package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunTagList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/tags" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "alpha", "color": "black"},
			{"id": 2, "name": "beta", "color": "red"},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTagList(tagListCmd, nil); err != nil {
			t.Fatalf("runTagList: %v", err)
		}
	})
	if !strings.Contains(out, `"alpha"`) || !strings.Contains(out, `"beta"`) {
		t.Errorf("expected both tags in output, got %q", out)
	}
}

func TestRunTagCreate(t *testing.T) {
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/tags" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 9, "name": "mytag", "color": "blue"})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	tagColor = "blue"
	t.Cleanup(func() { tagColor = "" })

	out := captureStdout(t, func() {
		if err := runTagCreate(tagCreateCmd, []string{"mytag"}); err != nil {
			t.Fatalf("runTagCreate: %v", err)
		}
	})
	if gotBody["name"] != "mytag" || gotBody["color"] != "blue" {
		t.Errorf("create body = %v, want name=mytag color=blue", gotBody)
	}
	if !strings.Contains(out, `"name":"mytag"`) {
		t.Errorf("expected created tag in output, got %q", out)
	}
}

func TestRunTagDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/tags/id/5" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTagDelete(tagDeleteCmd, []string{"5"}); err != nil {
			t.Fatalf("runTagDelete: %v", err)
		}
	})
	if !strings.Contains(out, "Tag 5 deleted") {
		t.Errorf("expected delete message, got %q", out)
	}
}

func TestRunTagCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/cards/7/tags" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "alpha"}})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTagCards(tagCardsCmd, []string{"7"}); err != nil {
			t.Fatalf("runTagCards: %v", err)
		}
	})
	if !strings.Contains(out, `"alpha"`) {
		t.Errorf("expected card tag in output, got %q", out)
	}
}

func TestRunTagAdd(t *testing.T) {
	var putBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/cards/1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "card_id": "1a", "title": "Hello", "body": "Some body", "link": ""})
		case r.Method == http.MethodPut && r.URL.Path == "/api/cards/1":
			json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": "Hello"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTagAdd(tagAddCmd, []string{"1", "mytag"}); err != nil {
			t.Fatalf("runTagAdd: %v", err)
		}
	})
	body, _ := putBody["body"].(string)
	if !strings.Contains(body, "#mytag") {
		t.Errorf("expected #mytag appended to body, got %q", body)
	}
	if !strings.Contains(out, "tagged #mytag") {
		t.Errorf("expected success message, got %q", out)
	}
}

func TestRunTagAddAlreadyTagged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/cards/1" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "card_id": "1a", "title": "Hello", "body": "Has #mytag here", "link": ""})
			return
		}
		t.Errorf("unexpected PUT after no-op tag add: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTagAdd(tagAddCmd, []string{"1", "mytag"}); err != nil {
			t.Fatalf("runTagAdd: %v", err)
		}
	})
	if !strings.Contains(out, "already tagged") {
		t.Errorf("expected already-tagged message, got %q", out)
	}
}

func TestTagInBody(t *testing.T) {
	cases := []struct {
		body, tag string
		want      bool
	}{
		{"has #alpha here", "alpha", true},
		{"has #alpha-beta here", "alpha", false}, // hyphen is part of the tag token
		{"has #alpha here", "alphax", false},
		{"has #alpha. here", "alpha", true}, // punctuation ends the token
		{"has #alpha##beta", "alpha", true},
		{"has #alphax here", "alpha", false},
		{"", "alpha", false},
		{"line1\n#alpha", "alpha", true},
	}
	for _, tc := range cases {
		if got := tagInBody(tc.body, tc.tag); got != tc.want {
			t.Errorf("tagInBody(%q, %q) = %v, want %v", tc.body, tc.tag, got, tc.want)
		}
	}
}
