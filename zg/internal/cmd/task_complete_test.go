package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunTaskCompleteSendsCompletePayload(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTaskComplete(&cobra.Command{}, []string{"42"}); err != nil {
			t.Fatalf("runTaskComplete failed: %v", err)
		}
	})

	// `zg task complete <id>` must issue a real update (is_complete + status)
	// instead of aliasing runTaskUpdate with no flags set (which errored
	// "No updates").
	if gotMethod != http.MethodPut || gotPath != "/api/tasks/42" {
		t.Errorf("expected PUT /api/tasks/42, got %s %s", gotMethod, gotPath)
	}
	if gotBody["is_complete"] != true {
		t.Errorf("expected is_complete=true, got %v", gotBody["is_complete"])
	}
	if gotBody["status"] != "done" {
		t.Errorf("expected status=done, got %v", gotBody["status"])
	}
	if !strings.Contains(out, "complete") {
		t.Errorf("expected completion message in output, got: %s", out)
	}
}
