package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// SendMessageRequest represents the request to send a message
type SendMessageRequest struct {
	Content         string   `json:"content"`
	ReferencedCards []string `json:"referenced_cards,omitempty"`
	Model           *string  `json:"model,omitempty"`
}

// GetReferencedCards retrieves the referenced cards and formats them for context
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

// StreamMessageRoute sends a message and streams the response using Server-Sent Events
func (s *Handler) StreamMessageRoute(w http.ResponseWriter, r *http.Request) {
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

	// Update conversation model if one is provided in the request
	if req.Model != nil && *req.Model != "" && *req.Model != conversation.Model {
		err := s.UpdateConversationModel(conversationID, *req.Model)
		if err != nil {
			log.Printf("Error updating conversation model: %v", err)
		}
	}

	// Check usage quotas
	if err := s.CheckChatUsageQuota(userID, "messages_per_day"); err != nil {
		http.Error(w, "Daily message limit exceeded", http.StatusTooManyRequests)
		return
	}

	referencedCardContext := s.GetReferencedCards(userID, req.ReferencedCards)
	content := req.Content + referencedCardContext

	// Save user message
	userMessage, err := s.SaveChatMessage(conversationID, "user", &content, nil, nil, req.ReferencedCards, "completed")
	if err != nil {
		log.Printf("Error saving user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Create a pending assistant message
	assistantMessage, err := s.SaveChatMessage(conversationID, "assistant", nil, nil, nil, nil, "processing")
	if err != nil {
		log.Printf("Error saving assistant message: %v", err)
		http.Error(w, "Failed to save assistant message", http.StatusInternalServerError)
		return
	}

	// Update usage quota
	s.IncrementChatUsageQuota(userID, "messages_per_day")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial messages
	initialData, _ := json.Marshal(map[string]interface{}{
		"user_message":      userMessage,
		"assistant_message": assistantMessage,
	})
	fmt.Fprintf(w, "event: messages\ndata: %s\n\n", initialData)
	w.(http.Flusher).Flush()

	// Stream the response
	s.streamAssistantResponse(w, userID, conversation, assistantMessage.ID, req.Model)
}

// SendMessageRoute sends a message and returns immediately, processing async
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

	// Update conversation model if one is provided in the request
	if req.Model != nil && *req.Model != "" && *req.Model != conversation.Model {
		err := s.UpdateConversationModel(conversationID, *req.Model)
		if err != nil {
			log.Printf("Error updating conversation model: %v", err)
			// Don't fail the request, just log the error
		}
	}

	// Check usage quotas
	if err := s.CheckChatUsageQuota(userID, "messages_per_day"); err != nil {
		http.Error(w, "Daily message limit exceeded", http.StatusTooManyRequests)
		return
	}

	referencedCardContext := s.GetReferencedCards(userID, req.ReferencedCards)

	content := req.Content + referencedCardContext
	// Save user message
	userMessage, err := s.SaveChatMessage(conversationID, "user", &content, nil, nil, req.ReferencedCards, "completed")
	if err != nil {
		log.Printf("Error saving user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Note: We'll update chat memory after we get the assistant response

	// Create a pending assistant message
	assistantMessage, err := s.SaveChatMessage(conversationID, "assistant", nil, nil, nil, nil, "pending")
	if err != nil {
		log.Printf("Error saving assistant message: %v", err)
		http.Error(w, "Failed to save assistant message", http.StatusInternalServerError)
		return
	}
	log.Printf("created new message %v %v", conversationID, assistantMessage.ID)

	// Update usage quota
	s.IncrementChatUsageQuota(userID, "messages_per_day")

	// Start async processing
	go s.processAssistantResponse(userID, conversation, assistantMessage.ID, req.Model)

	// Return both messages immediately
	response := []models.ChatMessage{*userMessage, *assistantMessage}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegenerateMessageRoute regenerates a specific assistant message
func (s *Handler) RegenerateMessageRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	conversationID := mux.Vars(r)["id"]
	messageID := mux.Vars(r)["messageId"]

	// Check if user has subscription for chat functionality
	if !s.UserHasSubscription(userID) {
		http.Error(w, "Chat functionality requires a Pro subscription", http.StatusForbidden)
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

	// Verify message exists and is an assistant message
	message, err := s.GetChatMessage(messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Message not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting message: %v", err)
			http.Error(w, "Failed to get message", http.StatusInternalServerError)
		}
		return
	}

	// Only allow regenerating assistant messages
	if message.Role != "assistant" {
		http.Error(w, "Can only regenerate assistant messages", http.StatusBadRequest)
		return
	}

	// Verify message belongs to this conversation
	if message.ConversationID != conversationID {
		http.Error(w, "Message does not belong to this conversation", http.StatusBadRequest)
		return
	}

	// Check usage quotas
	if err := s.CheckChatUsageQuota(userID, "messages_per_day"); err != nil {
		http.Error(w, "Daily message limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Reset the message to pending status
	err = s.UpdateMessageStatus(messageID, "pending")
	if err != nil {
		log.Printf("Error updating message status: %v", err)
		http.Error(w, "Failed to update message status", http.StatusInternalServerError)
		return
	}

	// Clear the message content and tool calls
	err = s.ClearMessageContent(messageID)
	if err != nil {
		log.Printf("Error clearing message content: %v", err)
		http.Error(w, "Failed to clear message content", http.StatusInternalServerError)
		return
	}

	// Update usage quota
	s.IncrementChatUsageQuota(userID, "messages_per_day")

	// Start async regeneration process
	go s.processAssistantResponse(userID, conversation, messageID, nil)

	// Return the updated message
	updatedMessage, err := s.GetChatMessage(messageID)
	if err != nil {
		log.Printf("Error getting updated message: %v", err)
		http.Error(w, "Failed to get updated message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}