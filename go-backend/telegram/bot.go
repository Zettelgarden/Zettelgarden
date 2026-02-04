package telegram

import (
	"context"
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
	msg.ParseMode = "markdown"

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
	if _, err := b.api.Send(action); err != nil {
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
		// Return the most recent conversation
		return &models.ChatConversation{
			ID:        conversations[0].ID,
			UserID:    b.userID,
			Model:     conversations[0].Model,
			Title:     conversations[0].Title,
			UpdatedAt: conversations[0].UpdatedAt,
		}, nil
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
	// Process using existing handler logic
	b.handler.ProcessAssistantResponse(b.userID, *conversation, messageID, nil)

	// Wait for processing to complete, then send response
	// Poll for message completion
	for i := 0; i < 60; i++ { // Max 60 seconds
		message, err := b.handler.GetChatMessage(messageID)
		if err != nil {
			log.Printf("[telegram] Error getting message: %v", err)
			b.sendMessage(ctx, chatID, "Error: Failed to get response.")
			return
		}

		if message.Status == "completed" && message.Content != nil {
			b.sendMessage(ctx, chatID, *message.Content)
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
