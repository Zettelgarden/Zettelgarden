package handlers

import (
	"fmt"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestSendMessageRoute_MessageValidation(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "openai/gpt-4o-mini", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	testCases := []struct {
		name           string
		payload        map[string]any
		expectedStatus int
		expectedError  string
	}{
		{
			name: "message too long",
			payload: map[string]any{
				"content": strings.Repeat("x", MaxMessageLength+1),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  fmt.Sprintf("message exceeds maximum length of %d characters", MaxMessageLength),
		},
		{
			name: "too many referenced cards",
			payload: map[string]any{
				"content":         "test message",
				"referenced_cards": []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}, // 11 cards
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  fmt.Sprintf("cannot reference more than %d cards", MaxReferencedCards),
		},
		{
			name: "invalid card ID format",
			payload: map[string]any{
				"content":         "test message",
				"referenced_cards": []string{"abc", "123"}, // "abc" is not numeric
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid card ID format: abc",
		},
		{
			name: "invalid model",
			payload: map[string]any{
				"content": "test message",
				"model":   "invalid-model",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid model: invalid-model",
		},
		{
			name: "valid message passes",
			payload: map[string]any{
				"content":         "valid test message",
				"referenced_cards": []string{"123"},
				"model":           "openai/gpt-4o-mini",
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			body := tests.CreateJsonBody(t, tt.payload)

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

			if rr.Code != tt.expectedStatus {
				t.Fatalf("wrong status: got %v, expected %v, body=%s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedError != "" {
				bodyStr := rr.Body.String()
				if !strings.Contains(bodyStr, tt.expectedError) {
					t.Fatalf("expected error '%s' in response, got: %s", tt.expectedError, bodyStr)
				}
			}
		})
	}
}

func TestStreamMessageRoute_MessageValidation(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "openai/gpt-4o-mini", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	testCases := []struct {
		name           string
		payload        map[string]any
		expectedStatus int
		expectedError  string
	}{
		{
			name: "message too long",
			payload: map[string]any{
				"content": strings.Repeat("x", MaxMessageLength+1),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  fmt.Sprintf("message exceeds maximum length of %d characters", MaxMessageLength),
		},
		{
			name: "too many referenced cards",
			payload: map[string]any{
				"content":         "test message",
				"referenced_cards": []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}, // 11 cards
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  fmt.Sprintf("cannot reference more than %d cards", MaxReferencedCards),
		},
		{
			name: "invalid card ID format",
			payload: map[string]any{
				"content":         "test message",
				"referenced_cards": []string{"abc", "123"}, // "abc" is not numeric
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid card ID format: abc",
		},
		{
			name: "invalid model",
			payload: map[string]any{
				"content": "test message",
				"model":   "invalid-model",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid model: invalid-model",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			body := tests.CreateJsonBody(t, tt.payload)

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

			if rr.Code != tt.expectedStatus {
				t.Fatalf("wrong status: got %v, expected %v, body=%s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedError != "" {
				bodyStr := rr.Body.String()
				if !strings.Contains(bodyStr, tt.expectedError) {
					t.Fatalf("expected error '%s' in response, got: %s", tt.expectedError, bodyStr)
				}
			}
		})
	}
}

func TestSendMessageRoute_QuotaExceeded(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "openai/gpt-4o-mini", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	// Exceed the quota by setting current usage to max limit
	quotaType := "messages_per_day"
	maxLimit := s.getDefaultQuotaLimit(quotaType)

	// Insert a quota record that is already at the limit
	query := `
		INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_DATE, NOW(), NOW())
		ON CONFLICT (user_id, quota_type, reset_date) DO UPDATE SET
			current_usage = $3,
			updated_at = NOW()
	`
	_, err = s.DB.Exec(query, 1, quotaType, maxLimit, maxLimit)
	if err != nil {
		t.Fatalf("Failed to set quota: %v", err)
	}

	payload := map[string]any{
		"content": "This should fail due to quota limit",
	}

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

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("wrong status: got %v, expected %v, body=%s", rr.Code, http.StatusTooManyRequests, rr.Body.String())
	}

	// Check for structured JSON error response
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "daily_message_limit_exceeded") {
		t.Fatalf("expected 'daily_message_limit_exceeded' in response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Daily message limit exceeded") {
		t.Fatalf("expected 'Daily message limit exceeded' in response, got: %s", bodyStr)
	}
}

func TestStreamMessageRoute_QuotaExceeded(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "openai/gpt-4o-mini", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	// Exceed the quota by setting current usage to max limit
	quotaType := "messages_per_day"
	maxLimit := s.getDefaultQuotaLimit(quotaType)

	// Insert a quota record that is already at the limit
	query := `
		INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_DATE, NOW(), NOW())
		ON CONFLICT (user_id, quota_type, reset_date) DO UPDATE SET
			current_usage = $3,
			updated_at = NOW()
	`
	_, err = s.DB.Exec(query, 1, quotaType, maxLimit, maxLimit)
	if err != nil {
		t.Fatalf("Failed to set quota: %v", err)
	}

	payload := map[string]any{
		"content": "This should fail due to quota limit",
	}

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

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("wrong status: got %v, expected %v, body=%s", rr.Code, http.StatusTooManyRequests, rr.Body.String())
	}

	// Check for structured JSON error response
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "daily_message_limit_exceeded") {
		t.Fatalf("expected 'daily_message_limit_exceeded' in response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Daily message limit exceeded") {
		t.Fatalf("expected 'Daily message limit exceeded' in response, got: %s", bodyStr)
	}
}

func TestRegenerateMessageRoute_QuotaExceeded(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	conv, err := s.CreateConversation(1, nil, "openai/gpt-4o-mini", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	// Exceed the quota by setting current usage to max limit
	quotaType := "messages_per_day"
	maxLimit := s.getDefaultQuotaLimit(quotaType)

	// Insert a quota record that is already at the limit
	query := `
		INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_DATE, NOW(), NOW())
		ON CONFLICT (user_id, quota_type, reset_date) DO UPDATE SET
			current_usage = $3,
			updated_at = NOW()
	`
	_, err = s.DB.Exec(query, 1, quotaType, maxLimit, maxLimit)
	if err != nil {
		t.Fatalf("Failed to set quota: %v", err)
	}

	// Create test messages - user message and assistant message
	_, err = s.SaveChatMessage(conv.ID, "user", stringPtr("Test user message"), nil, nil, nil, "completed")
	if err != nil {
		t.Fatalf("Failed to create user message: %v", err)
	}

	assistantMsg, err := s.SaveChatMessage(conv.ID, "assistant", stringPtr("Test assistant response"), nil, nil, nil, "completed")
	if err != nil {
		t.Fatalf("Failed to create assistant message: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/chat/conversations/"+conv.ID+"/messages/"+assistantMsg.ID+"/regenerate", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/chat/conversations/{id}/messages/{messageId}/regenerate", s.JwtMiddleware(s.RegenerateMessageRoute))
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("wrong status: got %v, expected %v, body=%s", rr.Code, http.StatusTooManyRequests, rr.Body.String())
	}

	// Check for structured JSON error response
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "daily_message_limit_exceeded") {
		t.Fatalf("expected 'daily_message_limit_exceeded' in response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Daily message limit exceeded") {
		t.Fatalf("expected 'Daily message limit exceeded' in response, got: %s", bodyStr)
	}
}

func TestIncrementChatUsageQuota(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 2 // Use different user to avoid conflicts
	quotaType := "messages_per_day"

	// First, check initial quota is 0
	initialQuotas, err := s.GetChatUsageQuotas(userID)
	if err != nil {
		t.Fatalf("GetChatUsageQuotas: %v", err)
	}

	var initialCount int
	for _, quota := range initialQuotas {
		if quota.QuotaType == quotaType {
			initialCount = quota.CurrentUsage
			break
		}
	}

	// Increment quota
	err = s.IncrementChatUsageQuota(userID, quotaType)
	if err != nil {
		t.Fatalf("IncrementChatUsageQuota: %v", err)
	}

	// Check quota was incremented
	finalQuotas, err := s.GetChatUsageQuotas(userID)
	if err != nil {
		t.Fatalf("GetChatUsageQuotas after increment: %v", err)
	}

	var finalCount int
	found := false
	for _, quota := range finalQuotas {
		if quota.QuotaType == quotaType {
			finalCount = quota.CurrentUsage
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("quota type %s not found after increment", quotaType)
	}

	expectedCount := initialCount + 1
	if finalCount != expectedCount {
		t.Fatalf("expected count %d, got %d", expectedCount, finalCount)
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}