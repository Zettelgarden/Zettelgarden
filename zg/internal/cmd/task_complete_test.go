package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// taskResponse (in task_test.go) is the full task object the backend returns
// from GET /api/tasks/{id}; the mock below reuses it.

func TestRunTaskCompletePreservesTaskData(t *testing.T) {
	var gotMethod, gotPath string
	var putBody map[string]any
	gotRequests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequests++
		gotMethod = r.Method
		gotPath = r.URL.Path
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(taskResponse))
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
				t.Errorf("failed to decode body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runTaskComplete(&cobra.Command{}, []string{"42"}); err != nil {
			t.Fatalf("runTaskComplete failed: %v", err)
		}
	})

	// `zg task complete <id>` must GET the task and PUT the full object back,
	// not send a partial payload (the backend PUT replaces the whole row).
	if gotRequests != 2 {
		t.Errorf("expected GET + PUT (2 requests), got %d", gotRequests)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/tasks/42" {
		t.Errorf("expected final PUT /api/tasks/42, got %s %s", gotMethod, gotPath)
	}

	// The other task fields must survive the completion update.
	if putBody["title"] != "Original title" {
		t.Errorf("expected title preserved, got %v", putBody["title"])
	}
	if putBody["card_pk"] != float64(7) {
		t.Errorf("expected card_pk preserved, got %v", putBody["card_pk"])
	}
	if putBody["description"] != "Original description" {
		t.Errorf("expected description preserved, got %v", putBody["description"])
	}
	if putBody["priority"] != "high" {
		t.Errorf("expected priority preserved, got %v", putBody["priority"])
	}
	if putBody["is_complete"] != true {
		t.Errorf("expected is_complete=true, got %v", putBody["is_complete"])
	}
	if putBody["status"] != "done" {
		t.Errorf("expected status=done, got %v", putBody["status"])
	}
	if !strings.Contains(out, "complete") {
		t.Errorf("expected completion message in output, got: %s", out)
	}
}
