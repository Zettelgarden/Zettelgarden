package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/handlers"
	"go-backend/models"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot
type Bot struct {
	api           *tgbotapi.BotAPI
	handler       *handlers.Handler
	allowedUserID int64
	userID        int  // Zettelgarden user ID for the allowed Telegram user
	cancel        context.CancelFunc
}

// NewBot creates a new Telegram bot instance
func NewBot(token string, allowedUserID int64, userID int, handler *handlers.Handler) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = false

	bot := &Bot{
		api:           api,
		handler:       handler,
		allowedUserID: allowedUserID,
		userID:        userID,
	}

	log.Printf("[telegram] Bot authorized as @%s", api.Self.UserName)

	return bot, nil
}

// Start begins polling for updates
func (b *Bot) Start(ctx context.Context) {
	ctx, b.cancel = context.WithCancel(ctx)

	log.Printf("[telegram] Starting bot for user_id=%d", b.allowedUserID)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[telegram] panic in update handler: %v", r)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[telegram] Stopping bot")
				b.api.StopReceivingUpdates()
				return
			case update, ok := <-updates:
				if !ok {
					log.Printf("[telegram] Updates channel closed")
					return
				}

				if update.Message != nil {
					b.handleMessage(ctx, update.Message)
				}
			}
		}
	}()
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	log.Printf("[telegram] Bot stopped")
}

// handleMessage processes an incoming message
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	// Security check: verify sender
	if message.From.ID != b.allowedUserID {
		log.Printf("[telegram] Rejected message from unauthorized user %d", message.From.ID)
		b.sendMessage(ctx, message.Chat.ID, "Unauthorized access.")
		return
	}

	// Ignore empty messages
	if message.Text == "" {
		return
	}

	log.Printf("[telegram] Received message from @%s (%d): %s",
		message.From.UserName, message.From.ID, message.Text)

	// Handle commands
	if strings.HasPrefix(message.Text, "/") {
		b.handleCommand(ctx, message)
		return
	}

	// Regular chat message
	b.handleChatMessage(ctx, message)
}

// sendMessage sends a message to the chat
func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	// Disable markdown parsing to avoid parsing errors from malformed markdown in LLM responses
	// msg.ParseMode = "markdown"

	// Handle Telegram message length limit
	const maxMessageLength = 4096
	if len(text) > maxMessageLength {
		// Split message into chunks
		for i := 0; i < len(text); i += maxMessageLength {
			end := i + maxMessageLength
			if end > len(text) {
				end = len(text)
			}
			chunk := text[i:end]
			msg.Text = chunk
			if _, err := b.api.Send(msg); err != nil {
				log.Printf("[telegram] Error sending message chunk: %v", err)
			}
			// Small delay to avoid rate limiting
			time.Sleep(100 * time.Millisecond)
		}
		return
	}

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("[telegram] Error sending message: %v", err)
	}
}

// handleChatMessage processes a regular chat message
func (b *Bot) handleChatMessage(ctx context.Context, message *tgbotapi.Message) {
	// Send typing indicator
	action := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	if _, err := b.api.Request(action); err != nil {
		log.Printf("[telegram] Error sending typing action: %v", err)
	}

	// Get or create Telegram conversation for this user
	conversation, err := b.getOrCreateConversation()
	if err != nil {
		log.Printf("[telegram] Error getting conversation: %v", err)
		b.sendMessage(ctx, message.Chat.ID, "Error: Unable to access chat conversation.")
		return
	}

	// Save user message to conversation
	userMessage, err := b.handler.SaveChatMessage(
		conversation.ID,
		"user",
		&message.Text,
		nil,
		nil,
		nil,
		"completed",
	)
	if err != nil {
		log.Printf("[telegram] Error saving user message: %v", err)
		b.sendMessage(ctx, message.Chat.ID, "Error: Failed to save message.")
		return
	}

	log.Printf("[telegram] Saved user message %s to conversation %s", userMessage.ID, conversation.ID)

	// Create pending assistant message
	assistantMessage, err := b.handler.SaveChatMessage(
		conversation.ID,
		"assistant",
		nil,
		nil,
		nil,
		nil,
		"pending",
	)
	if err != nil {
		log.Printf("[telegram] Error creating assistant message: %v", err)
		b.sendMessage(ctx, message.Chat.ID, "Error: Failed to create response.")
		return
	}

	log.Printf("[telegram] Created pending assistant message %s", assistantMessage.ID)

	// Send immediate response and process async
	go b.processAssistantResponse(ctx, message.Chat.ID, conversation, assistantMessage.ID)
}

// getOrCreateConversation gets the latest Telegram conversation or creates a new one
func (b *Bot) getOrCreateConversation() (*models.ChatConversation, error) {
	// Try to get latest conversation for this user
	conversations, err := b.handler.GetUserConversations(b.userID, nil)
	if err == nil && len(conversations) > 0 {
		latest := conversations[0]

		// Check if conversation is stale (older than 2 hours)
		// This prevents issues with LLM providers when reusing very old conversation contexts
		staleThreshold := time.Now().Add(-2 * time.Hour)
		if latest.UpdatedAt.After(staleThreshold) {
			// Conversation is recent, reuse it
			return &models.ChatConversation{
				ID:        latest.ID,
				UserID:    b.userID,
				Model:     latest.Model,
				Title:     latest.Title,
				UpdatedAt: latest.UpdatedAt,
			}, nil
		}
		// Conversation is stale, fall through to create new one
		log.Printf("[telegram] Latest conversation is stale (last updated %v), creating new one", latest.UpdatedAt)
	}

	// Create new conversation
	title := fmt.Sprintf("Telegram Chat %s", time.Now().Format("2006-01-02 15:04"))
	conversation, err := b.handler.CreateConversation(
		b.userID,
		&title,
		"google/gemini-2.5-flash", // Default model
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	log.Printf("[telegram] Created new conversation %s", conversation.ID)
	return conversation, nil
}

// processAssistantResponse processes the assistant response asynchronously
func (b *Bot) processAssistantResponse(ctx context.Context, chatID int64, conversation *models.ChatConversation, messageID string) {
	// Process using ChatService
	if b.handler.ChatService == nil {
		log.Printf("[telegram] ERROR: ChatService is nil!")
		b.sendMessage(ctx, chatID, "Error: Chat service not available.")
		return
	}
	b.handler.ChatService.ProcessAssistantResponse(
		ctx,
		b.userID,
		conversation,
		messageID,
		nil,
		b.handler.GetConversationMessages,
		b.handler.UpdateMessageStatus,
	)

	// Wait for processing to complete, then send response
	// Poll for message completion
	for i := 0; i < 60; i++ { // Max 60 seconds
		// Check for context cancellation (bot shutdown)
		select {
		case <-ctx.Done():
			log.Printf("[telegram] Bot shutting down, stopping message processing for message %s", messageID)
			return
		default:
			// Continue processing
		}

		message, err := b.handler.GetChatMessage(messageID)
		if err != nil {
			log.Printf("[telegram] Error getting message: %v", err)
			b.sendMessage(ctx, chatID, "Error: Failed to get response.")
			return
		}

		if message.Status == "completed" && message.Content != nil {
			// Build response with tool usage summary
			response := b.buildResponseWithToolSummary(conversation.ID, messageID, *message.Content)
			b.sendMessage(ctx, chatID, response)
			return
		}

		if message.Status == "failed" {
			b.sendMessage(ctx, chatID, "Sorry, something went wrong processing your request.")
			return
		}

		time.Sleep(1 * time.Second)
	}

	b.sendMessage(ctx, chatID, "Request is taking longer than expected. Please check the web UI for the response.")
}

// buildResponseWithToolSummary builds a response with tool usage summary appended
func (b *Bot) buildResponseWithToolSummary(conversationID, assistantMessageID, content string) string {
	// Get all messages in this conversation to find tool calls and results
	messages, err := b.handler.GetConversationMessages(conversationID)
	if err != nil {
		log.Printf("[telegram] Error getting conversation messages: %v", err)
		return content
	}

	// Find assistant message and collect tool calls/results
	var toolCalls []string
	var toolErrors []string
	foundAssistant := false

	for _, msg := range messages {
		// Stop processing once we've passed the assistant message
		if msg.ID == assistantMessageID {
			if msg.Role == "assistant" {
				foundAssistant = true
				// Collect tool calls from assistant message
				for _, tc := range msg.ToolCalls {
					toolName := tc.Function.Name
					toolCalls = append(toolCalls, toolName)
				}
			}
			continue
		}

		// Only process tool messages that come after the assistant message
		if !foundAssistant {
			continue
		}

		// Process tool result messages
		if msg.Role == "tool" && msg.Content != nil {
			// Parse the tool result to check for errors
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(*msg.Content), &result); err == nil {
				if hasErr, ok := result["error"].(bool); ok && hasErr {
					// Extract error message if available
					if errMsg, ok := result["message"].(string); ok {
						toolErrors = append(toolErrors, errMsg)
					} else {
						toolErrors = append(toolErrors, "unknown error")
					}
				}
			}
		}

		// Stop if we hit another user or assistant message
		if msg.Role == "user" || (msg.Role == "assistant" && msg.ID != assistantMessageID) {
			break
		}
	}

	// Build the response with tool summary (no markdown formatting)
	response := content

	if len(toolCalls) > 0 {
		response += "\n\n---\nTools used: " + formatToolList(toolCalls)
	}

	if len(toolErrors) > 0 {
		response += "\n⚠️ Some tools had errors:"
		for _, errMsg := range toolErrors {
			response += "\n  - " + errMsg
		}
	}

	return response
}

// formatToolList formats a list of tool names for display
func formatToolList(tools []string) string {
	if len(tools) == 0 {
		return "none"
	}

	// Create a simple formatted list
	var result string
	for i, tool := range tools {
		if i > 0 {
			result += ", "
		}
		// Format tool name: replace underscores with spaces, capitalize
		formatted := strings.ReplaceAll(tool, "_", " ")
		if len(formatted) > 0 {
			formatted = strings.ToUpper(formatted[:1]) + formatted[1:]
		}
		result += formatted
	}
	return result
}
