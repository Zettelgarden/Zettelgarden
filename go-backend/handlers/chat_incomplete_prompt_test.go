package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestLooksLikeIncompleteUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		expect  bool
	}{
		{name: "short wh-word ending", input: "hello how", expect: true},
		{name: "single conjunction", input: "so", expect: true},
		{name: "capitalized wh-word", input: "What", expect: true},
		{name: "trailing comma", input: "hello,", expect: true},
		{name: "question mark", input: "hello how are you?", expect: false},
		{name: "period", input: "hello.", expect: false},
		{name: "long message", input: strings.Repeat("a", 45), expect: false},
		{name: "normal short question", input: "summarize card 123", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeIncompleteUserMessage(tt.input)
			if got != tt.expect {
				t.Fatalf("looksLikeIncompleteUserMessage(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestSendMessageRoute_IncompletePromptReturnsClarification(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "google/gemini-2.5-flash", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	payload := map[string]any{"content": "hello how"}
	body := tests.CreateJsonBody(t, payload)

	req, err := http.NewRequest("POST", "/api/chat/conversations/"+conv.ID+"/messages", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/chat/conversations/{id}/messages", s.JwtMiddleware(s.SendMessageRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status: got %v, body=%s", rr.Code, rr.Body.String())
	}

	var msgs []models.ChatMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &msgs); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	assistant := msgs[1]
	if assistant.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", assistant.Role)
	}
	if assistant.Status != "completed" {
		t.Fatalf("expected assistant status completed, got %q", assistant.Status)
	}
	if assistant.Content == nil || strings.TrimSpace(*assistant.Content) == "" {
		t.Fatalf("expected non-empty assistant content")
	}
	if !strings.Contains(strings.ToLower(*assistant.Content), "cut off") {
		t.Fatalf("expected clarification mentioning cut off, got %q", *assistant.Content)
	}
}

func TestStreamMessageRoute_IncompletePromptReturnsImmediateDone(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "google/gemini-2.5-flash", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	payload := map[string]any{"content": "hello how"}
	body := tests.CreateJsonBody(t, payload)

	req, err := http.NewRequest("POST", "/api/chat/conversations/"+conv.ID+"/messages/stream", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/chat/conversations/{id}/messages/stream", s.JwtMiddleware(s.StreamMessageRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status: got %v, body=%s", rr.Code, rr.Body.String())
	}

	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "event: messages") {
		t.Fatalf("expected messages event, body=%q", bodyStr)
	}
	if !strings.Contains(bodyStr, "event: done") {
		t.Fatalf("expected done event, body=%q", bodyStr)
	}

	data := extractSSEEventData(t, bodyStr, "messages")
	var event struct {
		UserMessage      models.ChatMessage `json:"user_message"`
		AssistantMessage models.ChatMessage `json:"assistant_message"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("failed to parse messages event JSON: %v (data=%q)", err, data)
	}

	if event.AssistantMessage.Status != "completed" {
		t.Fatalf("expected assistant status completed, got %q", event.AssistantMessage.Status)
	}
	if event.AssistantMessage.Content == nil || strings.TrimSpace(*event.AssistantMessage.Content) == "" {
		t.Fatalf("expected non-empty assistant content")
	}
	if !strings.Contains(strings.ToLower(*event.AssistantMessage.Content), "cut off") {
		t.Fatalf("expected clarification mentioning cut off, got %q", *event.AssistantMessage.Content)
	}
}

func extractSSEEventData(t *testing.T, body string, eventName string) string {
	t.Helper()
	marker := "event: " + eventName + "\ndata: "
	idx := strings.Index(body, marker)
	if idx == -1 {
		t.Fatalf("event %q not found", eventName)
	}
	start := idx + len(marker)
	rest := body[start:]
	end := strings.Index(rest, "\n\n")
	if end == -1 {
		t.Fatalf("event %q data terminator not found", eventName)
	}
	return rest[:end]
}
