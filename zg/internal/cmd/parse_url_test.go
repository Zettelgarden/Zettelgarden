package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunParseURL(t *testing.T) {
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/url/parse" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"title":     "Example Article",
			"content":   "## Hello\n\nmarkdown body",
			"url":       "https://example.com/post",
			"site_name": "Example",
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runParseURL(parseURLCmd, []string{"https://example.com/post"}); err != nil {
			t.Fatalf("runParseURL: %v", err)
		}
	})
	if gotBody["url"] != "https://example.com/post" {
		t.Errorf("parse body = %v, want url sent", gotBody)
	}
	if !strings.Contains(out, `"title":"Example Article"`) || !strings.Contains(out, "markdown body") {
		t.Errorf("expected parsed content in output, got %q", out)
	}
}
