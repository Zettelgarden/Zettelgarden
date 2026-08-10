package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunArticleCreate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 9, "card_id": "9", "title": "Parsed article", "body": "markdown", "link": "https://example.com/post"}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().StringVarP(&articleCardID, "card-id", "c", "", "Optional card ID")
	cmd.Flags().StringVarP(&articleTags, "tags", "t", "", "Custom tags")

	if err := cmd.Flags().Set("card-id", "4.2"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("tags", "#read-later"); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := runArticleCreate(cmd, []string{"https://example.com/post"}); err != nil {
			t.Fatalf("runArticleCreate failed: %v", err)
		}
	})

	// Article creation goes to the live POST /api/articles route.
	if gotMethod != http.MethodPost || gotPath != "/api/articles" {
		t.Errorf("expected POST /api/articles, got %s %s", gotMethod, gotPath)
	}
	if gotBody["url"] != "https://example.com/post" {
		t.Errorf("expected url in body, got %v", gotBody["url"])
	}
	if gotBody["card_id"] != "4.2" {
		t.Errorf("expected card_id in body, got %v", gotBody["card_id"])
	}
	if gotBody["tags"] != "#read-later" {
		t.Errorf("expected tags in body, got %v", gotBody["tags"])
	}

	// The created card should be echoed back.
	if !strings.Contains(out, "Parsed article") {
		t.Errorf("expected created card in output, got: %s", out)
	}
}
