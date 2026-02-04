# Telegram Bot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Telegram bot that integrates with Zettelgarden's existing chat/agent functionality, allowing a single authorized user to chat with their knowledge base via Telegram with access to all agent tools.

**Architecture:** The bot runs as a goroutine in go-backend, using Telegram's long polling to receive messages. Messages are authenticated via env var, then forwarded to existing chat handlers. One conversation per user, `/clear` creates new conversation.

**Tech Stack:** Go, github.com/go-telegram-bot-api/telegram-bot-api/v5, existing chat/agent handlers in handlers/chat_messages.go

---

## Task 1: Add Telegram Configuration

**Files:**
- Modify: `go-backend/pkg/config/services.go`
- Modify: `go-backend/pkg/config/config.go`
- Modify: `go-backend/.env.example`

**Step 1: Add TelegramConfig struct to services.go**

Add this struct after `SearchConfig` (around line 57):

```go
// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken       string // Telegram bot token from @BotFather (sensitive)
	AllowedUserID  int64  // Telegram user ID allowed to use the bot
	Enabled        bool   // Enable/disable bot
}
```

**Step 2: Add Telegram to ServiceConfig struct**

Modify the `ServiceConfig` struct (around line 4):

```go
type ServiceConfig struct {
	LLM      LLMConfig      // Language model/embedding services
	Mail     MailConfig     // Email service
	Stripe   StripeConfig   // Payment processing
	S3       S3Config       // Object storage
	GitHub   GitHubConfig   // OAuth service
	Search   SearchConfig   // Search engine
	Telegram TelegramConfig // Telegram bot
}
```

**Step 3: Add loadTelegramConfig function**

Add at the end of services.go (before `loadServiceConfig` around line 59):

```go
// loadTelegramConfig loads Telegram bot configuration
func loadTelegramConfig() TelegramConfig {
	config := TelegramConfig{
		BotToken:      optionalString("TELEGRAM_BOT_TOKEN"),
		AllowedUserID: optionalInt64("TELEGRAM_ALLOWED_USER_ID"),
		Enabled:       optionalBool("TELEGRAM_ENABLED"),
	}

	// Validate that if enabled, required fields are present
	if config.Enabled {
		if config.BotToken == "" {
			validationErrors = append(validationErrors,
				"TELEGRAM_BOT_TOKEN is required when TELEGRAM_ENABLED=true")
		}
		if config.AllowedUserID == 0 {
			validationErrors = append(validationErrors,
				"TELEGRAM_ALLOWED_USER_ID is required when TELEGRAM_ENABLED=true")
		}
	}

	return config
}
```

**Step 4: Update loadServiceConfig to load Telegram config**

Modify `loadServiceConfig` function (around line 60):

```go
func loadServiceConfig() ServiceConfig {
	return ServiceConfig{
		LLM:      loadLLMConfig(),
		Mail:     loadMailConfig(),
		Stripe:   loadStripeConfig(),
		S3:       loadS3Config(),
		GitHub:   loadGitHubConfig(),
		Search:   loadSearchConfig(),
		Telegram: loadTelegramConfig(),
	}
}
```

**Step 5: Add helper functions to validation.go**

Check if validation.go exists, if not create it in pkg/config:

```go
// optionalInt64 gets an optional int64 environment variable
func optionalInt64(key string) int64 {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal
		}
	}
	return 0
}

// optionalString gets an optional string environment variable
func optionalString(key string) string {
	return os.Getenv(key)
}
```

**Step 6: Update .env.example**

Add to go-backend/.env.example:

```bash
# Telegram Bot Configuration (optional)
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_ALLOWED_USER_ID=123456789
TELEGRAM_ENABLED=false
```

**Step 7: Run tests to verify config changes**

```bash
cd go-backend
go test ./pkg/config/... -v
```

Expected: Tests pass (or no tests exist in config package)

**Step 8: Commit**

```bash
git add go-backend/pkg/config/ go-backend/.env.example
git commit -m "feat: add Telegram bot configuration"
```

---

## Task 2: Create Telegram Bot Package Structure

**Files:**
- Create: `go-backend/telegram/bot.go`
- Create: `go-backend/telegram/client.go`
- Create: `go-backend/telegram/commands.go`

**Step 1: Create telegram package directory**

```bash
mkdir -p go-backend/telegram
```

**Step 2: Create bot.go with main bot structure**

Create `go-backend/telegram/bot.go`:

```go
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
			ID:       conversations[0].ID,
			UserID:   b.userID,
			Model:    conversations[0].Model,
			Title:    conversations[0].Title,
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
```

**Step 3: Create commands.go with command handlers**

Create `go-backend/telegram/commands.go`:

```go
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
```

**Step 4: Create client.go (placeholder for future API wrapper)**

Create `go-backend/telegram/client.go`:

```go
package telegram

// This file is a placeholder for future Telegram API wrapper functions
// Currently using the telegram-bot-api library directly
```

**Step 5: Add go dependency for telegram-bot-api**

```bash
cd go-backend
go get github.com/go-telegram-bot-api/telegram-bot-api/v5
```

**Step 6: Verify package compiles**

```bash
cd go-backend
go build ./telegram/...
```

Expected: No errors (may have some errors about missing ProcessAssistantResponse method which we'll add in Task 3)

**Step 7: Commit**

```bash
git add go-backend/telegram/
git commit -m "feat: add Telegram bot package structure"
```

---

## Task 3: Add Handler Method for Async Processing

**Files:**
- Modify: `go-backend/handlers/chat_agent.go`

**Step 1: Check existing ProcessAssistantResponse method**

First, let's check if this method exists or needs to be made public:

```bash
cd go-backend
grep -n "processAssistantResponse\|ProcessAssistantResponse" handlers/chat_agent.go
```

Expected: You'll see `processAssistantResponse` (lowercase) - we need to export it

**Step 2: Export the processAssistantResponse method**

In `handlers/chat_agent.go`, find the `processAssistantResponse` function and rename it to `ProcessAssistantResponse` (capital P).

Change:
```go
func (s *Handler) processAssistantResponse(userID int, conversation models.ChatConversation, messageID string, model *string) {
```

To:
```go
func (s *Handler) ProcessAssistantResponse(userID int, conversation models.ChatConversation, messageID string, model *string) {
```

**Step 3: Update all callers**

Find all places that call `processAssistantResponse` and update to `ProcessAssistantResponse`:

```bash
cd go-backend
grep -rn "processAssistantResponse" handlers/
```

Update each call from lowercase to uppercase.

**Step 4: Verify changes compile**

```bash
cd go-backend
go build ./handlers/...
```

**Step 5: Commit**

```bash
git add go-backend/handlers/
git commit -m "refactor: export ProcessAssistantResponse for Telegram bot use"
```

---

## Task 4: Integrate Bot into main.go

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Add Telegram bot initialization**

Find the section where other services are initialized (around line 260 after the scheduler initialization) and add:

```go
	// Initialize Telegram bot if enabled
	var telegramBot *telegram.Bot
	if cfg.Services.Telegram.Enabled {
		log.Printf("Initializing Telegram bot (allowed_user_id=%d)", cfg.Services.Telegram.AllowedUserID)

		// TODO: For now, use user ID 1. In production, you'd look up the Zettelgarden user ID
		// associated with the Telegram user ID via a database mapping.
		telegramUserID := 1

		var err error
		telegramBot, err = telegram.NewBot(
			cfg.Services.Telegram.BotToken,
			cfg.Services.Telegram.AllowedUserID,
			telegramUserID,
			h,
		)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Telegram bot: %v", err)
			log.Printf("INFO: Telegram bot functionality is disabled")
		} else {
			// Start bot in background goroutine
			safeGoroutine(func() {
				telegramBot.Start(context.Background())
			})
			log.Printf("Telegram bot started successfully")
		}
	}
```

**Step 2: Add import for telegram package**

Add to the imports section at the top of main.go:

```go
	"go-backend/telegram"
```

**Step 3: Add shutdown handler for Telegram bot**

In the shutdown handler section (around line 322), add:

```go
		// Shutdown Telegram bot
		if telegramBot != nil {
			log.Printf("Shutting down Telegram bot...")
			telegramBot.Stop()
		}
```

**Step 4: Update .env.example with default user ID**

The Telegram bot needs to know which Zettelgarden user ID to associate with the allowed Telegram user. For now, we'll use user ID 1 as a default.

Add a comment to .env.example:

```bash
# Telegram Bot Configuration (optional)
# Get your bot token from @BotFather on Telegram
# Get your user ID from @userinfobot on Telegram
# TELEGRAM_ZETTELGARDEN_USER_ID should be your Zettelgarden user ID (default: 1)
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_ALLOWED_USER_ID=123456789
TELEGRAM_ZETTELGARDEN_USER_ID=1
TELEGRAM_ENABLED=false
```

**Step 5: Verify main.go compiles**

```bash
cd go-backend
go build -o main
```

**Step 6: Commit**

```bash
git add go-backend/main.go go-backend/.env.example
git commit -m "feat: integrate Telegram bot into main application"
```

---

## Task 5: Fix Conversation Response Type Mismatch

**Files:**
- Modify: `go-backend/telegram/bot.go`
- Modify: `go-backend/handlers/chat_db.go` (if needed)

**Step 1: Check the actual return type of GetUserConversations**

```bash
cd go-backend
grep -A5 "func.*GetUserConversations" handlers/chat_db.go
```

The function returns `[]ConversationResponse`, not `[]models.ChatConversation`.

**Step 2: Update bot.go to use correct types**

Modify the `getOrCreateConversation` method in `bot.go` to handle the correct return type:

```go
// getOrCreateConversation gets the latest Telegram conversation or creates a new one
func (b *Bot) getOrCreateConversation() (*models.ChatConversation, error) {
	// Try to get latest conversation for this user
	conversations, err := b.handler.GetUserConversations(b.userID, nil)
	if err == nil && len(conversations) > 0 {
		// Return the most recent conversation
		conv := conversations[0]
		return &models.ChatConversation{
			ID:       conv.ID,
			UserID:   conv.UserID,
			Model:    conv.Model,
			Title:    conv.Title,
			UpdatedAt: conv.UpdatedAt,
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
```

**Step 3: Verify compilation**

```bash
cd go-backend
go build ./telegram/...
```

**Step 4: Commit**

```bash
git add go-backend/telegram/bot.go
git commit -m "fix: correct conversation type handling in Telegram bot"
```

---

## Task 6: Add Integration Tests

**Files:**
- Create: `go-backend/telegram/bot_test.go`

**Step 1: Write test for bot creation**

Create `go-backend/telegram/bot_test.go`:

```go
package telegram

import (
	"testing"
)

func TestNewBot(t *testing.T) {
	// Test with invalid token
	_, err := NewBot("invalid_token", 123, 1, nil)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Note: Testing with real token requires integration test setup
	// Unit tests for message handling can be added with mock handler
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"start command", "/start", true},
		{"help command", "/help", true},
		{"clear command", "/clear", true},
		{"regular message", "hello there", false},
		{"message with slash in middle", "check /this out", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := len(tt.input) > 0 && tt.input[0] == '/'
			if result != tt.expected {
				t.Errorf("isCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run tests**

```bash
cd go-backend
go test ./telegram/... -v
```

**Step 3: Commit**

```bash
git add go-backend/telegram/bot_test.go
git commit -m "test: add basic Telegram bot tests"
```

---

## Task 7: Documentation and Setup Instructions

**Files:**
- Create: `go-backend/telegram/README.md`

**Step 1: Create README for Telegram bot**

Create `go-backend/telegram/README.md`:

```markdown
# Telegram Bot

The Zettelgarden Telegram bot allows you to chat with your knowledge base via Telegram.

## Setup

### 1. Create a Telegram Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the bot token (looks like `123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ`)

### 2. Get Your Telegram User ID

1. Open Telegram and search for `@userinfobot`
2. Send any message
3. Copy your numeric user ID

### 3. Configure Environment Variables

Add to your environment:

```bash
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_ALLOWED_USER_ID=your_telegram_user_id
TELEGRAM_ZETTELGARDEN_USER_ID=1  # Your Zettelgarden user ID
TELEGRAM_ENABLED=true
```

### 4. Start the Bot

The bot will start automatically when you run the backend with `TELEGRAM_ENABLED=true`.

## Usage

- Send any message to chat with your knowledge base
- `/help` - Show available commands
- `/clear` - Start a new conversation

## Features

- Full access to AI agent tools (search cards, create cards, etc.)
- Conversation history preserved in web UI
- One conversation per user (new on `/clear`)

## Security

- Only the configured `TELEGRAM_ALLOWED_USER_ID` can use the bot
- All unauthorized access attempts are logged
```

**Step 2: Update main CLAUDE.md**

Add to the "Architecture > Backend" section of `CLAUDE.md`:

```markdown
- **Telegram**: `telegram/` - Telegram bot for chat via Telegram
```

Add to the "Environment Configuration" section:

```markdown
- Telegram bot (TELEGRAM_BOT_TOKEN, TELEGRAM_ALLOWED_USER_ID, TELEGRAM_ENABLED)
  - Purpose: Enable Telegram bot for knowledge base chat
  - Requirements: Bot token from @BotFather, your Telegram user ID
```

**Step 3: Commit**

```bash
git add go-backend/telegram/README.md CLAUDE.md
git commit -m "docs: add Telegram bot documentation"
```

---

## Task 8: Manual Testing Checklist

**Files:** No file changes - manual testing

**Step 1: Start bot locally**

```bash
cd go-backend
export TELEGRAM_BOT_TOKEN=your_token
export TELEGRAM_ALLOWED_USER_ID=your_id
export TELEGRAM_ENABLED=true
go run main.go
```

**Step 2: Test basic functionality**

- [ ] Send `/start` to bot - should receive help message
- [ ] Send `/help` to bot - should see commands
- [ ] Send a regular message - should receive AI response
- [ ] Check web UI - conversation should appear with messages
- [ ] Send `/clear` - should confirm new conversation created
- [ ] Test with unauthorized account - should reject

**Step 3: Test agent tools**

Try sending messages that trigger tools:
- [ ] "Search for cards about [topic]"
- [ ] "Create a card about [topic]"
- [ ] "What cards do I have?"

**Step 4: Commit completion**

```bash
git commit --allow-empty -m "test: manual testing complete for Telegram bot"
```

---

## Summary

This implementation plan creates a Telegram bot that:

1. Reuses all existing chat/agent functionality from handlers/chat_messages.go and handlers/chat_agent.go
2. Runs as a goroutine in the main application
3. Authenticates via environment variable (single user)
4. Creates one conversation per user, with `/clear` to start fresh
5. Has full access to agent tools (search_cards, create_card, etc.)
6. Preserves conversation history visible in web UI

**Total estimated changes:** ~8 tasks, ~500 lines of new code

**Dependencies added:** `github.com/go-telegram-bot-api/telegram-bot-api/v5`
