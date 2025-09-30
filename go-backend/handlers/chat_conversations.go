package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CreateConversationRequest represents the request to create a new conversation
type CreateConversationRequest struct {
	Title         *string `json:"title"`
	Model         string  `json:"model"`
	SystemPrompt  *string `json:"system_prompt"`
	PrimaryCardID *int    `json:"primary_card_id"`
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

// ConversationStatusResponse represents the status of a conversation's processing
type ConversationStatusResponse struct {
	ConversationID string `json:"conversation_id"`
	HasPending     bool   `json:"has_pending"`
	HasProcessing  bool   `json:"has_processing"`
	HasFailed      bool   `json:"has_failed"`
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
	conversation, err := s.CreateConversation(userID, req.Title, req.Model, req.SystemPrompt, req.PrimaryCardID)
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

	// Check for optional primary_card_id filter
	primaryCardIDStr := r.URL.Query().Get("primary_card_id")
	var primaryCardID *int
	if primaryCardIDStr != "" {
		cardID, err := strconv.Atoi(primaryCardIDStr)
		if err != nil {
			http.Error(w, "Invalid primary_card_id parameter", http.StatusBadRequest)
			return
		}
		primaryCardID = &cardID
	}

	conversations, err := s.GetUserConversations(userID, primaryCardID)
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

// GetConversationStatusRoute gets the processing status of a conversation
func (s *Handler) GetConversationStatusRoute(w http.ResponseWriter, r *http.Request) {
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

	// Check for pending, processing, or failed messages
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending_count,
			COALESCE(SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END), 0) as processing_count,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed_count
		FROM chat_messages
		WHERE conversation_id = $1
	`

	var pendingCount, processingCount, failedCount int
	err = s.DB.QueryRow(query, conversationID).Scan(&pendingCount, &processingCount, &failedCount)
	if err != nil {
		log.Printf("Error getting conversation status: %v", err)
		http.Error(w, "Failed to get conversation status", http.StatusInternalServerError)
		return
	}

	status := ConversationStatusResponse{
		ConversationID: conversationID,
		HasPending:     pendingCount > 0,
		HasProcessing:  processingCount > 0,
		HasFailed:      failedCount > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}