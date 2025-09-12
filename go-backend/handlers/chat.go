package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/llms"
	"go-backend/models"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type CreateChatCompletionRequest struct {
	ConversationID *string `json:"conversation_id"`
	Message        string  `json:"message"`
	CardPKs        []int   `json:"card_pks"`
}

func (h *Handler) CreateChatCompletion(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req CreateChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conversationID := ""
	if req.ConversationID != nil {
		conversationID = *req.ConversationID
	} else {
		// Create a new conversation
		err := h.DB.QueryRow(`
			INSERT INTO chat_conversations (user_id, title, model)
			VALUES ($1, $2, $3)
			RETURNING id
		`, userID, "New Conversation", models.MODEL).Scan(&conversationID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Fetch card content
	var contextBuilder strings.Builder
	if len(req.CardPKs) > 0 {
		args := make([]interface{}, len(req.CardPKs)+1)
		for i, pk := range req.CardPKs {
			args[i] = pk
		}
		args[len(req.CardPKs)] = userID

		query := `SELECT content FROM cards WHERE id IN (`
		for i := 0; i < len(req.CardPKs); i++ {
			if i > 0 {
				query += ","
			}
			query += fmt.Sprintf("$%d", i+1)
		}
		query += `) AND user_id = $` + fmt.Sprintf("%d", len(req.CardPKs)+1)

		rows, err := h.DB.Query(query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var content string
			if err := rows.Scan(&content); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			contextBuilder.WriteString(content)
			contextBuilder.WriteString("\n\n")
		}
	}

	// Save user message
	_, err := h.DB.Exec(`
		INSERT INTO chat_completions (user_id, conversation_id, role, content, card_chunks)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, conversationID, "user", req.Message, pq.Array(req.CardPKs))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get message history
	rows, err := h.DB.Query(`
		SELECT role, content
		FROM chat_completions
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, m)
	}

	llmClient := llms.NewDefaultClient(h.DB, userID)
	assistantResponse, err := llms.CreateChatCompletion(llmClient, r.Context(), messages, contextBuilder.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save assistant message
	_, err = h.DB.Exec(`
		INSERT INTO chat_completions (user_id, conversation_id, role, content)
		VALUES ($1, $2, $3, $4)
	`, userID, conversationID, "assistant", assistantResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"message": assistantResponse}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := h.DB.Query(`
		SELECT id, title, created_at, model, message_count, updated_at
		FROM chat_conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.Model, &c.MessageCount, &c.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		conversations = append(conversations, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

func (h *Handler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	conversationID := vars["id"]

	rows, err := h.DB.Query(`
		SELECT id, conversation_id, role, content, created_at, card_chunks
		FROM chat_completions
		WHERE user_id = $1 AND conversation_id = $2
		ORDER BY created_at ASC
	`, userID, conversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt, &m.CardChunks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
