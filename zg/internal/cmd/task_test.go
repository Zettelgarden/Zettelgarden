package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// taskResponse is the full task object the backend returns from
// GET /api/tasks/{id}.
const taskResponse = `{
	"id": 42,
	"card_pk": 7,
	"user_id": 1,
	"title": "Original title",
	"description": "Original description",
	"priority": "high",
	"status": "in_progress",
	"is_complete": false,
	"scheduled_date": "2026-08-10T00:00:00Z",
	"due_date": null,
	"reminder_time": null
}`

// updateTaskMockServer serves GET /api/tasks/42 (full task) followed by a PUT,
// capturing the PUT body, and returns it for assertions.
func updateTaskMockServer(t *testing.T) (*httptest.Server, *map[string]any, *int) {
	t.Helper()
	var putBody map[string]any
	putCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(taskResponse))
		case http.MethodPut:
			putCount++
			if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
				t.Errorf("failed to decode body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	return server, &putBody, &putCount
}

func TestRunTaskUpdateIncomplete(t *testing.T) {
	server, putBody, putCount := updateTaskMockServer(t)
	defer server.Close()
	writeTestConfig(t, server.URL)

	cmd := newTaskUpdateCmd()
	if err := cmd.Flags().Set("incomplete", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runTaskUpdate(cmd, []string{"42"}); err != nil {
		t.Fatalf("runTaskUpdate failed: %v", err)
	}

	// --incomplete must mark the task incomplete (is_complete=false), not
	// complete as the old shared-flag binding did.
	if (*putBody)["is_complete"] != false {
		t.Errorf("expected is_complete=false for --incomplete, got %v", (*putBody)["is_complete"])
	}
	// And the untouched fields must survive the replace-PUT.
	if (*putBody)["title"] != "Original title" {
		t.Errorf("expected title preserved, got %v", (*putBody)["title"])
	}
	if (*putBody)["card_pk"] != float64(7) {
		t.Errorf("expected card_pk preserved, got %v", (*putBody)["card_pk"])
	}
	if *putCount != 1 {
		t.Errorf("expected 1 PUT, got %d", *putCount)
	}
}

func TestRunTaskUpdateComplete(t *testing.T) {
	server, putBody, _ := updateTaskMockServer(t)
	defer server.Close()
	writeTestConfig(t, server.URL)

	cmd := newTaskUpdateCmd()
	if err := cmd.Flags().Set("complete", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runTaskUpdate(cmd, []string{"42"}); err != nil {
		t.Fatalf("runTaskUpdate failed: %v", err)
	}

	if (*putBody)["is_complete"] != true {
		t.Errorf("expected is_complete=true for --complete, got %v", (*putBody)["is_complete"])
	}
}

func TestRunTaskUpdateConflictingFlags(t *testing.T) {
	server, _, putCount := updateTaskMockServer(t)
	defer server.Close()
	writeTestConfig(t, server.URL)

	cmd := newTaskUpdateCmd()
	if err := cmd.Flags().Set("complete", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("incomplete", "true"); err != nil {
		t.Fatal(err)
	}

	err := runTaskUpdate(cmd, []string{"42"})
	if err != nil {
		t.Fatalf("runTaskUpdate returned err: %v (errors are written to stdout)", err)
	}
	if *putCount != 0 {
		t.Errorf("expected no PUT for conflicting flags, got %d", *putCount)
	}

	out := captureStdout(t, func() {
		_ = runTaskUpdate(cmd, []string{"42"})
	})
	if !strings.Contains(out, "Conflicting flags") {
		t.Errorf("expected conflicting-flags error in output, got: %s", out)
	}
}

func TestRunTaskListIncomplete(t *testing.T) {
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tasks": [], "total": 0, "limit": 50, "offset": 0}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newTaskListCmd()
	if err := cmd.Flags().Set("incomplete", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("runTaskList failed: %v", err)
	}

	// --incomplete must send completed=false (show incomplete tasks), not
	// completed=true as the old shared-flag binding did.
	if !strings.Contains(gotQuery, "completed=false") {
		t.Errorf("expected completed=false in query, got %q", gotQuery)
	}
}

func TestRunTaskListCompleted(t *testing.T) {
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tasks": [], "total": 0, "limit": 50, "offset": 0}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newTaskListCmd()
	if err := cmd.Flags().Set("completed", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("runTaskList failed: %v", err)
	}

	if !strings.Contains(gotQuery, "completed=true") {
		t.Errorf("expected completed=true in query, got %q", gotQuery)
	}
}

func TestRunTaskListConflictingFlags(t *testing.T) {
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tasks": [], "total": 0, "limit": 50, "offset": 0}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newTaskListCmd()
	if err := cmd.Flags().Set("completed", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("incomplete", "true"); err != nil {
		t.Fatal(err)
	}

	err := runTaskList(cmd, nil)
	if err != nil {
		t.Fatalf("runTaskList returned err: %v (errors are written to stdout)", err)
	}
	if gotQuery != "" {
		t.Errorf("expected no request for conflicting flags, got query %q", gotQuery)
	}

	out := captureStdout(t, func() {
		_ = runTaskList(cmd, nil)
	})
	if !strings.Contains(out, "Conflicting flags") {
		t.Errorf("expected conflicting-flags error in output, got: %s", out)
	}
}
