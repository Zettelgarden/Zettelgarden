// Package chat provides HTTP handlers for chat endpoints.
// It bridges HTTP requests with the ChatService for handling AI agent conversations.
package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/server"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for chat endpoints.
// It maintains references to the server and chat service for routing requests.
type Handler struct {
	server       *server.Server
	chatService  *Service
}

// NewHandler creates a new chat HTTP handler.
func NewHandler(srv *server.Server, chatService *Service) *Handler {
	return &Handler{
		server:      srv,
		chatService: chatService,
	}
}

// RegisterRoutes registers all chat-related HTTP routes.
func RegisterRoutes(router *mux.Router, handler *Handler) {
	// Chat conversation routes
	router.HandleFunc("/api/chat/conversations", handler.requireAuth(handler.CreateConversation)).Methods("POST")
	router.HandleFunc("/api/chat/conversations", handler.requireAuth(handler.ListConversations)).Methods("GET")
	router.HandleFunc("/api/chat/conversations/{id}", handler.requireAuth(handler.GetConversation)).Methods("GET")
	router.HandleFunc("/api/chat/conversations/{id}", handler.requireAuth(handler.UpdateConversation)).Methods("PATCH")
	router.HandleFunc("/api/chat/conversations/{id}", handler.requireAuth(handler.DeleteConversation)).Methods("DELETE")

	// Chat message routes
	router.HandleFunc("/api/chat/conversations/{id}/messages", handler.requireAuth(handler.SendMessage)).Methods("POST")
	router.HandleFunc("/api/chat/conversations/{id}/messages", handler.requireAuth(handler.GetMessages)).Methods("GET")
	router.HandleFunc("/api/chat/conversations/{id}/stream", handler.requireAuth(handler.StreamMessage)).Methods("POST")

	// Chat instructions routes
	router.HandleFunc("/api/chat/instructions", handler.requireAuth(handler.GetInstructions)).Methods("GET")
	router.HandleFunc("/api/chat/instructions", handler.requireAuth(handler.UpdateInstructions)).Methods("PUT")

	// Chat usage routes
	router.HandleFunc("/api/chat/usage", handler.requireAuth(handler.GetUsage)).Methods("GET")
}

// requireAuth is middleware that ensures a user is authenticated.
// It extracts the user ID from the request context and passes it to the handler.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The JWT middleware in main.go should have already validated the token
		// and set the user ID in the request context
		userID := r.Context().Value("user_id")
		if userID == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Call the next handler with the authenticated context
		next(w, r)
	}
}

// getUserID extracts the user ID from the request context.
func (h *Handler) getUserID(r *http.Request) (int, error) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		return 0, http.ErrNoCookie
	}

	// Handle both int and float64 types (JSON numbers are float64)
	switch v := userID.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		// Try to parse as string
		return strconv.Atoi(v)
	default:
		return 0, http.ErrNoCookie
	}
}

// CreateConversation creates a new chat conversation.
func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title        *string `json:"title"`
		Model        *string `json:"model"`
		PrimaryCardID *int    `json:"primary_card_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create conversation
	conversationID := ""
	query := `
		INSERT INTO chat_conversations (id, user_id, title, model, primary_card_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	err = h.server.DB.QueryRow(query, userID, req.Title, req.Model, req.PrimaryCardID).Scan(&conversationID)
	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	// Fetch the created conversation
	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// ListConversations lists all conversations for the authenticated user.
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, title, model, primary_card_id, created_at, updated_at
		FROM chat_conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := h.server.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve conversations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var conversations []models.ChatConversation
	for rows.Next() {
		var c models.ChatConversation
		var title sql.NullString
		var primaryCardID sql.NullInt64
		err := rows.Scan(&c.ID, &title, &c.Model, &primaryCardID, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}
		if title.Valid {
			c.Title = &title.String
		}
		if primaryCardID.Valid {
			val := int(primaryCardID.Int64)
			c.PrimaryCardID = &val
		}
		c.UserID = userID
		conversations = append(conversations, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

// GetConversation retrieves a single conversation by ID.
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// UpdateConversation updates a conversation's title or primary card.
func (h *Handler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	var req struct {
		Title        *string `json:"title"`
		PrimaryCardID *int    `json:"primary_card_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE chat_conversations
		SET title = COALESCE($1, title),
		    primary_card_id = COALESCE($2, primary_card_id),
		    updated_at = NOW()
		WHERE id = $3 AND user_id = $4
	`
	_, err = h.server.DB.Exec(query, req.Title, req.PrimaryCardID, conversationID, userID)
	if err != nil {
		http.Error(w, "Failed to update conversation", http.StatusInternalServerError)
		return
	}

	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// DeleteConversation deletes a conversation and all its messages.
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	// Delete in a transaction
	tx, err := h.server.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete messages
	_, err = tx.Exec("DELETE FROM chat_messages WHERE conversation_id = $1", conversationID)
	if err != nil {
		http.Error(w, "Failed to delete messages", http.StatusInternalServerError)
		return
	}

	// Delete conversation
	result, err := tx.Exec("DELETE FROM chat_conversations WHERE id = $1 AND user_id = $2", conversationID, userID)
	if err != nil {
		http.Error(w, "Failed to delete conversation", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SendMessage sends a new message in a conversation (non-streaming).
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	var req struct {
		Content      string  `json:"content"`
		ModelOverride *string `json:"model_override"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get conversation
	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Save user message
	userMessageID := ""
	query := `
		INSERT INTO chat_messages (id, conversation_id, role, content, status, created_at)
		VALUES (gen_random_uuid(), $1, 'user', $2, 'completed', NOW())
		RETURNING id
	`
	err = h.server.DB.QueryRow(query, conversationID, req.Content).Scan(&userMessageID)
	if err != nil {
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Create assistant message placeholder
	assistantMessageID := ""
	query = `
		INSERT INTO chat_messages (id, conversation_id, role, content, status, created_at)
		VALUES (gen_random_uuid(), $1, 'assistant', NULL, 'pending', NOW())
		RETURNING id
	`
	err = h.server.DB.QueryRow(query, conversationID).Scan(&assistantMessageID)
	if err != nil {
		http.Error(w, "Failed to create assistant message", http.StatusInternalServerError)
		return
	}

	// Process asynchronously
	go func() {
		ctx := context.Background()
		_ = h.chatService.ProcessAssistantResponse(
			ctx,
			userID,
			conversation,
			assistantMessageID,
			req.ModelOverride,
			h.getConversationMessages,
			h.updateMessage,
		)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_message_id":      userMessageID,
		"assistant_message_id": assistantMessageID,
	})
}

// StreamMessage sends a new message with streaming response.
func (h *Handler) StreamMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	var req struct {
		Content      string  `json:"content"`
		ModelOverride *string `json:"model_override"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get conversation
	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Save user message
	userMessageID := ""
	query := `
		INSERT INTO chat_messages (id, conversation_id, role, content, status, created_at)
		VALUES (gen_random_uuid(), $1, 'user', $2, 'completed', NOW())
		RETURNING id
	`
	err = h.server.DB.QueryRow(query, conversationID, req.Content).Scan(&userMessageID)
	if err != nil {
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Create assistant message placeholder
	assistantMessageID := ""
	query = `
		INSERT INTO chat_messages (id, conversation_id, role, content, status, created_at)
		VALUES (gen_random_uuid(), $1, 'assistant', NULL, 'pending', NOW())
		RETURNING id
	`
	err = h.server.DB.QueryRow(query, conversationID).Scan(&assistantMessageID)
	if err != nil {
		http.Error(w, "Failed to create assistant message", http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create context with timeout for streaming
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Stream the response
	err = h.chatService.StreamAssistantResponse(
		ctx,
		w,
		userID,
		conversation,
		assistantMessageID,
		req.ModelOverride,
		h.getConversationMessages,
		h.updateMessage,
	)

	if err != nil {
		// Error already sent via SSE
		return
	}
}

// GetMessages retrieves all messages in a conversation.
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationID := vars["id"]

	// Verify conversation belongs to user
	conversation, err := h.getConversationByID(conversationID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// Get conversation ownership check
	if conversation.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	messages, err := h.getConversationMessages(conversationID)
	if err != nil {
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// GetInstructions retrieves the user's chat instructions.
func (h *Handler) GetInstructions(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var instructions models.ChatInstructions
	query := `
		SELECT instructions FROM chat_instructions WHERE user_id = $1
	`
	err = h.server.DB.QueryRow(query, userID).Scan(&instructions.Instructions)
	if err == sql.ErrNoRows {
		// No instructions set, return empty
		instructions.Instructions = ""
	} else if err != nil {
		http.Error(w, "Failed to retrieve instructions", http.StatusInternalServerError)
		return
	}

	instructions.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instructions)
}

// UpdateInstructions updates the user's chat instructions.
func (h *Handler) UpdateInstructions(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO chat_instructions (user_id, instructions, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			instructions = EXCLUDED.instructions,
			updated_at = NOW()
	`
	_, err = h.server.DB.Exec(query, userID, req.Instructions)
	if err != nil {
		http.Error(w, "Failed to update instructions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":      userID,
		"instructions": req.Instructions,
	})
}

// GetUsage retrieves the user's chat usage statistics.
func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var messagesToday, messagesThisMonth int
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN 1 ELSE 0 END), 0) as messages_today,
			COALESCE(SUM(CASE WHEN created_at >= DATE_TRUNC('month', CURRENT_DATE) THEN 1 ELSE 0 END), 0) as messages_this_month
		FROM chat_messages
		WHERE conversation_id IN (SELECT id FROM chat_conversations WHERE user_id = $1)
		  AND role = 'user'
	`
	err = h.server.DB.QueryRow(query, userID).Scan(&messagesToday, &messagesThisMonth)
	if err != nil {
		http.Error(w, "Failed to retrieve usage", http.StatusInternalServerError)
		return
	}

	usage := map[string]interface{}{
		"user_id":             userID,
		"messages_today":      messagesToday,
		"messages_this_month": messagesThisMonth,
		"daily_limit":         100,  // TODO: Make this configurable
		"monthly_limit":       3000, // TODO: Make this configurable
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// Helper methods

func (h *Handler) getConversationByID(conversationID string, userID int) (*models.ChatConversation, error) {
	var c models.ChatConversation
	var title sql.NullString
	var primaryCardID sql.NullInt64
	var userIDPtr sql.NullInt64

	query := `
		SELECT id, title, model, primary_card_id, user_id, created_at, updated_at
		FROM chat_conversations
		WHERE id = $1
	`
	err := h.server.DB.QueryRow(query, conversationID).Scan(
		&c.ID, &title, &c.Model, &primaryCardID, &userIDPtr, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if title.Valid {
		c.Title = &title.String
	}
	if primaryCardID.Valid {
		val := int(primaryCardID.Int64)
		c.PrimaryCardID = &val
	}
	if userIDPtr.Valid {
		c.UserID = int(userIDPtr.Int64)
	}

	return &c, nil
}

func (h *Handler) getConversationMessages(conversationID string) ([]models.ChatMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, status, created_at, sequence_number
		FROM chat_messages
		WHERE conversation_id = $1
		ORDER BY sequence_number ASC
	`

	rows, err := h.server.DB.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		var content, toolCalls sql.NullString
		var toolCallID sql.NullString

		err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &content, &toolCalls, &m.Status, &m.CreatedAt, &m.SequenceNumber)
		if err != nil {
			continue
		}

		if content.Valid {
			m.Content = &content.String
		}
		if toolCalls.Valid {
			_ = json.Unmarshal([]byte(toolCalls.String), &m.ToolCalls)
		}
		if toolCallID.Valid {
			m.ToolCallID = &toolCallID.String
		}

		messages = append(messages, m)
	}

	return messages, nil
}

func (h *Handler) updateMessage(messageID string, content *string, toolCallsJSON *string, status string) error {
	query := `
		UPDATE chat_messages
		SET content = COALESCE($1, content),
		    tool_calls = COALESCE($2, tool_calls),
		    status = $3
		WHERE id = $4
	`
	_, err := h.server.DB.Exec(query, content, toolCallsJSON, status, messageID)
	return err
}

func (h *Handler) updateConversationTitle(conversationID, title string) error {
	query := `
		UPDATE chat_conversations
		SET title = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := h.server.DB.Exec(query, title, conversationID)
	return err
}
