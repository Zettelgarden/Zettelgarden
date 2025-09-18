package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"go-backend/llms"
	"go-backend/models"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
)

// CreateConversationRequest represents the request to create a new conversation
type CreateConversationRequest struct {
	Title        *string `json:"title"`
	Model        string  `json:"model"`
	SystemPrompt *string `json:"system_prompt"`
}

// SendMessageRequest represents the request to send a message
type SendMessageRequest struct {
	Content string `json:"content"`
}

// ConversationResponse includes the conversation with message count
type ConversationResponse struct {
	models.ChatConversation
	MessageCount int `json:"message_count"`
}

// ConversationWithMessagesResponse includes conversation and its messages
type ConversationWithMessagesResponse struct {
	Conversation models.ChatConversation `json:"conversation"`
	Messages     []models.ChatMessage    `json:"messages"`
}

// CreateConversationRoute creates a new chat conversation
func (s *Handler) CreateConversationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req CreateConversationRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Set default model if not provided
	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}

	// Create conversation
	conversation, err := s.CreateConversation(userID, req.Title, req.Model, req.SystemPrompt)
	if err != nil {
		log.Printf("Error creating conversation: %v", err)
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// GetConversationsRoute lists user's conversations
func (s *Handler) GetConversationsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	conversations, err := s.GetUserConversations(userID)
	if err != nil {
		log.Printf("Error getting conversations: %v", err)
		http.Error(w, "Failed to get conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

// GetConversationRoute gets a specific conversation with its messages
func (s *Handler) GetConversationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	// Get conversation
	conversation, err := s.GetConversation(userID, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting conversation: %v", err)
			http.Error(w, "Failed to get conversation", http.StatusInternalServerError)
		}
		return
	}

	// Get messages
	messages, err := s.GetConversationMessages(conversationID)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	response := ConversationWithMessagesResponse{
		Conversation: *conversation,
		Messages:     messages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SendMessageRoute sends a message and gets LLM response
func (s *Handler) SendMessageRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	var req SendMessageRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	// Verify conversation exists and belongs to user
	conversation, err := s.GetConversation(userID, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting conversation: %v", err)
			http.Error(w, "Failed to get conversation", http.StatusInternalServerError)
		}
		return
	}

	// Check usage quotas
	if err := s.CheckChatUsageQuota(userID, "messages_per_day"); err != nil {
		http.Error(w, "Daily message limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Save user message
	userMessage, err := s.SaveChatMessage(conversationID, "user", &req.Content, nil, nil)
	if err != nil {
		log.Printf("Error saving user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Get conversation history for LLM
	messages, err := s.GetConversationMessages(conversationID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		http.Error(w, "Failed to get conversation history", http.StatusInternalServerError)
		return
	}

	// Generate LLM response with tools
	assistantMessage, err := s.GenerateChatResponse(userID, conversation, messages)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	// Update usage quota
	s.IncrementChatUsageQuota(userID, "messages_per_day")

	// Return both messages
	response := []models.ChatMessage{*userMessage, *assistantMessage}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteConversationRoute deletes a conversation
func (s *Handler) DeleteConversationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	// Verify conversation exists and belongs to user
	_, err := s.GetConversation(userID, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting conversation: %v", err)
			http.Error(w, "Failed to get conversation", http.StatusInternalServerError)
		}
		return
	}

	// Delete conversation (cascade will handle messages)
	err = s.DeleteConversation(conversationID)
	if err != nil {
		log.Printf("Error deleting conversation: %v", err)
		http.Error(w, "Failed to delete conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StarConversationRoute toggles starred status of a conversation
func (s *Handler) StarConversationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	// Verify conversation exists and belongs to user
	conversation, err := s.GetConversation(userID, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting conversation: %v", err)
			http.Error(w, "Failed to get conversation", http.StatusInternalServerError)
		}
		return
	}

	// Toggle starred status
	newStarred := !conversation.Starred
	err = s.UpdateConversationStarred(conversationID, newStarred)
	if err != nil {
		log.Printf("Error updating conversation starred status: %v", err)
		http.Error(w, "Failed to update conversation", http.StatusInternalServerError)
		return
	}

	conversation.Starred = newStarred
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// GetUsageQuotaRoute gets user's usage quota status
func (s *Handler) GetUsageQuotaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	quotas, err := s.GetChatUsageQuotas(userID)
	if err != nil {
		log.Printf("Error getting usage quotas: %v", err)
		http.Error(w, "Failed to get usage quotas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotas)
}

// Database methods for chat functionality

// CreateConversation creates a new chat conversation
func (s *Handler) CreateConversation(userID int, title *string, model string, systemPrompt *string) (*models.ChatConversation, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO chat_conversations (id, user_id, title, model, system_prompt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, user_id, title, model, system_prompt, starred, created_at, updated_at
	`

	var conversation models.ChatConversation
	err := s.DB.QueryRow(query, id, userID, title, model, systemPrompt).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Model,
		&conversation.SystemPrompt,
		&conversation.Starred,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	return &conversation, err
}

// GetUserConversations gets all conversations for a user
func (s *Handler) GetUserConversations(userID int) ([]ConversationResponse, error) {
	query := `
		SELECT c.id, c.user_id, c.title, c.model, c.system_prompt, c.starred,
		       c.created_at, c.updated_at, COUNT(m.id) as message_count
		FROM chat_conversations c
		LEFT JOIN chat_messages m ON c.id = m.conversation_id
		WHERE c.user_id = $1
		GROUP BY c.id, c.user_id, c.title, c.model, c.system_prompt, c.starred, c.created_at, c.updated_at
		ORDER BY c.updated_at DESC
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []ConversationResponse
	for rows.Next() {
		var conv ConversationResponse
		err := rows.Scan(
			&conv.ID,
			&conv.UserID,
			&conv.Title,
			&conv.Model,
			&conv.SystemPrompt,
			&conv.Starred,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&conv.MessageCount,
		)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetConversation gets a specific conversation
func (s *Handler) GetConversation(userID int, conversationID string) (*models.ChatConversation, error) {
	query := `
		SELECT id, user_id, title, model, system_prompt, starred, created_at, updated_at
		FROM chat_conversations
		WHERE id = $1 AND user_id = $2
	`

	var conversation models.ChatConversation
	err := s.DB.QueryRow(query, conversationID, userID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Model,
		&conversation.SystemPrompt,
		&conversation.Starred,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	return &conversation, err
}

// GetConversationMessages gets all messages for a conversation
func (s *Handler) GetConversationMessages(conversationID string) ([]models.ChatMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, created_at
		FROM chat_messages
		WHERE conversation_id = $1
		ORDER BY sequence_number ASC
	`

	rows, err := s.DB.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var msg models.ChatMessage
		var toolCalls *string

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&toolCalls,
			&msg.ToolCallID,
			&msg.SequenceNumber,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse tool calls JSON
		if toolCalls != nil && *toolCalls != "" {
			if err := json.Unmarshal([]byte(*toolCalls), &msg.ToolCalls); err != nil {
				log.Printf("Error parsing tool calls: %v", err)
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveChatMessage saves a chat message
func (s *Handler) SaveChatMessage(conversationID, role string, content *string, toolCalls []models.ChatToolCall, toolCallID *string) (*models.ChatMessage, error) {
	// Get next sequence number
	var sequenceNumber int
	err := s.DB.QueryRow("SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM chat_messages WHERE conversation_id = $1", conversationID).Scan(&sequenceNumber)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()

	// Convert tool calls to JSON - handle JSONB properly
	var toolCallsJSON *string
	if toolCalls != nil && len(toolCalls) > 0 {
		toolCallsData, err := json.Marshal(toolCalls)
		if err != nil {
			return nil, err
		}
		toolCallsStr := string(toolCallsData)
		toolCallsJSON = &toolCallsStr
	} else {
		toolCallsJSON = nil
	}

	query := `
		INSERT INTO chat_messages (id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, created_at
	`

	var message models.ChatMessage
	var returnedToolCalls *string

	err = s.DB.QueryRow(query, id, conversationID, role, content, toolCallsJSON, toolCallID, sequenceNumber).Scan(
		&message.ID,
		&message.ConversationID,
		&message.Role,
		&message.Content,
		&returnedToolCalls,
		&message.ToolCallID,
		&message.SequenceNumber,
		&message.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse returned tool calls
	if returnedToolCalls != nil && *returnedToolCalls != "" {
		if err := json.Unmarshal([]byte(*returnedToolCalls), &message.ToolCalls); err != nil {
			log.Printf("Error parsing returned tool calls: %v", err)
		}
	}

	return &message, nil
}

// DeleteConversation deletes a conversation
func (s *Handler) DeleteConversation(conversationID string) error {
	query := `DELETE FROM chat_conversations WHERE id = $1`
	_, err := s.DB.Exec(query, conversationID)
	return err
}

// UpdateConversationStarred updates the starred status of a conversation
func (s *Handler) UpdateConversationStarred(conversationID string, starred bool) error {
	query := `UPDATE chat_conversations SET starred = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.DB.Exec(query, starred, conversationID)
	return err
}

// GenerateChatResponse generates an LLM response with tool calling support
func (s *Handler) GenerateChatResponse(userID int, conversation *models.ChatConversation, messages []models.ChatMessage) (*models.ChatMessage, error) {
	// Import the tools package (we'll need this in the imports)
	// Convert messages to OpenAI format
	var openaiMessages []openai.ChatCompletionMessage

	log.Printf("Starting prompting")
	systemPrompt := `
	You are the Research Assistant for a Zettelkasten knowledge base.  
Your role is to help the user explore, retrieve, and synthesize information across their cards.  
You can interact with the knowledge base directly, but for complex or exploratory tasks you should delegate work to specialized subagents using the 'Task' tool.

### Core Behaviors:
- Always aim to preserve conversation flow with the user.  
- You likely should not be calling many tools except for Task. Let the subagents do the work.
- When a user request involves **searching, multiple queries, uncertain directions, or research across many cards**, break the problem down into subtasks and launch one or more subagents using the 'Task' tool.  
- Think step-by-step: consider whether you’d benefit from launching subtasks before trying to answer directly.  
- Only use knowledge base tools directly when the operation is **simple and direct** (e.g., fetching a single known card by ID).  

### Subtasks & Subagents:
- Use the 'Task' tool to launch a subagent for:  
  - research queries such as "find me cards about..."
  - Searches requiring semantic exploration or card hierarchy traversal  
  - Filtering and analyzing results across many cards  
  - Gathering supporting evidence before synthesizing an answer  
- Prefer spawning **more than one subtask** if distinct branches of exploration are possible. For example: “search one way by tag, another by semantic similarity.”

Available Subagent:
- 'general-purpose': General research, searching, and multi-step exploration.

### Knowledge Base Tools:
- 'Task': Launch a subagent for multi-step research tasks  
- 'search_cards': Text/semantic search for cards  
- 'get_card_by_id': Retrieve a card by exact ID  
- 'browse_card_hierarchy': Navigate parent/child/card relationships  
- 'filter_cards_by_metadata': Filter cards by tags, stars, or dates  

### Responding to the User:
- Always answer naturally and clearly in plain language first.  
- When referencing **cards** in your response:
  - If you mention **2 or more cards**, or include detailed card information, also provide a structured JSON block at the end of your answer.  
  - The JSON must use **exactly** the schema returned by the knowledge base tools.  
  - Do **not** invent fields—only include what the tools provide.  

### JSON Card Block Format:
---CARDS---
` +
		"```" + `json
{
  "cards": [
    {
      "id": 123,
      "card_id": "2.54.1",
      "title": "AI Research Project",
      "body_preview": "This project focuses on...",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-16T14:20:00Z",
      "tags": ["ai", "research", "project"]
    }
  ]
}
  ` +
		"```" + `

### Error & Fallbacks:
- If a subagent fails or gives no useful results, explain this briefly and suggest next steps.  
- If the user request is ambiguous, clarify it or launch parallel subtasks to cover different interpretations.  

Remember:  
- **Decompose first.** If the problem can be broken down, launch subtasks.  
- **Respond clearly.** Use JSON only when needed.  
- **Preserve context.** The conversation should feel continuous even while research is delegated.  
`

	openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	// Add conversation messages
	for _, msg := range messages {
		var content string
		if msg.Content != nil {
			content = *msg.Content
		}

		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			var toolCalls []openai.ToolCall
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// Handle tool call responses
		if msg.ToolCallID != nil {
			openaiMsg.ToolCallID = *msg.ToolCallID
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Create LLM client
	client := llms.NewDefaultClient(s.DB, userID)
	client.Model = conversation.Model

	// Get tools registry
	toolRegistry := llms.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	// Loop until no more tool calls are needed
	for {
		// Make request with tools
		resp, err := client.Client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    conversation.Model,
				Messages: openaiMessages,
				Tools:    tools,
			},
		)

		if err != nil {
			return nil, err
		}

		assistantMessage := resp.Choices[0].Message

		// If no tool calls, save the final response and return
		if len(assistantMessage.ToolCalls) == 0 {
			return s.SaveChatMessage(conversation.ID, "assistant", &assistantMessage.Content, nil, nil)
		}

		// Save assistant message with tool calls
		var toolCalls []models.ChatToolCall
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			toolCalls = append(toolCalls, models.ChatToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: models.ChatToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			})
		}

		assistantMsg, err := s.SaveChatMessage(conversation.ID, "assistant", &assistantMessage.Content, toolCalls, nil)
		if err != nil {
			return nil, err
		}

		// Execute tool calls and save tool responses
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			result, err := toolRegistry.ExecuteTool(tc.Function.Name, args, userID, s.DB, s.Server.TypesenseClient, &conversation.ID, &assistantMsg.ID)
			if err != nil {
				log.Printf("Error executing tool %s: %v", tc.Function.Name, err)
				result = map[string]interface{}{
					"error": err.Error(),
				}
			}

			// Convert result to JSON string for tool response
			resultJSON, _ := json.Marshal(result)
			resultStr := string(resultJSON)

			// Save tool response message
			_, err = s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID)
			if err != nil {
				log.Printf("Error saving tool response: %v", err)
			}
		}

		// Get updated messages including tool responses for next iteration
		updatedMessages, err := s.GetConversationMessages(conversation.ID)
		if err != nil {
			return nil, err
		}

		// Convert to OpenAI format for next request
		openaiMessages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		}

		for _, msg := range updatedMessages {
			var content string
			if msg.Content != nil {
				content = *msg.Content
			}

			openaiMsg := openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: content,
			}

			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ToolCall
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Function.Arguments)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   tc.ID,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: string(argsJSON),
						},
					})
				}
				openaiMsg.ToolCalls = toolCalls
			}

			if msg.ToolCallID != nil {
				openaiMsg.ToolCallID = *msg.ToolCallID
			}

			openaiMessages = append(openaiMessages, openaiMsg)
		}
	}
}

// Usage quota methods

// CheckChatUsageQuota checks if user has exceeded their quota
func (s *Handler) CheckChatUsageQuota(userID int, quotaType string) error {
	return nil
	// quota, err := s.getChatUsageQuota(userID, quotaType)
	// if err != nil {
	// 	// If no quota exists, create default quotas
	// 	if err == sql.ErrNoRows {
	// 		err = s.initializeDefaultQuotas(userID)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		quota, err = s.getChatUsageQuota(userID, quotaType)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	} else {
	// 		return err
	// 	}
	// }

	// // Check if quota is exceeded
	// if quota.CurrentUsage >= quota.MaxLimit {
	// 	return fmt.Errorf("quota exceeded for %s", quotaType)
	// }

	// return nil
}

// IncrementChatUsageQuota increments the usage counter
func (s *Handler) IncrementChatUsageQuota(userID int, quotaType string) error {
	query := `
		INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date)
		VALUES ($1, $2, 1, $3, CURRENT_DATE)
		ON CONFLICT (user_id, quota_type, reset_date)
		DO UPDATE SET
			current_usage = chat_usage_quotas.current_usage + 1,
			updated_at = NOW()
	`

	maxLimit := s.getDefaultQuotaLimit(quotaType)
	_, err := s.DB.Exec(query, userID, quotaType, maxLimit)
	return err
}

// GetChatUsageQuotas gets all usage quotas for a user
func (s *Handler) GetChatUsageQuotas(userID int) ([]models.ChatUsageQuota, error) {
	query := `
		SELECT id, user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at
		FROM chat_usage_quotas
		WHERE user_id = $1 AND reset_date = CURRENT_DATE
		ORDER BY quota_type
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotas []models.ChatUsageQuota
	for rows.Next() {
		var quota models.ChatUsageQuota
		err := rows.Scan(
			&quota.ID,
			&quota.UserID,
			&quota.QuotaType,
			&quota.CurrentUsage,
			&quota.MaxLimit,
			&quota.ResetDate,
			&quota.CreatedAt,
			&quota.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quotas = append(quotas, quota)
	}

	return quotas, nil
}

// Helper methods

func (s *Handler) getChatUsageQuota(userID int, quotaType string) (*models.ChatUsageQuota, error) {
	query := `
		SELECT id, user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at
		FROM chat_usage_quotas
		WHERE user_id = $1 AND quota_type = $2 AND reset_date = CURRENT_DATE
	`

	var quota models.ChatUsageQuota
	err := s.DB.QueryRow(query, userID, quotaType).Scan(
		&quota.ID,
		&quota.UserID,
		&quota.QuotaType,
		&quota.CurrentUsage,
		&quota.MaxLimit,
		&quota.ResetDate,
		&quota.CreatedAt,
		&quota.UpdatedAt,
	)

	return &quota, err
}

func (s *Handler) initializeDefaultQuotas(userID int) error {
	quotas := []struct {
		quotaType string
		maxLimit  int
	}{
		{"messages_per_day", s.getDefaultQuotaLimit("messages_per_day")},
		{"tool_calls_per_day", s.getDefaultQuotaLimit("tool_calls_per_day")},
		{"conversations_per_day", s.getDefaultQuotaLimit("conversations_per_day")},
	}

	for _, quota := range quotas {
		query := `
			INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date)
			VALUES ($1, $2, 0, $3, CURRENT_DATE)
			ON CONFLICT (user_id, quota_type, reset_date) DO NOTHING
		`
		_, err := s.DB.Exec(query, userID, quota.quotaType, quota.maxLimit)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Handler) getDefaultQuotaLimit(quotaType string) int {
	// TODO: Check user subscription level for PRO limits
	switch quotaType {
	case "messages_per_day":
		return 50 // Free tier limit
	case "tool_calls_per_day":
		return 100 // Free tier limit
	case "conversations_per_day":
		return 10 // Free tier limit
	default:
		return 10
	}
}
