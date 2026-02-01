package services

import (
	"net/http"
	"net/http/httptest"
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
