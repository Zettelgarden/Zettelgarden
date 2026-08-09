package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunTaskUpdateIncomplete(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
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
	if gotBody["is_complete"] != false {
		t.Errorf("expected is_complete=false for --incomplete, got %v", gotBody["is_complete"])
	}
}

func TestRunTaskUpdateComplete(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	writeTestConfig(t, server.URL)

	cmd := newTaskUpdateCmd()
	if err := cmd.Flags().Set("complete", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runTaskUpdate(cmd, []string{"42"}); err != nil {
		t.Fatalf("runTaskUpdate failed: %v", err)
	}

	if gotBody["is_complete"] != true {
		t.Errorf("expected is_complete=true for --complete, got %v", gotBody["is_complete"])
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
