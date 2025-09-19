package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	Content         string   `json:"content"`
	ReferencedCards []string `json:"referenced_cards,omitempty"`
	Model           *string  `json:"model,omitempty"`
}

// UpdateConversationTitleRequest represents the request to update a conversation title
type UpdateConversationTitleRequest struct {
	Title string `json:"title"`
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

	// Check if user has subscription for chat functionality
	if !s.UserHasSubscription(userID) {
		http.Error(w, "Chat functionality requires a Pro subscription", http.StatusForbidden)
		return
	}

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

func (s *Handler) GetReferencedCards(userID int, cardIDs []string) string {
	// Fetch referenced card data if any cards are referenced
	referencedCardsContext := ""
	if len(cardIDs) > 0 {
		// Remove duplicates
		cardIDSet := make(map[string]bool)
		var uniqueCardIDs []string
		for _, cardID := range cardIDs {
			if !cardIDSet[cardID] {
				cardIDSet[cardID] = true
				uniqueCardIDs = append(uniqueCardIDs, cardID)
			}
		}

		var cards []models.Card
		for _, idString := range uniqueCardIDs {
			id, err := strconv.Atoi(idString)
			if err != nil {
				continue
			}

			card, err := s.QueryFullCard(userID, id)
			if err != nil {
				continue
			}
			cards = append(cards, card)

		}
		if len(cards) > 0 {
			var cardContexts []string
			for _, card := range cards {
				cardContext := fmt.Sprintf("ID (primary key): %v\nCard ID: %s\nTitle: %s\nContent:\n%s",
					card.ID, card.CardID, card.Title, card.Body)
				cardContexts = append(cardContexts, cardContext)
			}
			referencedCardsContext = "\n\n<referenced cards>\n" + strings.Join(cardContexts, "\n\n---\n")
			referencedCardsContext += "</referenced cards>"
		}
	}
	return referencedCardsContext

}

// SendMessageRoute sends a message and gets LLM response
func (s *Handler) SendMessageRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	// Check if user has subscription for chat functionality
	if !s.UserHasSubscription(userID) {
		http.Error(w, "Chat functionality requires a Pro subscription", http.StatusForbidden)
		return
	}

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

	referencedCardContext := s.GetReferencedCards(userID, req.ReferencedCards)

	content := req.Content + referencedCardContext
	// Save user message
	userMessage, err := s.SaveChatMessage(conversationID, "user", &content, nil, nil, req.ReferencedCards)
	if err != nil {
		log.Printf("Error saving user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Generate title if this is the first message (and conversation has no title)
	if conversation.Title == nil || *conversation.Title == "" {
		generatedTitle := s.generateConversationTitle(userID, req.Content)
		err := s.UpdateConversationTitle(conversationID, generatedTitle)
		if err != nil {
			log.Printf("Error updating conversation title: %v", err)
			// Don't fail the request if title update fails, just log it
		}
	}

	// Get conversation history for LLM
	messages, err := s.GetConversationMessages(conversationID)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		http.Error(w, "Failed to get conversation history", http.StatusInternalServerError)
		return
	}

	// Determine which model to use - request override or conversation default
	modelToUse := conversation.Model
	if req.Model != nil && *req.Model != "" {
		modelToUse = *req.Model
	}

	// Generate LLM response with tools
	assistantMessage, err := s.GenerateChatResponse(userID, conversation, messages, modelToUse)
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

// UpdateConversationTitleRoute updates the title of a conversation
func (s *Handler) UpdateConversationTitleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]

	var req UpdateConversationTitleRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate title
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if len(req.Title) > 100 {
		http.Error(w, "Title too long (max 100 characters)", http.StatusBadRequest)
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

	// Update conversation title
	err = s.UpdateConversationTitle(conversationID, req.Title)
	if err != nil {
		log.Printf("Error updating conversation title: %v", err)
		http.Error(w, "Failed to update conversation title", http.StatusInternalServerError)
		return
	}

	// Update the conversation object with new title
	conversation.Title = &req.Title
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
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, created_at
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
		var referencedCards *string

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&toolCalls,
			&msg.ToolCallID,
			&msg.SequenceNumber,
			&referencedCards,
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

		// Parse referenced cards JSON
		if referencedCards != nil && *referencedCards != "" {
			if err := json.Unmarshal([]byte(*referencedCards), &msg.ReferencedCards); err != nil {
				log.Printf("Error parsing referenced cards: %v", err)
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveChatMessage saves a chat message
func (s *Handler) SaveChatMessage(conversationID, role string, content *string, toolCalls []models.ChatToolCall, toolCallID *string, referencedCards []string) (*models.ChatMessage, error) {
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

	// Convert referenced cards to JSON
	var referencedCardsJSON *string
	if referencedCards != nil && len(referencedCards) > 0 {
		referencedCardsData, err := json.Marshal(referencedCards)
		if err != nil {
			return nil, err
		}
		referencedCardsStr := string(referencedCardsData)
		referencedCardsJSON = &referencedCardsStr
	} else {
		referencedCardsJSON = nil
	}

	query := `
		INSERT INTO chat_messages (id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, created_at
	`

	var message models.ChatMessage
	var returnedToolCalls *string
	var returnedReferencedCards *string

	err = s.DB.QueryRow(query, id, conversationID, role, content, toolCallsJSON, toolCallID, sequenceNumber, referencedCardsJSON).Scan(
		&message.ID,
		&message.ConversationID,
		&message.Role,
		&message.Content,
		&returnedToolCalls,
		&message.ToolCallID,
		&message.SequenceNumber,
		&returnedReferencedCards,
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

	// Parse returned referenced cards
	if returnedReferencedCards != nil && *returnedReferencedCards != "" {
		if err := json.Unmarshal([]byte(*returnedReferencedCards), &message.ReferencedCards); err != nil {
			log.Printf("Error parsing returned referenced cards: %v", err)
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

// UpdateConversationTitle updates the title of a conversation
func (s *Handler) UpdateConversationTitle(conversationID string, title string) error {
	query := `UPDATE chat_conversations SET title = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.DB.Exec(query, title, conversationID)
	return err
}

// GenerateChatResponse generates an LLM response with tool calling support
func (s *Handler) GenerateChatResponse(userID int, conversation *models.ChatConversation, messages []models.ChatMessage, model string) (*models.ChatMessage, error) {
	// Import the tools package (we'll need this in the imports)
	// Convert messages to OpenAI format
	var openaiMessages []openai.ChatCompletionMessage

	log.Printf("Starting prompting")
	systemPrompt, err := prompts.GetResearchAssistantPrompt()
	if err != nil {
		log.Printf("Error loading system prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		systemPrompt = "You are the Research Assistant for a Zettelkasten knowledge base. Help users explore and synthesize information across their cards."
	}

	// Collect all referenced card IDs from user messages
	var allReferencedCardIDs []string
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.ReferencedCards) > 0 {
			allReferencedCardIDs = append(allReferencedCardIDs, msg.ReferencedCards...)
		}
	}

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
	client := services.NewDefaultClient(s.DB, userID)
	client.Model = model
	client.RequestType = "chat"

	// Get tools registry
	toolRegistry := services.NewToolRegistry()
	tools := toolRegistry.GetToolDefinitions()

	// Loop until no more tool calls are needed
	for {
		resp, err := services.ExecuteLLMToolRequest(client, openaiMessages, tools)

		if err != nil {
			log.Printf("executing LLM error: %v", err)
			return nil, err
		}

		assistantMessage := resp.Choices[0].Message

		// If no tool calls, save the final response and return
		if len(assistantMessage.ToolCalls) == 0 {
			return s.SaveChatMessage(conversation.ID, "assistant", &assistantMessage.Content, nil, nil, nil)
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

		assistantMsg, err := s.SaveChatMessage(conversation.ID, "assistant", &assistantMessage.Content, toolCalls, nil, nil)
		if err != nil {
			return nil, err
		}

		// Execute tool calls and save tool responses
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			ctx := &services.ToolContext{
				UserID:          userID,
				DB:              s.DB,
				TypesenseClient: s.Server.TypesenseClient,
				ConversationID:  &conversation.ID,
				MessageID:       &assistantMsg.ID,
				Model:           model,
			}
			result, err := toolRegistry.ExecuteTool(tc.Function.Name, args, ctx)
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
			_, err = s.SaveChatMessage(conversation.ID, "tool", &resultStr, nil, &tc.ID, nil)
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
		// Load system prompt again for consistency
		currentSystemPrompt, promptErr := prompts.GetResearchAssistantPrompt()
		if promptErr != nil {
			log.Printf("Error reloading system prompt: %v, using previous", promptErr)
			currentSystemPrompt = systemPrompt // Use the previously loaded prompt
		}

		openaiMessages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: currentSystemPrompt,
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

// GetCardsByCardIDs retrieves card data for the given card IDs
func (s *Handler) GetCardsByCardIDs(userID int, cardIDs []string) ([]models.Card, error) {
	if len(cardIDs) == 0 {
		return []models.Card{}, nil
	}

	// Create placeholders for the IN clause
	placeholders := make([]string, len(cardIDs))
	args := make([]interface{}, len(cardIDs)+1)
	args[0] = userID

	for i, cardID := range cardIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = cardID
	}

	query := fmt.Sprintf(`
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND id IN (%s) AND is_deleted = FALSE
		ORDER BY created_at DESC
	`, strings.Join(placeholders, ", "))

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var card models.Card
		err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Body,
			&card.Link,
			&card.ParentID,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}

	return cards, nil
}

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

// generateConversationTitle generates a title for a conversation based on the user's first message
func (s *Handler) generateConversationTitle(userID int, userMessage string) string {
	// Load title generation prompt
	titlePrompt, err := prompts.GetTitleGeneratorPrompt()
	if err != nil {
		log.Printf("Error loading title generation prompt: %v", err)
		// Fallback to truncated user message
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	// Create LLM client for title generation
	client := services.NewDefaultClient(s.DB, userID) // Use system user ID for title generation
	client.RequestType = "chat"
	client.Model = "google/gemini-2.5-flash-lite" // Use faster, cheaper model for title generation

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: titlePrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		},
	}

	resp, err := services.ExecuteLLMRequest(client, messages)

	if err != nil {
		log.Printf("Error generating conversation title: %v", err)
		// Fallback to truncated user message
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Ensure title isn't too long
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	// If title generation failed or returned empty, use fallback
	if title == "" {
		if len(userMessage) > 40 {
			return userMessage[:40] + "..."
		}
		return userMessage
	}

	return title
}
