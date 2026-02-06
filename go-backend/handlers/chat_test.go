package handlers

import (
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestIsToolResultEmpty(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected bool
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: true,
		},
		{
			name:     "empty map",
			result:   map[string]interface{}{},
			expected: true,
		},
		{
			name:     "only empty string",
			result:   map[string]interface{}{"data": ""},
			expected: true,
		},
		{
			name:     "only whitespace string",
			result:   map[string]interface{}{"data": "   "},
			expected: true,
		},
		{
			name:     "empty array",
			result:   map[string]interface{}{"data": []interface{}{}},
			expected: true,
		},
		{
			name:     "empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{}},
			expected: true,
		},
		{
			name:     "only error field",
			result:   map[string]interface{}{"error": "some error"},
			expected: true,
		},
		{
			name:     "non-empty string",
			result:   map[string]interface{}{"data": "some value"},
			expected: false,
		},
		{
			name:     "non-empty array",
			result:   map[string]interface{}{"data": []interface{}{"item"}},
			expected: false,
		},
		{
			name:     "non-empty nested map",
			result:   map[string]interface{}{"data": map[string]interface{}{"key": "value"}},
			expected: false,
		},
		{
			name:     "numeric value",
			result:   map[string]interface{}{"count": 42},
			expected: false,
		},
		{
			name:     "boolean value",
			result:   map[string]interface{}{"success": true},
			expected: false,
		},
		{
			name:     "mixed empty and non-empty",
			result:   map[string]interface{}{"empty": "", "data": "value"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolResultEmpty(tt.result)
			if result != tt.expected {
				t.Errorf("isToolResultEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestShouldClearToolResult(t *testing.T) {
	toolCallID := "call_123"
	userMessage := models.ChatMessage{Role: "user", Content: strPtr("Hello")}
	assistantMessage := models.ChatMessage{Role: "assistant", Content: strPtr("Hi")}
	toolMessage := models.ChatMessage{Role: "tool", ToolCallID: &toolCallID, Content: strPtr("Tool result")}

	tests := []struct {
		name          string
		msg           models.ChatMessage
		messageIndex  int
		totalMessages int
		expected      bool
	}{
		{
			name:          "user message should not be cleared",
			msg:           userMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "assistant message should not be cleared",
			msg:           assistantMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "recent tool message (last 10) should not be cleared",
			msg:           toolMessage,
			messageIndex:  15,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "tool message at exactly 10 from end should not be cleared",
			msg:           toolMessage,
			messageIndex:  10,
			totalMessages: 20,
			expected:      false,
		},
		{
			name:          "old tool message (>10 from end) should be cleared",
			msg:           toolMessage,
			messageIndex:  5,
			totalMessages: 20,
			expected:      true,
		},
		{
			name:          "very old tool message should be cleared",
			msg:           toolMessage,
			messageIndex:  0,
			totalMessages: 20,
			expected:      true,
		},
		{
			name:          "tool message in short conversation should not be cleared",
			msg:           toolMessage,
			messageIndex:  0,
			totalMessages: 8,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := services.NewMessageConverter()
			result := converter.ShouldClearToolResult(tt.msg, tt.messageIndex, tt.totalMessages)
			if result != tt.expected {
				t.Errorf("ShouldClearToolResult() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		messages []openai.ChatCompletionMessage
		minCount int // minimum expected tokens
		maxCount int // maximum expected tokens
	}{
		{
			name:     "empty messages",
			messages: []openai.ChatCompletionMessage{},
			minCount: 0,
			maxCount: 10,
		},
		{
			name: "single short message",
			messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: "Hello"},
			},
			minCount: 10,
			maxCount: 20,
		},
		{
			name: "multiple messages",
			messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "What is the weather?"},
				{Role: "assistant", Content: "I don't have access to weather information."},
			},
			minCount: 40,
			maxCount: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimateTokenCount(tt.messages)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("estimateTokenCount() = %v, expected between %v and %v", count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestGetModelContextLimit(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedLimit int
	}{
		{
			name:          "gemini flash",
			model:         "google/gemini-2.5-flash",
			expectedLimit: 1000000,
		},
		{
			name:          "gemini pro",
			model:         "google/gemini-2.5-pro",
			expectedLimit: 2000000,
		},
		{
			name:          "gpt-4o-mini",
			model:         "openai/gpt-4o-mini",
			expectedLimit: 128000,
		},
		{
			name:          "claude sonnet",
			model:         "anthropic/claude-sonnet-4",
			expectedLimit: 200000,
		},
		{
			name:          "unknown model",
			model:         "unknown/model",
			expectedLimit: 100000, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := getModelContextLimit(tt.model)
			if limit != tt.expectedLimit {
				t.Errorf("getModelContextLimit() = %v, expected %v", limit, tt.expectedLimit)
			}
		})
	}
}

// Integration Tests for Chat Service
// Test fixtures are imported from chat_test_fixtures.go
// These tests require database access and use the test transaction pattern

func TestChatServiceIntegration_BasicConversation(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("create and retrieve conversation", func(t *testing.T) {
		userID := 1
		title := "Test Conversation"

		conversation, err := h.CreateConversation(userID, &title, "google/gemini-2.5-flash", nil, nil)
		if err != nil {
			t.Fatalf("CreateConversation failed: %v", err)
		}

		if conversation.Title == nil || *conversation.Title != title {
			t.Errorf("Expected title %s, got %v", title, conversation.Title)
		}

		// Retrieve conversation
		retrieved, err := h.GetConversation(userID, conversation.ID)
		if err != nil {
			t.Fatalf("GetConversation failed: %v", err)
		}

		if retrieved.ID != conversation.ID {
			t.Errorf("Expected ID %s, got %s", conversation.ID, retrieved.ID)
		}
	})

	t.Run("save and retrieve message", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Message Test")

		content := "This is a test message"
		message, err := h.SaveChatMessage(conversation.ID, "user", &content, nil, nil, nil, "completed")
		if err != nil {
			t.Fatalf("SaveChatMessage failed: %v", err)
		}

		if message.Content == nil || *message.Content != content {
			t.Errorf("Expected content %s, got %v", content, message.Content)
		}

		if message.Role != "user" {
			t.Errorf("Expected role 'user', got %s", message.Role)
		}

		// Retrieve message
		retrieved, err := h.GetChatMessage(message.ID)
		if err != nil {
			t.Fatalf("GetChatMessage failed: %v", err)
		}

		if retrieved.ID != message.ID {
			t.Errorf("Expected ID %s, got %s", message.ID, retrieved.ID)
		}
	})

	t.Run("get conversation messages", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Get Messages Test")

		// Create multiple messages
		for i := 0; i < 3; i++ {
			role := "user"
			if i%2 == 1 {
				role = "assistant"
			}
			content := "Message " + string(rune('A'+i))
			_, err := h.SaveChatMessage(conversation.ID, role, &content, nil, nil, nil, "completed")
			if err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}
		}

		// Get all messages
		messages, err := h.GetConversationMessages(conversation.ID)
		if err != nil {
			t.Fatalf("GetConversationMessages failed: %v", err)
		}

		if len(messages) < 3 {
			t.Errorf("Expected at least 3 messages, got %d", len(messages))
		}
	})
}

func TestChatServiceIntegration_ErrorHandling(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("get non-existent conversation", func(t *testing.T) {
		_, err := h.GetConversation(1, "non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent conversation")
		}
	})

	t.Run("get non-existent message", func(t *testing.T) {
		_, err := h.GetChatMessage("non-existent-message-id")
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
	})

	t.Run("user cannot access another user's conversation", func(t *testing.T) {
		conv, err := h.CreateConversation(1, strPtr("Private Conversation"), "google/gemini-2.5-flash", nil, nil)
		if err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}

		_, err = h.GetConversation(2, conv.ID)
		if err == nil {
			t.Error("Expected error when user 2 tries to access user 1's conversation")
		}
	})
}

func TestChatServiceIntegration_UpdateOperations(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("update conversation title", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Update Test")

		newTitle := "Updated Title"
		err := h.UpdateConversationTitle(conversation.ID, newTitle)
		if err != nil {
			t.Fatalf("UpdateConversationTitle failed: %v", err)
		}

		updated, err := h.GetConversation(1, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation: %v", err)
		}

		if updated.Title == nil || *updated.Title != newTitle {
			t.Errorf("Expected title %s, got %v", newTitle, updated.Title)
		}
	})

	t.Run("update conversation model", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Model Update Test")

		newModel := "openai/gpt-4o-mini"
		err := h.UpdateConversationModel(conversation.ID, newModel)
		if err != nil {
			t.Fatalf("UpdateConversationModel failed: %v", err)
		}

		updated, err := h.GetConversation(1, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation: %v", err)
		}

		if updated.Model != newModel {
			t.Errorf("Expected model %s, got %s", newModel, updated.Model)
		}
	})

	t.Run("update message status", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Status Test")

		content := "Status test message"
		message, err := h.SaveChatMessage(conversation.ID, "assistant", &content, nil, nil, nil, "pending")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		// Update status to processing
		err = h.UpdateMessageStatus(message.ID, "processing")
		if err != nil {
			t.Fatalf("UpdateMessageStatus failed: %v", err)
		}

		// Verify status was updated
		updated, err := h.GetChatMessage(message.ID)
		if err != nil {
			t.Fatalf("Failed to get message: %v", err)
		}

		if updated.Status != "processing" {
			t.Errorf("Expected status 'processing', got %s", updated.Status)
		}
	})
}

func TestChatServiceIntegration_StarredConversations(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("star and unstar conversation", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Star Test")

		// Star the conversation
		err := h.UpdateConversationStarred(conversation.ID, true)
		if err != nil {
			t.Fatalf("UpdateConversationStarred failed: %v", err)
		}

		updated, err := h.GetConversation(1, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation: %v", err)
		}

		if !updated.Starred {
			t.Error("Expected conversation to be starred")
		}

		// Unstar the conversation
		err = h.UpdateConversationStarred(conversation.ID, false)
		if err != nil {
			t.Fatalf("Failed to unstar conversation: %v", err)
		}

		updated, err = h.GetConversation(1, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation: %v", err)
		}

		if updated.Starred {
			t.Error("Expected conversation to not be starred")
		}
	})
}

func TestChatServiceIntegration_Deletion(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("delete conversation", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Delete Test")

		// Add some messages
		for i := 0; i < 3; i++ {
			role := "user"
			if i%2 == 1 {
				role = "assistant"
			}
			content := "Message " + string(rune('A'+i))
			_, err := h.SaveChatMessage(conversation.ID, role, &content, nil, nil, nil, "completed")
			if err != nil {
				t.Fatalf("Failed to create message: %v", err)
			}
		}

		// Delete conversation
		err := h.DeleteConversation(conversation.ID)
		if err != nil {
			t.Fatalf("DeleteConversation failed: %v", err)
		}

		// Verify conversation is deleted
		_, err = h.GetConversation(1, conversation.ID)
		if err == nil {
			t.Error("Expected error when getting deleted conversation")
		}
	})
}

func TestChatServiceIntegration_ConversationWithCard(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("create conversation with primary card", func(t *testing.T) {
		card := CreateTestCard(t, h.GetDB(), 1, "Card for Chat", "This card is linked to a chat conversation.")

		conversation, err := h.CreateConversation(1, strPtr("Card Discussion"), "google/gemini-2.5-flash", nil, &card.ID)
		if err != nil {
			t.Fatalf("CreateConversation failed: %v", err)
		}

		if conversation.PrimaryCardID == nil {
			t.Error("Expected PrimaryCardID to be set")
		}

		if *conversation.PrimaryCardID != card.ID {
			t.Errorf("Expected PrimaryCardID %d, got %d", card.ID, *conversation.PrimaryCardID)
		}
	})
}

func TestChatServiceIntegration_MessageRetrieval(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("get messages up to specific message", func(t *testing.T) {
		conversation := CreateTestConversation(t, h.GetDB(), 1, "google/gemini-2.5-flash", "Messages Up To Test")

		// Create multiple messages
		var targetMessageID string
		for i := 0; i < 5; i++ {
			role := "user"
			if i%2 == 1 {
				role = "assistant"
			}
			content := "Message " + string(rune('A'+i))
			msg, err := h.SaveChatMessage(conversation.ID, role, &content, nil, nil, nil, "completed")
			if err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}

			// Mark the 3rd message as our target
			if i == 2 {
				targetMessageID = msg.ID
			}
		}

		// Get messages up to target
		messages, err := h.GetConversationMessagesUpTo(conversation.ID, targetMessageID)
		if err != nil {
			t.Fatalf("GetConversationMessagesUpTo failed: %v", err)
		}

		// Should have 3 messages (indices 0, 1, 2)
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages up to target, got %d", len(messages))
		}

		// Verify target is last
		if messages[len(messages)-1].ID != targetMessageID {
			t.Errorf("Last message should be target, got %s", messages[len(messages)-1].ID)
		}
	})
}

func TestChatServiceIntegration_ProFeatures(t *testing.T) {
	h := NewHandler()
	defer tests.Teardown()

	t.Run("user memory integration", func(t *testing.T) {
		userID := 1
		memory := "User prefers concise responses and examples in code blocks."
		CreateTestUserMemory(t, h.GetDB(), userID, memory)

		// Verify memory can be retrieved via tools
		registry := services.NewToolRegistry()

		conversation := CreateTestConversation(t, h.GetDB(), userID, "google/gemini-2.5-flash", "Memory Test")
		message := CreateTestMessage(t, h.GetDB(), conversation.ID, "user", "test", 1)

		ctx := &services.ToolContext{
			UserID:          userID,
			DB:              h.Server.DB,
			TypesenseClient: h.Server.TypesenseClient,
			ConversationID:  &conversation.ID,
			MessageID:       &message.ID,
			Model:           "google/gemini-2.5-flash",
		}

		result, err := registry.ExecuteTool("get_user_memory", map[string]interface{}{}, ctx)
		if err != nil {
			t.Fatalf("get_user_memory tool failed: %v", err)
		}

		retrievedMemory, ok := result["memory"].(string)
		if !ok {
			t.Fatal("Expected memory to be a string")
		}

		if retrievedMemory != memory {
			t.Errorf("Expected memory '%s', got '%s'", memory, retrievedMemory)
		}
	})
}

// Benchmark tests for performance tracking

func BenchmarkCreateConversation(b *testing.B) {
	h := NewHandler()
	defer tests.Teardown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		title := "Benchmark Test"
		_, err := h.CreateConversation(1, &title, "google/gemini-2.5-flash", nil, nil)
		if err != nil {
			b.Fatalf("CreateConversation failed: %v", err)
		}
	}
}

func BenchmarkSaveMessage(b *testing.B) {
	h := NewHandler()
	defer tests.Teardown()

	conversation := CreateTestConversation(&testing.T{}, h.GetDB(), 1, "google/gemini-2.5-flash", "Benchmark Message Test")
	content := "This is a benchmark test message content."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := h.SaveChatMessage(conversation.ID, "user", &content, nil, nil, nil, "completed")
		if err != nil {
			b.Fatalf("SaveChatMessage failed: %v", err)
		}
	}
}

func BenchmarkGetConversationMessages(b *testing.B) {
	h := NewHandler()
	defer tests.Teardown()

	conversation := CreateTestConversation(&testing.T{}, h.GetDB(), 1, "google/gemini-2.5-flash", "Benchmark Get Messages")

	// Create 50 messages
	for i := 0; i < 50; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		content := "Message number " + string(rune('A'+i%26))
		_, err := h.SaveChatMessage(conversation.ID, role, &content, nil, nil, nil, "completed")
		if err != nil {
			b.Fatalf("Failed to save message: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := h.GetConversationMessages(conversation.ID)
		if err != nil {
			b.Fatalf("GetConversationMessages failed: %v", err)
		}
	}
}