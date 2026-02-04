package telegram

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCommand processes bot commands
func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	command := message.Text

	log.Printf("[telegram] Handling command: %s", command)

	switch command {
	case "/start", "/help":
		b.sendHelp(ctx, message.Chat.ID)
	case "/clear":
		b.handleClear(ctx, message.Chat.ID)
	default:
		b.sendMessage(ctx, message.Chat.ID, fmt.Sprintf("Unknown command: %s\n\nType /help for available commands.", command))
	}
}

// sendHelp sends the help message
func (b *Bot) sendHelp(ctx context.Context, chatID int64) {
	helpText := `*Zettelgarden Telegram Bot*

Commands:
/clear - Start a new conversation

Just send me a message to chat with your knowledge base! The AI has access to your cards and can help you search, create, and organize your knowledge.`
	b.sendMessage(ctx, chatID, helpText)
}

// handleClear creates a new conversation (clearing context)
func (b *Bot) handleClear(ctx context.Context, chatID int64) {
	title := fmt.Sprintf("Telegram Chat %s", time.Now().Format("2006-01-02 15:04"))
	conversation, err := b.handler.CreateConversation(
		b.userID,
		&title,
		"google/gemini-2.5-flash",
		nil,
		nil,
	)
	if err != nil {
		log.Printf("[telegram] Error creating new conversation: %v", err)
		b.sendMessage(ctx, chatID, "Error: Failed to create new conversation.")
		return
	}

	log.Printf("[telegram] Created new conversation %s via /clear", conversation.ID)
	b.sendMessage(ctx, chatID, fmt.Sprintf("Started a new conversation (%s)", conversation.ID))
}
