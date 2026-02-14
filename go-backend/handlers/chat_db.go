package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateConversation creates a new chat conversation
func (s *Handler) CreateConversation(userID int, title *string, model string, systemPrompt *string, primaryCardID *int) (*models.ChatConversation, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO chat_conversations (id, user_id, title, model, system_prompt, primary_card_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, user_id, title, model, system_prompt, primary_card_id, starred, created_at, updated_at
	`

	var conversation models.ChatConversation
	err := s.DB.QueryRow(query, id, userID, title, model, systemPrompt, primaryCardID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Model,
		&conversation.SystemPrompt,
		&conversation.PrimaryCardID,
		&conversation.Starred,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	return &conversation, err
}

// GetUserConversations gets all conversations for a user, optionally filtered by primary_card_id
func (s *Handler) GetUserConversations(userID int, primaryCardID *int) ([]ConversationResponse, error) {
	query := `
		SELECT c.id, c.user_id, c.title, c.model, c.system_prompt, c.primary_card_id, c.starred,
		       c.created_at, c.updated_at, COUNT(m.id) as message_count
		FROM chat_conversations c
		LEFT JOIN chat_messages m ON c.id = m.conversation_id
		WHERE c.user_id = $1
	`

	args := []interface{}{userID}

	// Add optional primary_card_id filter
	if primaryCardID != nil {
		query += " AND c.primary_card_id = $2"
		args = append(args, *primaryCardID)
	}

	query += `
		GROUP BY c.id, c.user_id, c.title, c.model, c.system_prompt, c.primary_card_id, c.starred, c.created_at, c.updated_at
		ORDER BY c.updated_at DESC
	`

	rows, err := s.DB.Query(query, args...)
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
			&conv.PrimaryCardID,
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
		SELECT id, user_id, title, model, system_prompt, primary_card_id, starred, created_at, updated_at
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
		&conversation.PrimaryCardID,
		&conversation.Starred,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	return &conversation, err
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

// UpdateConversationModel updates the model of a conversation
func (s *Handler) UpdateConversationModel(conversationID string, model string) error {
	query := `UPDATE chat_conversations SET model = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.DB.Exec(query, model, conversationID)
	return err
}

// GetConversationMessages gets all messages for a conversation
func (s *Handler) GetConversationMessages(conversationID string) ([]models.ChatMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, status, created_at
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
			&msg.Status,
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

// GetConversationMessagesUpTo gets all messages for a conversation up to (but not including) a specific message
func (s *Handler) GetConversationMessagesUpTo(conversationID, messageID string) ([]models.ChatMessage, error) {
	// First get the sequence number of the target message
	var targetSequence int
	err := s.DB.QueryRow("SELECT sequence_number FROM chat_messages WHERE id = $1", messageID).Scan(&targetSequence)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, status, created_at
		FROM chat_messages
		WHERE conversation_id = $1 AND sequence_number < $2
		ORDER BY sequence_number ASC
	`

	rows, err := s.DB.Query(query, conversationID, targetSequence)
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
			&msg.Status,
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
func (s *Handler) SaveChatMessage(conversationID, role string, content *string, toolCalls []models.ChatToolCall, toolCallID *string, referencedCards []string, status string) (*models.ChatMessage, error) {
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
		INSERT INTO chat_messages (id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, status, created_at
	`

	var message models.ChatMessage
	var returnedToolCalls *string
	var returnedReferencedCards *string

	err = s.DB.QueryRow(query, id, conversationID, role, content, toolCallsJSON, toolCallID, sequenceNumber, referencedCardsJSON, status).Scan(
		&message.ID,
		&message.ConversationID,
		&message.Role,
		&message.Content,
		&returnedToolCalls,
		&message.ToolCallID,
		&message.SequenceNumber,
		&returnedReferencedCards,
		&message.Status,
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

// UpdateMessageStatus updates the status of a message (thread-safe for concurrent operations)
func (s *Handler) UpdateMessageStatus(messageID, status string) error {
	// Acquire per-message mutex to prevent race conditions on status updates
	mu := s.getMessageMutex(messageID)
	mu.Lock()
	defer mu.Unlock()
	defer s.cleanupMessageMutex(messageID)

	// Validate status transition
	currentStatus, err := s.getMessageStatusLocked(messageID)
	if err != nil {
		return err
	}

	// Define valid status transitions
	validTransitions := map[string][]string{
		"pending":   {"processing", "failed"},
		"processing": {"completed", "failed"},
	}

	// Check if transition is valid
	if currentStatus != "" {
		validNext, ok := validTransitions[currentStatus]
		if !ok || !containsString(validNext, status) {
			log.Printf("Invalid status transition for message %s: %s -> %s", messageID, currentStatus, status)
			return fmt.Errorf("invalid status transition from %s to %s", currentStatus, status)
		}
	}

	query := `UPDATE chat_messages SET status = $1 WHERE id = $2`
	_, err = s.DB.Exec(query, status, messageID)
	return err
}

// getMessageStatusLocked gets the current status of a message (must be called with message mutex held)
func (s *Handler) getMessageStatusLocked(messageID string) (string, error) {
	var status string
	query := `SELECT status FROM chat_messages WHERE id = $1`
	err := s.DB.QueryRow(query, messageID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Message doesn't exist yet
		}
		return "", err
	}
	return status, nil
}

// containsString checks if a string slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetChatMessage gets a single chat message by ID
func (s *Handler) GetChatMessage(messageID string) (*models.ChatMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, sequence_number, referenced_cards, status, created_at
		FROM chat_messages
		WHERE id = $1
	`

	var msg models.ChatMessage
	var toolCalls *string
	var referencedCards *string

	err := s.DB.QueryRow(query, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&toolCalls,
		&msg.ToolCallID,
		&msg.SequenceNumber,
		&referencedCards,
		&msg.Status,
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

	return &msg, nil
}

// ClearMessageContent clears the content and tool calls of a message
func (s *Handler) ClearMessageContent(messageID string) error {
	query := `UPDATE chat_messages SET content = NULL, tool_calls = NULL WHERE id = $1`
	_, err := s.DB.Exec(query, messageID)
	return err
}

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

// UpdateUserMessage updates the content of a user message
func (s *Handler) UpdateUserMessage(messageID, content string) error {
	query := `UPDATE chat_messages SET content = $1 WHERE id = $2 AND role = 'user'`
	result, err := s.DB.Exec(query, content, messageID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetMessageSequenceNumber gets the sequence number of a message
func (s *Handler) GetMessageSequenceNumber(messageID string) (int, string, error) {
	var seqNum int
	var conversationID string
	query := `SELECT sequence_number, conversation_id FROM chat_messages WHERE id = $1`
	err := s.DB.QueryRow(query, messageID).Scan(&seqNum, &conversationID)
	return seqNum, conversationID, err
}

// DeleteMessagesAfter deletes all messages in a conversation with sequence_number > targetSeq
func (s *Handler) DeleteMessagesAfter(conversationID string, targetSeq int) error {
	query := `DELETE FROM chat_messages WHERE conversation_id = $1 AND sequence_number > $2`
	_, err := s.DB.Exec(query, conversationID, targetSeq)
	return err
}

// GetMessageCreatedAt gets the creation time of a message for edit window validation
func (s *Handler) GetMessageCreatedAt(messageID string) (time.Time, error) {
	var createdAt time.Time
	query := `SELECT created_at FROM chat_messages WHERE id = $1`
	err := s.DB.QueryRow(query, messageID).Scan(&createdAt)
	return createdAt, err
}