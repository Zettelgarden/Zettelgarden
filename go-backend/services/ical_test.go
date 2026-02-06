package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchICalURLWithAuth(t *testing.T) {
	// Create test server that requires auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Basic Auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return valid iCal content
		w.Header().Set("Content-Type", "text/calendar")
		w.Write([]byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-event-1
DTSTART:20250201T100000Z
DTEND:20250201T110000Z
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR`))
	}))
	defer server.Close()

	t.Run("without credentials returns 401", func(t *testing.T) {
		_, err := FetchICalURL(server.URL, "", "")
		if err == nil {
			t.Error("Expected error without credentials")
		}
	})

	t.Run("with valid credentials succeeds", func(t *testing.T) {
		events, err := FetchICalURL(server.URL, "testuser", "testpass")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})

	t.Run("with invalid credentials returns 401", func(t *testing.T) {
		_, err := FetchICalURL(server.URL, "wrong", "credentials")
		if err == nil {
			t.Error("Expected error with invalid credentials")
		}
	})
}

func TestFetchICalURLWithoutAuth(t *testing.T) {
	// Create test server that doesn't require auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		w.Write([]byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-event-1
DTSTART:20250201T100000Z
DTEND:20250201T110000Z
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR`))
	}))
	defer server.Close()

	t.Run("empty credentials works", func(t *testing.T) {
		events, err := FetchICalURL(server.URL, "", "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})
}

func TestParseICalendarWithTodos(t *testing.T) {
	t.Run("parses VTODO components", func(t *testing.T) {
		icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo-1
DTSTART:20250201T100000Z
DUE:20250201T120000Z
SUMMARY:Test Task
DESCRIPTION:Test Description
PRIORITY:5
STATUS:NEEDS-ACTION
PERCENT-COMPLETE:50
END:VTODO
END:VCALENDAR`

		events, todos, err := ParseICalendarWithTodos(strings.NewReader(icalData))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
		if len(todos) != 1 {
			t.Fatalf("Expected 1 todo, got %d", len(todos))
		}

		todo := todos[0]
		if todo.UID != "todo-1" {
			t.Errorf("Expected UID 'todo-1', got '%s'", todo.UID)
		}
		if todo.Summary != "Test Task" {
			t.Errorf("Expected Summary 'Test Task', got '%s'", todo.Summary)
		}
		if todo.Description != "Test Description" {
			t.Errorf("Expected Description 'Test Description', got '%s'", todo.Description)
		}
		if todo.Priority != 5 {
			t.Errorf("Expected Priority 5, got %d", todo.Priority)
		}
		if todo.Status != "NEEDS-ACTION" {
			t.Errorf("Expected Status 'NEEDS-ACTION', got '%s'", todo.Status)
		}
		if todo.PercentComplete != 50 {
			t.Errorf("Expected PercentComplete 50, got %d", todo.PercentComplete)
		}
	})

	t.Run("parses VTODO with COMPLETED status", func(t *testing.T) {
		icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo-2
SUMMARY:Completed Task
STATUS:COMPLETED
COMPLETED:20250201T120000Z
END:VTODO
END:VCALENDAR`

		_, todos, err := ParseICalendarWithTodos(strings.NewReader(icalData))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(todos) != 1 {
			t.Fatalf("Expected 1 todo, got %d", len(todos))
		}

		todo := todos[0]
		if todo.Completed.IsZero() {
			t.Errorf("Expected Completed timestamp to be set")
		}
	})

	t.Run("parses mixed VEVENT and VTODO", func(t *testing.T) {
		icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:event-1
DTSTART:20250201T100000Z
DTEND:20250201T110000Z
SUMMARY:Test Event
END:VEVENT
BEGIN:VTODO
UID:todo-1
SUMMARY:Test Task
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`

		events, todos, err := ParseICalendarWithTodos(strings.NewReader(icalData))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		if len(todos) != 1 {
			t.Errorf("Expected 1 todo, got %d", len(todos))
		}
	})

	t.Run("skips VTODO without UID", func(t *testing.T) {
		icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
SUMMARY:Task without UID
END:VTODO
END:VCALENDAR`

		_, todos, err := ParseICalendarWithTodos(strings.NewReader(icalData))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(todos) != 0 {
			t.Errorf("Expected 0 todos (skipped), got %d", len(todos))
		}
	})

	t.Run("handles VTODO without due date", func(t *testing.T) {
		icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo-3
SUMMARY:Task without due
STATUS:IN-PROCESS
END:VTODO
END:VCALENDAR`

		_, todos, err := ParseICalendarWithTodos(strings.NewReader(icalData))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(todos) != 1 {
			t.Fatalf("Expected 1 todo, got %d", len(todos))
		}

		todo := todos[0]
		if !todo.Due.IsZero() {
			t.Errorf("Expected zero Due time, got %v", todo.Due)
		}
	})
}

func TestFetchICalURLWithTodos(t *testing.T) {
	t.Run("fetches VTODO components from feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/calendar")
			w.Write([]byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo-1
DTSTART:20250201T100000Z
DUE:20250201T120000Z
SUMMARY:Test Task from Feed
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`))
		}))
		defer server.Close()

		events, todos, err := FetchICalURLWithTodos(server.URL, "", "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
		if len(todos) != 1 {
			t.Fatalf("Expected 1 todo, got %d", len(todos))
		}
	})
}

func TestMapTodoStatus(t *testing.T) {
	t.Run("maps COMPLETED status to done=true", func(t *testing.T) {
		status, isComplete := mapTodoStatus("COMPLETED", 0)
		if status != "done" {
			t.Errorf("Expected status 'done', got '%s'", status)
		}
		if !isComplete {
			t.Error("Expected isComplete=true")
		}
	})

	t.Run("maps PERCENT-COMPLETE 100 to done=true", func(t *testing.T) {
		status, isComplete := mapTodoStatus("", 100)
		if status != "done" {
			t.Errorf("Expected status 'done', got '%s'", status)
		}
		if !isComplete {
			t.Error("Expected isComplete=true")
		}
	})

	t.Run("maps NEEDS-ACTION to todo", func(t *testing.T) {
		status, isComplete := mapTodoStatus("NEEDS-ACTION", 0)
		if status != "todo" {
			t.Errorf("Expected status 'todo', got '%s'", status)
		}
		if isComplete {
			t.Error("Expected isComplete=false")
		}
	})

	t.Run("maps IN-PROCESS to in-progress", func(t *testing.T) {
		status, isComplete := mapTodoStatus("IN-PROCESS", 0)
		if status != "in-progress" {
			t.Errorf("Expected status 'in-progress', got '%s'", status)
		}
		if isComplete {
			t.Error("Expected isComplete=false")
		}
	})

	t.Run("maps CANCELLED to empty status (skip)", func(t *testing.T) {
		status, isComplete := mapTodoStatus("CANCELLED", 0)
		if status != "" {
			t.Errorf("Expected empty status for CANCELLED, got '%s'", status)
		}
		if isComplete {
			t.Error("Expected isComplete=false for CANCELLED")
		}
	})
}
